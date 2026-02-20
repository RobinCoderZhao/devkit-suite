package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RobinCoderZhao/devkit-suite/internal/api"
	"github.com/RobinCoderZhao/devkit-suite/internal/user"
	"github.com/RobinCoderZhao/devkit-suite/internal/watchbot"
	"github.com/RobinCoderZhao/devkit-suite/pkg/storage"
	_ "modernc.org/sqlite"
)

func main() {
	port := getEnv("API_PORT", "8080")
	dbPath := getEnv("WATCHBOT_DB", "data/watchbot.db")
	jwtSecret := getEnv("JWT_SECRET", "super-secret-devkit-jwt-key")

	db, err := storage.Open(storage.Config{Driver: storage.SQLite, DSN: dbPath})
	if err != nil {
		slog.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initial Migration
	schemaContent, err := os.ReadFile("pkg/storage/schema.sql")
	if err == nil {
		ctx := context.Background()
		if err := db.Migrate(ctx, string(schemaContent)); err != nil {
			slog.Error("schema migration failed", "error", err)
			os.Exit(1)
		}
	}

	uStore := user.NewStore(db)
	wStore := watchbot.NewStore(db)

	server := api.NewServer(uStore, wStore, jwtSecret)
	mux := server.Routes()

	// Add CORS middleware
	handler := corsMiddleware(mux)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	go func() {
		slog.Info("Starting REST API Server", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// corsMiddleware simple middleware to allow Dev Next.js local development
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000") // Next.js default port
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
