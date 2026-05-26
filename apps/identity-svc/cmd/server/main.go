package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelseyhightower/envconfig"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	"github.com/aureum/identity-svc/internal/application"
	"github.com/aureum/identity-svc/internal/infrastructure/api"
	kc "github.com/aureum/identity-svc/internal/infrastructure/auth"
	appcache "github.com/aureum/identity-svc/internal/infrastructure/cache"
	"github.com/aureum/identity-svc/internal/infrastructure/middleware"
	"github.com/aureum/identity-svc/internal/infrastructure/persistence"
	"github.com/aureum/pkg/cache"
	"github.com/aureum/pkg/idempotency"
)

type Config struct {
	Port              string `envconfig:"PORT" default:"8080"`
	GRPCPort          string `envconfig:"GRPC_PORT" default:"9090"`
	DatabaseURL       string `envconfig:"DATABASE_URL" required:"true"`
	ReadDatabaseURL   string `envconfig:"READ_DATABASE_URL" required:"true"`
	RedisAddr         string `envconfig:"REDIS_ADDR" default:"localhost:6379"`
	RedisPassword     string `envconfig:"REDIS_PASSWORD"`
	KafkaBrokers      string `envconfig:"KAFKA_BROKERS" default:"localhost:9092"`
	KeycloakURL       string `envconfig:"KEYCLOAK_URL" default:"http://localhost:8081"`
	KeycloakRealm     string `envconfig:"KEYCLOAK_REALM" default:"aureum"`
	KeycloakClientID  string `envconfig:"KEYCLOAK_CLIENT_ID" default:"identity-svc-confidential"`
	KeycloakClientSec string `envconfig:"KEYCLOAK_CLIENT_SECRET" required:"true"`
	JWTSecret         string `envconfig:"JWT_SECRET" required:"true"`
	RateLimitPerIP    int    `envconfig:"RATE_LIMIT_PER_IP" default:"5"`
	RateLimitWindow   string `envconfig:"RATE_LIMIT_WINDOW" default:"15m"`
	CacheTTL          string `envconfig:"CACHE_TTL" default:"5m"`
	IdempotencyTTL    string `envconfig:"IDEMPOTENCY_TTL" default:"24h"`
}

func main() {
	os.Exit(run())
}

func run() int {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Error("failed to load config", "error", err)
		return 1
	}

	writePool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to write database", "error", err)
		return 1
	}
	defer writePool.Close()

	readPool, err := pgxpool.New(ctx, cfg.ReadDatabaseURL)
	if err != nil {
		log.Error("failed to connect to read database", "error", err)
		return 1
	}
	defer readPool.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("failed to connect to redis", "error", err)
		return 1
	}
	defer func() { _ = rdb.Close() }()

	redisCache, err := cache.NewRedisCache(cfg.RedisAddr, cfg.RedisPassword, 0)
	if err != nil {
		log.Error("failed to create redis cache", "error", err)
		return 1
	}

	idempStore := idempotency.NewStore(rdb)
	keycloakClient := kc.NewKeycloakClient(cfg.KeycloakURL, cfg.KeycloakRealm, cfg.KeycloakClientID, cfg.KeycloakClientSec)
	tokenBlacklist := appcache.NewTokenBlacklist(rdb)

	writeRepo := persistence.NewUserWriteRepository(writePool)
	outboxRepo := persistence.NewOutboxRepository(writePool)

	authSvc := application.NewAuthService(
		writeRepo, keycloakClient, outboxRepo,
		idempStore, redisCache, tokenBlacklist, cfg.JWTSecret,
	)

	handler := api.NewHandler(authSvc)

	rateLimitWindow, err := time.ParseDuration(cfg.RateLimitWindow)
	if err != nil {
		rateLimitWindow = 15 * time.Minute
	}
	rateLimiter := middleware.NewRateLimiter(rdb, cfg.RateLimitPerIP, rateLimitWindow)

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(rateLimiter.Middleware)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "ok")
	})

	handler.RegisterRoutes(r, cfg.JWTSecret)

	httpServer := &http.Server{
		Addr:         net.JoinHostPort("", cfg.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		log.Info("starting HTTP server", "port", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server error", "error", err)
		}
	}()

	lc := net.ListenConfig{}
	grpcListener, err := lc.Listen(ctx, "tcp", net.JoinHostPort("", cfg.GRPCPort))
	if err != nil {
		log.Error("failed to listen for gRPC", "error", err)
		return 1
	}

	grpcServer := grpc.NewServer()

	go func() {
		log.Info("starting gRPC server", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Error("gRPC server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info("shutting down", "signal", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	grpcServer.GracefulStop()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("http server forced shutdown", "error", err)
	}

	log.Info("server stopped")
	return 0
}
