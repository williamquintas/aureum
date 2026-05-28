package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	"github.com/aureum/pkg/idempotency"

	transactionv1 "github.com/aureum/proto/gen/transaction/transactionv1"
	"github.com/aureum/transaction-svc/internal/application"
	"github.com/aureum/transaction-svc/internal/infrastructure/api"
	"github.com/aureum/transaction-svc/internal/infrastructure/persistence"
)

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	defer rdb.Close()

	outboxRepo := persistence.NewOutboxRepository(dbPool)

	incomeRepo := persistence.NewIncomeRepo(dbPool)
	fixedExpenseRepo := persistence.NewFixedExpenseRepo(dbPool)
	variableExpenseRepo := persistence.NewVariableExpenseRepo(dbPool)

	idempStore := idempotency.NewStore(rdb)

	svc := application.NewService(
		incomeRepo, fixedExpenseRepo, variableExpenseRepo,
		outboxRepo, idempStore,
	)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(authInterceptor),
	)

	handler := api.NewGRPCHandler(svc)
	transactionv1.RegisterTransactionServiceServer(grpcServer, handler)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", cfg.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		log.Printf("transaction-svc listening on port %s", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	grpcServer.GracefulStop()
}

type config struct {
	GRPCPort     string
	DatabaseURL  string
	RedisURL     string
	KafkaBrokers []string
	JWTSecret    string
}

func loadConfig() config {
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50054"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://aureum:aureum@localhost:5432/transactiondb"
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

	return config{
		GRPCPort:     port,
		DatabaseURL:  dbURL,
		RedisURL:     redisURL,
		KafkaBrokers: []string{brokers},
		JWTSecret:    secret,
	}
}

func authInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	userID := extractUserIDFromToken(ctx)
	if userID == "" {
		userID = extractUserIDFromMetadata(ctx)
	}
	if userID == "" {
		userID = "system"
	}
	ctx = context.WithValue(ctx, "user_id", userID)
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
