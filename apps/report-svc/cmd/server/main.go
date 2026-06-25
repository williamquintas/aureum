package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aureum/pkg/db"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/aureum/pkg/cache"
	ff "github.com/aureum/pkg/featureflag"
	"github.com/aureum/pkg/kafka"
	"github.com/aureum/pkg/telemetry"

	reportv1 "github.com/aureum/proto/gen/report/reportv1"
	"github.com/aureum/report-svc/internal/application"
	"github.com/aureum/report-svc/internal/infrastructure/api"
	"github.com/aureum/report-svc/internal/infrastructure/messaging"
	"github.com/aureum/report-svc/internal/infrastructure/persistence"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := loadConfig()

	if err := telemetry.InitOTEL("report-svc", "1.0.0"); err != nil {
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

	dbPool, err := db.NewPostgresPool(cfg.DatabaseURL, 25)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		return 1
	}
	defer dbPool.Close()

	if err := db.RunMigrations(cfg.DatabaseURL, "migrations"); err != nil {
		log.Error("failed to run migrations", "error", err)
		return 1
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("failed to connect to redis", "error", err)
		return 1
	}
	defer func() { _ = rdb.Close() }()

	redisCache, err := cache.NewRedisCache(cfg.RedisURL, "", 0)
	if err != nil {
		log.Error("failed to create redis cache", "error", err)
		return 1
	}
	defer func() { _ = redisCache.Close() }()

	monthlyRepo := persistence.NewMonthlySummaryRepo(dbPool)
	categoryRepo := persistence.NewCategorySummaryRepo(dbPool)
	budgetRepo := persistence.NewBudgetVsActualRepo(dbPool)
	portfolioRepo := persistence.NewPortfolioSnapshotRepo(dbPool)
	debtRepo := persistence.NewDebtSummaryRepo(dbPool)
	ccRepo := persistence.NewCreditCardSummaryRepo(dbPool)

	var flagClient application.FeatureFlag
	if cfg.UnleashURL != "" && cfg.UnleashToken != "" {
		uc, err := ff.NewClient(cfg.UnleashURL, "report-svc", cfg.UnleashToken)
		if err != nil {
			log.Error("failed to create unleash client", "error", err)
			return 1
		}
		defer func() { _ = uc.Close() }()
		flagClient = &unleashFlag{client: uc}
	} else {
		flags := strings.Split(cfg.EnabledFlags, ",")
		flagClient = &envFlag{flags: flags}
	}

	// ── Create projectors for event consumption ─────────────────────────
	monthlyProj := application.NewMonthlySummaryProjector(monthlyRepo)
	categoryProj := application.NewCategorySummaryProjector(categoryRepo)
	budgetProj := application.NewBudgetVsActualProjector(budgetRepo)
	portfolioProj := application.NewPortfolioSnapshotProjector(portfolioRepo)
	debtProj := application.NewDebtSummaryProjector(debtRepo)

	eventHandler := messaging.NewEventHandler(monthlyProj, categoryProj, budgetProj, portfolioProj, debtProj)

	// ── Start Kafka consumers ──────────────────────────────────────────
	// report-svc consumes domain events from upstream services to project
	// read models. Each event type is published to a separate topic.
	type topicGroup struct {
		topic   string
		groupID string
	}
	consumerTopics := []topicGroup{
		{"transaction-events", "report-svc-transaction"},
		{"budget-events", "report-svc-budget"},
		{"debt-events", "report-svc-debt"},
		{"investment-events", "report-svc-investment"},
	}

	for _, ct := range consumerTopics {
		cg, err := kafka.NewConsumerGroup(cfg.KafkaBrokers, ct.groupID, []string{ct.topic})
		if err != nil {
			log.Error("failed to create kafka consumer", "topic", ct.topic, "error", err)
			return 1
		}
		defer func(c *kafka.ConsumerGroup) { _ = c.Close() }(cg)

		adapter := messaging.NewConsumerAdapter(cg, eventHandler)

		go func(topic string, a *messaging.ConsumerAdapter) {
			log.Info("starting kafka consumer", "topic", topic)
			if err := a.Start(ctx); err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Error("kafka consumer error", "topic", topic, "error", err)
				}
			}
			log.Info("kafka consumer stopped", "topic", topic)
		}(ct.topic, adapter)
	}

	svc := application.NewService(
		monthlyRepo, categoryRepo, budgetRepo, portfolioRepo, debtRepo, ccRepo,
		redisCache, flagClient,
	)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor),
		telemetry.GRPCUnaryInterceptor(),
	)

	handler := api.NewGRPCHandler(svc)
	reportv1.RegisterReportServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Error("failed to listen", "error", err)
		return 1
	}

	go func() {
		log.Info("report-svc listening", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server error", "error", err)
		}
	}()

	metricsMux := http.NewServeMux()
	metricsMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})

	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.MetricsPort),
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

	cancel() // stop Kafka consumers (breaks Consume loop)

	grpcServer.GracefulStop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error("metrics server forced shutdown", "error", err)
	}

	log.Info("server stopped")
	return 0
}

type config struct {
	GRPCPort     string
	DatabaseURL  string
	RedisURL     string
	KafkaBrokers []string
	JWTSecret    string
	MetricsPort  string
	UnleashURL   string
	UnleashToken string
	EnabledFlags string
	CacheTTL     string
}

func loadConfig() config {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50057"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://aureum:aureum@localhost:5432/reportdb"
	}
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		fmt.Fprintf(os.Stderr, "JWT_SECRET is required\n")
		os.Exit(1)
	}
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9097"
	}

	return config{
		GRPCPort:     port,
		DatabaseURL:  dbURL,
		RedisURL:     redisURL,
		KafkaBrokers: []string{brokers},
		JWTSecret:    secret,
		MetricsPort:  metricsPort,
		UnleashURL:   os.Getenv("UNLEASH_URL"),
		UnleashToken: os.Getenv("UNLEASH_TOKEN"),
		EnabledFlags: os.Getenv("ENABLED_FLAGS"),
		CacheTTL:     os.Getenv("CACHE_TTL"),
	}
}

type envFlag struct {
	flags []string
}

func (e *envFlag) IsEnabled(_ context.Context, flag string) bool {
	for _, f := range e.flags {
		if strings.TrimSpace(f) == flag {
			return true
		}
	}
	return false
}

type unleashFlag struct {
	client *ff.Client
}

func (u *unleashFlag) IsEnabled(ctx context.Context, flag string) bool {
	return u.client.IsEnabled(ctx, flag)
}

type ctxKey string

const userIDKey ctxKey = "user_id"

func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	userID := extractUserIDFromMetadata(ctx)
	if userID == "" {
		userID = "system"
	}
	ctx = context.WithValue(ctx, userIDKey, userID)
	return handler(ctx, req)
}

func extractUserIDFromMetadata(ctx context.Context) string {
	return ""
}
