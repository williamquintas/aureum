package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	"github.com/vektah/gqlparser/v2/ast"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/aureum/graphql-bff/graph"
)

type Config struct {
	Port              string `envconfig:"PORT" default:"8082"`
	TransactionSvc    string `envconfig:"TRANSACTION_SVC" default:"localhost:50054"`
	IdentitySvc       string `envconfig:"IDENTITY_SVC" default:"localhost:50053"`
	PlaygroundEnabled bool   `envconfig:"PLAYGROUND_ENABLED" default:"true"`
}

func main() {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	txConn, err := grpc.Dial(cfg.TransactionSvc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to transaction-svc: %v", err)
	}
	defer txConn.Close()

	idConn, err := grpc.Dial(cfg.IdentitySvc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to identity-svc: %v", err)
	}
	defer idConn.Close()

	resolver := graph.NewResolver(txConn, idConn)

	srv := handler.New(
		graph.NewExecutableSchema(
			graph.Config{
				Resolvers: resolver,
				Directives: graph.DirectiveRoot{
					Auth: graph.AuthDirective(resolver.IDClient),
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

	r.Handle("/graphql", srv)

	if cfg.PlaygroundEnabled {
		r.Handle("/playground", playground.Handler("GraphQL Playground", "/graphql"))
	}

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("graphql-bff listening on port %s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")
	httpServer.Shutdown(context.Background())
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
