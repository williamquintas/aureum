package main

import (
	"context"
	"fmt"
	"log"
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
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	"github.com/aureum/pkg/cache"
	ff "github.com/aureum/pkg/featureflag"
	"github.com/aureum/pkg/idempotency"
	"github.com/aureum/pkg/kafka"
	"github.com/aureum/pkg/outbox"
	"github.com/aureum/pkg/telemetry"

	"github.com/aureum/debt-svc/internal/application"
	"github.com/aureum/debt-svc/internal/infrastructure/api"
	"github.com/aureum/debt-svc/internal/infrastructure/persistence"
	debtv1 "github.com/aureum/proto/gen/debt/debtv1"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := loadConfig()

	if err := telemetry.InitOTEL("debt-svc", "1.0.0"); err != nil {
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

	outboxRepo := persistence.NewOutboxRepository(dbPool)
	debtRepo := persistence.NewDebtRepo(dbPool)
	paymentRepo := persistence.NewPaymentRepo(dbPool)
	idempStore := idempotency.NewStore(rdb)

	kafkaProducer, err := kafka.NewProducer(cfg.KafkaBrokers)
	if err != nil {
		log.Error("failed to create kafka producer", "error", err)
		return 1
	}
	defer kafkaProducer.Close()

	outboxStore := outbox.NewStore(dbPool)
	outboxPublisher := outbox.NewPublisher(outboxStore, kafkaProducer, "debt-events", 5*time.Second)
	outboxPublisher.Start(ctx)

	var flagClient application.FeatureFlag
	if cfg.UnleashURL != "" && cfg.UnleashToken != "" {
		uc, err := ff.NewClient(cfg.UnleashURL, "debt-svc", cfg.UnleashToken)
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

	svc := application.NewService(
		debtRepo, paymentRepo,
		outboxRepo, idempStore, redisCache, flagClient,
	)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor),
		telemetry.GRPCUnaryInterceptor(),
	)

	handler := api.NewGRPCHandler(svc)
	debtv1.RegisterDebtServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Error("failed to listen", "error", err)
		return 1
	}

	go func() {
		log.Info("debt-svc listening", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Error("gRPC server error", "error", err)
		}
	}()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "# metrics endpoint ready")
	}))
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

	outboxPublisher.Stop()
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
		port = "50055"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://aureum:aureum@localhost:5432/debtdb"
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
		log.Fatal("JWT_SECRET is required")
	}
	metricsPort := os.Getenv("METRICS_PORT")
	if metricsPort == "" {
		metricsPort = "9095"
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

func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	userID := extractUserIDFromToken(ctx)
	if userID == "" {
		userID = extractUserIDFromMetadata(ctx)
	}
	if userID == "" {
		userID = "system"
	}
	ctx = api.UserContext(ctx, userID)
	return handler(ctx, req)
}

func extractUserIDFromMetadata(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		values := md.Get("x-user-id")
		if len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func extractUserIDFromToken(ctx context.Context) string {
	return ""
}
