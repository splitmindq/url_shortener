package main

import (
	"URL-Shortener/internal/config"
	del "URL-Shortener/internal/http-server/handlers/url/delete"
	"URL-Shortener/internal/http-server/handlers/url/get"
	"URL-Shortener/internal/http-server/handlers/url/redirect"
	"URL-Shortener/internal/http-server/handlers/url/save"
	logger "URL-Shortener/internal/http-server/middleware"
	"URL-Shortener/internal/lib/logger/sl"
	"URL-Shortener/internal/storage/postgres"
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	envLocal = "local"
	envProd  = "production"
	envDev   = "development"
)

func main() {

	cfg := config.MustLoadConfig()

	log := setupLogger(cfg.Env)
	log.Info("starting url-shortener", slog.String("env", cfg.Env))

	if cfg.Env == envLocal || cfg.Env == envDev {
		log.Debug("debug messages are enabled")
	}

	storage, err := postgres.NewStorage(cfg.ConnString(), cfg)
	if err != nil {
		log.Error("error init storage", sl.Err(err))
		os.Exit(1)
	}
	defer storage.Close()

	go monitorPoolStats(storage, log)

	router := chi.NewRouter()

	//mw
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	//mw-l
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)
	//should delete if router will be changed
	router.Use(middleware.URLFormat)

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/url", save.New(log, storage, cfg.AliasLength, cfg.MaxAttempts))
		r.Get("/url/{alias}", get.New(log, storage))
		r.Delete("/url/{alias}", del.New(log, storage))
		r.Get("/{alias}", redirect.New(log, storage))
	})

	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("starting server", slog.String("address", cfg.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Error("Server failed: %v", err)

	case sig := <-osSignals:
		log.Info("Received signal: %v", sig)
		log.Info("Starting graceful shutdown...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Info("⚠️ Graceful shutdown failed: %v", err)
			server.Close()
		}

		log.Info("Server stopped gracefully")
	}
}

func setupLogger(env string) *slog.Logger {
	log := new(slog.Logger)
	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource:   false,
			Level:       slog.LevelDebug,
			ReplaceAttr: nil,
		}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource:   false,
			Level:       slog.LevelDebug,
			ReplaceAttr: nil,
		}))
	case envProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource:   false,
			Level:       slog.LevelInfo,
			ReplaceAttr: nil,
		}))

	}
	return log
}

func monitorPoolStats(storage *postgres.Storage, log *slog.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := storage.GetPoolStats()
		if stats != nil {
			log.Debug("DB pool statistics",
				slog.Int("total_connections", int(stats.TotalConns())),
				slog.Int("idle_connections", int(stats.IdleConns())),
				slog.Int("max_connections", int(stats.MaxConns())),
				slog.Int("acquired_connections", int(stats.AcquiredConns())),
			)
		}
	}
}
