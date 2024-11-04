package main

import (
	"Url-shortener-go/internal/config"
	del "Url-shortener-go/internal/http-server/handlers/url/delete"
	"Url-shortener-go/internal/http-server/handlers/url/redirect"
	"Url-shortener-go/internal/http-server/handlers/url/save"
	"Url-shortener-go/internal/http-server/middleware/logger"
	"Url-shortener-go/internal/lib/logger/slog_logger"
	"Url-shortener-go/internal/storage/postgresql"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log/slog"
	"net/http"
	"os"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustReadConfig()

	log := setupLogger(cfg.Env)
	log.Info("starting server")
	log.Debug("debug logging enabled")

	storage, err := postgresql.NewStorage()
	if err != nil {
		log.Error("error creating postgresql storage", slog_logger.Err(err))
		os.Exit(1)
	}

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			cfg.HTTPServer.User: cfg.HTTPServer.Password,
		}))

		r.Post("/", save.New(log, storage))
		r.Delete("/{alias}", del.New(log, storage))
	})

	router.Get("/url/{alias}", redirect.New(log, storage))

	log.Info("server started")
	server := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	if err = server.ListenAndServe(); err != nil {
		log.Error("error starting server", slog_logger.Err(err))
	}

	log.Error("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var loggerToSetup *slog.Logger
	switch env {
	case envLocal:
		loggerToSetup = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		loggerToSetup = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		loggerToSetup = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return loggerToSetup
}
