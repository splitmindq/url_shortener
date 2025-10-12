package main

import (
	"URL-Shortener/internal/config"
	logger "URL-Shortener/internal/http-server/middleware"
	"URL-Shortener/internal/lib/logger/sl"
	"URL-Shortener/internal/storage/sqlite"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"os"
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

	storage, err := sqlite.NewStorage(cfg.StoragePath)
	if err != nil {
		log.Error("error init storage", sl.Err(err))
		os.Exit(1)
	}

	_ = storage

	router := chi.NewRouter()

	//mw
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(logger.New(log))
}

func setupLogger(env string) *slog.Logger {
	log := new(slog.Logger)
	switch env {
	case "local":
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource:   false,
			Level:       slog.LevelDebug,
			ReplaceAttr: nil,
		}))
	case "development":
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource:   false,
			Level:       slog.LevelDebug,
			ReplaceAttr: nil,
		}))
	case "production":
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource:   false,
			Level:       slog.LevelInfo,
			ReplaceAttr: nil,
		}))

	}
	return log
}
