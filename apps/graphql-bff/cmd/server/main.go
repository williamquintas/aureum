package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/kelseyhightower/envconfig"
	"github.com/redis/go-redis/v9"
	"github.com/vektah/gqlparser/v2/ast"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/aureum/graphql-bff/graph"
	"github.com/aureum/graphql-bff/internal/infrastructure/cache"
	"github.com/aureum/graphql-bff/internal/infrastructure/clients"
	"github.com/aureum/graphql-bff/internal/infrastructure/featureflag"
	"github.com/aureum/graphql-bff/internal/infrastructure/idempotency"
	"github.com/aureum/pkg/telemetry"
)

type Config struct {
	Port              string `envconfig:"PORT" default:"8082"`
	TransactionSvc    string `envconfig:"TRANSACTION_SVC" default:"localhost:50054"`
	IdentitySvc       string `envconfig:"IDENTITY_SVC" default:"localhost:50053"`
	BudgetSvc         string `envconfig:"BUDGET_SVC" default:"localhost:50055"`
	CreditCardSvc     string `envconfig:"CREDIT_CARD_SVC" default:"localhost:50056"`
	DebtSvc           string `envconfig:"DEBT_SVC" default:"localhost:50057"`
	InvestmentSvc     string `envconfig:"INVESTMENT_SVC" default:"localhost:50058"`
	RedisAddr         string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	RedisPassword     string `envconfig:"REDIS_PASSWORD" default:""`
	RedisDB           string `envconfig:"REDIS_DB" default:"0"`
	UnleashURL        string `envconfig:"UNLEASH_URL" default:""`
	UnleashAPIToken   string `envconfig:"UNLEASH_API_TOKEN" default:""`
	PlaygroundEnabled bool   `envconfig:"PLAYGROUND_ENABLED" default:"true"`
	MetricsPort       string `envconfig:"METRICS_PORT" default:"9095"`
}

func main() {
	os.Exit(run())
}

func run() int {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Error("failed to load config", "error", err)
		return 1
	}

	if err := telemetry.InitOTEL("graphql-bff", "1.0.0"); err != nil {
		log.Error("failed to init telemetry", "error", err)
		return 1
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := telemetry.ShutdownOTEL(shutdownCtx); err != nil {
			log.Error("failed to shutdown telemetry", "error", err)
		}
	}()

	txConn, err := grpc.Dial(cfg.TransactionSvc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error("failed to connect to transaction-svc", "error", err)
		return 1
	}
	defer txConn.Close()

	idConn, err := grpc.Dial(cfg.IdentitySvc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error("failed to connect to identity-svc", "error", err)
		return 1
	}
	defer idConn.Close()

	bgtConn, err := grpc.Dial(cfg.BudgetSvc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error("failed to connect to budget-svc", "error", err)
		return 1
	}
	defer bgtConn.Close()

	cccConn, err := grpc.Dial(cfg.CreditCardSvc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error("failed to connect to creditcard-svc", "error", err)
		return 1
	}
	defer cccConn.Close()

	dbtConn, err := grpc.Dial(cfg.DebtSvc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error("failed to connect to debt-svc", "error", err)
		return 1
	}
	defer dbtConn.Close()

	invConn, err := grpc.Dial(cfg.InvestmentSvc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Error("failed to connect to investment-svc", "error", err)
		return 1
	}
	defer invConn.Close()

	txClient := clients.NewTransactionServiceClient(txConn)
	idClient := clients.NewIdentityServiceClient(idConn)
	bgtClient := clients.NewBudgetServiceClient(bgtConn)
	cccClient := clients.NewCreditCardServiceClient(cccConn)
	dbtClient := clients.NewDebtServiceClient(dbtConn)
	invClient := clients.NewInvestmentServiceClient(invConn)

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       func() int { n, _ := strconv.Atoi(cfg.RedisDB); return n }(),
	})
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Error("failed to close redis", "error", err)
		}
	}()

	cacheStore, err := cache.NewRedisCache(cfg.RedisAddr, cfg.RedisPassword, func() int { n, _ := strconv.Atoi(cfg.RedisDB); return n }())
	if err != nil {
		log.Error("failed to init cache", "error", err)
		return 1
	}
	_ = cacheStore

	idempStore := idempotency.NewStore(rdb)
	_ = idempStore

	var ffClient *featureflag.Client
	if cfg.UnleashURL != "" {
		var ffErr error
		ffClient, ffErr = featureflag.NewClient(cfg.UnleashURL, "graphql-bff", cfg.UnleashAPIToken)
		if ffErr != nil {
			log.Error("failed to init feature flags", "error", ffErr)
		}
	}

	resolver := graph.NewResolver(txClient, idClient, bgtClient, cccClient, dbtClient, invClient, cacheStore, ffClient)

	srv := handler.New(
		graph.NewExecutableSchema(
			graph.Config{
				Resolvers: resolver,
				Directives: graph.DirectiveRoot{
					Auth:        graph.AuthDirective(resolver.IDClient),
					FeatureFlag: graph.FeatureFlagDirective(resolver.FFClient),
				},
			},
		),
	)

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	srv.Use(extension.Introspection{})

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(30 * time.Second))
	r.Use(corsMiddleware)
	r.Use(telemetry.HTTPMiddleware("graphql-bff"))
	r.Use(graph.IdempotencyMiddleware)

	r.Handle("/graphql", srv)

	if cfg.PlaygroundEnabled {
		r.Handle("/playground", playground.Handler("GraphQL Playground", "/graphql"))
	}

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Info("graphql-bff listening", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "error", err)
		}
	}()

	metricsMux := http.NewServeMux()
	metricsMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "# metrics endpoint ready")
	})
	metricsMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	metricsServer := &http.Server{
		Addr:    ":" + cfg.MetricsPort,
		Handler: metricsMux,
	}
	go func() {
		log.Info("metrics HTTP server listening", "port", cfg.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("metrics server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info("shutting down", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("http server forced shutdown", "error", err)
	}
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error("metrics server forced shutdown", "error", err)
	}

	log.Info("server stopped")
	return 0
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
