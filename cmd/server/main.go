package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	redisclient "github.com/redis/go-redis/v9"

	"github.com/atls-academy/engineer-challenge/internal/application/command"
	"github.com/atls-academy/engineer-challenge/internal/application/query"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/crypto"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/eventbus"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/observability"
	"github.com/atls-academy/engineer-challenge/internal/infrastructure/persistence/postgres"
	redisinfra "github.com/atls-academy/engineer-challenge/internal/infrastructure/persistence/redis"
	gqlhandler "github.com/atls-academy/engineer-challenge/internal/interfaces/graphql"
)

func main() {
	logger := observability.NewLogger(getEnv("LOG_LEVEL", "info"))

	// Database
	db, err := sql.Open("postgres", getEnv("DATABASE_URL", "postgres://auth:auth@localhost:5432/auth?sslmode=disable"))
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	logger.Info("connected to database")

	// Redis
	rdb := redisclient.NewClient(&redisclient.Options{
		Addr:     getEnv("REDIS_URL", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		logger.Warn("redis not available, rate limiting disabled", "error", err)
	} else {
		logger.Info("connected to redis")
	}

	// Infrastructure
	hasher := crypto.NewBcryptHasher()
	tokenIssuer := crypto.NewJWTIssuer(
		getEnv("JWT_SECRET", "change-me-in-production-32-chars!"),
		getEnv("JWT_ISSUER", "auth-service"),
	)
	bus := eventbus.NewInMemoryEventBus(logger)
	rateLimiter := redisinfra.NewRateLimiter(rdb)

	// Repositories
	userRepo := postgres.NewUserRepository(db)
	sessionRepo := postgres.NewSessionRepository(db)

	// Command handlers
	registerHandler := command.NewRegisterUserHandler(userRepo, hasher, bus, logger)
	authenticateHandler := command.NewAuthenticateUserHandler(userRepo, sessionRepo, hasher, tokenIssuer, bus, logger)
	refreshTokenHandler := command.NewRefreshTokenHandler(sessionRepo, tokenIssuer, bus, logger)
	requestResetHandler := command.NewRequestPasswordResetHandler(userRepo, bus, logger)
	resetPasswordHandler := command.NewResetPasswordHandler(userRepo, hasher, bus, logger)

	// Query handlers
	getUserHandler := query.NewGetUserByIDHandler(userRepo, logger)

	// GraphQL
	resolver := gqlhandler.NewResolver(gqlhandler.ResolverConfig{
		RegisterHandler:      registerHandler,
		AuthenticateHandler:  authenticateHandler,
		RefreshTokenHandler:  refreshTokenHandler,
		RequestResetHandler:  requestResetHandler,
		ResetPasswordHandler: resetPasswordHandler,
		GetUserHandler:       getUserHandler,
		TokenValidator:       tokenIssuer,
		RateLimiter:          rateLimiter,
		Logger:               logger,
	})

	schema, err := resolver.Schema()
	if err != nil {
		logger.Error("failed to create GraphQL schema", "error", err)
		os.Exit(1)
	}

	// HTTP Server
	mux := http.NewServeMux()

	// CORS middleware
	corsHandler := corsMiddleware(gqlhandler.NewHandler(schema))
	mux.Handle("/graphql", corsHandler)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"ok"}`)
	})

	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("server starting", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}
	logger.Info("server stopped")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
