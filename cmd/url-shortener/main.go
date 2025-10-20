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
	"fmt"
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

	router := setupRouter(log, storage, cfg)

	server := setupServer(cfg, router)

	runServer(server, log, cfg)
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

func setupRouter(log *slog.Logger, storage *postgres.Storage, cfg *config.Config) *chi.Mux {
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

	return router
}

func setupServer(cfg *config.Config, r *chi.Mux) *http.Server {
	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      r,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
	return server
}

func runServer(server *http.Server, log *slog.Logger, cfg *config.Config) {
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("starting server", slog.String("address", cfg.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("server error: %w", err)
		}
	}()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Error("server failed", sl.Err(err))
		os.Exit(1)
	case sig := <-osSignals:
		log.Info("received signal", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Error("graceful shutdown failed", sl.Err(err))
			if closeErr := server.Close(); closeErr != nil {
				log.Error("forced shutdown also failed", sl.Err(closeErr))
			}
		} else {
			log.Info("server stopped gracefully")
		}
	}
}
