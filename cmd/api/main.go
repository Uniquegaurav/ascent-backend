package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kumargaurav/summit-backend/internal/auth"
	"github.com/kumargaurav/summit-backend/internal/config"
	"github.com/kumargaurav/summit-backend/internal/db"
	"github.com/kumargaurav/summit-backend/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		slog.Error("migrations failed", "err", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	// Hourly sweep of expired OTPs and dead refresh tokens.
	cleanupCtx, stopCleanup := context.WithCancel(ctx)
	defer stopCleanup()
	go func() {
		authRepo := auth.NewRepo(pool)
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			if err := authRepo.CleanupExpired(cleanupCtx); err != nil && cleanupCtx.Err() == nil {
				slog.Error("auth cleanup failed", "err", err)
			}
			select {
			case <-ticker.C:
			case <-cleanupCtx.Done():
				return
			}
		}
	}()

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.New(pool, cfg),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("summit-backend listening", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}
}
