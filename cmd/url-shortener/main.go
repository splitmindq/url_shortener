package main

import (
	"URL-Shortener/internal/config"
	"fmt"
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

	//TODO DELETE
	fmt.Println(cfg)

	log := setupLogger(cfg.Env)
	log.Info("starting url-shortener", slog.String("env", cfg.Env))

	if cfg.Env == envLocal || cfg.Env == envDev {
		log.Debug("debug messages are enabled")
	}
	//storage

	//router

	//server
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
