package main

import (
	"log/slog"
	"net/http"
	"os"
	"server/internal/config"
	"server/internal/http_server/handlers/url/redirect"
	"server/internal/http_server/handlers/url/save"
	"server/internal/http_server/middleware/logger"
	"server/internal/lib/logger/sl"
	"server/internal/storage/sqlite"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

const (
    envLocal = "local"
    envDev   = "dev"
    envProd  = "prod"
)

func main() {
	config := config.MustLoad()
	log := setupLogger(config.Env)
	log = log.With(slog.String("env", config.Env)) // к каждому сообщению будет добавляться поле с информацией о текущем окружении

    log.Info("initializing server", slog.String("address", config.Address)) // Помимо сообщения выведем параметр с адресом
    log.Debug("logger debug mode enabled")

	storage, err := sqlite.New(config.StoragePath)
    if err != nil {
        log.Error("failed to initialize storage", sl.Err(err))
		os.Exit(1)
    }

    router := chi.NewRouter()

    router.Use(middleware.RequestID) // Добавляет request_id в каждый запрос, для трейсинга
    router.Use(middleware.Logger) // Логирование всех запросов
    router.Use(logger.New(log))
    router.Use(middleware.Recoverer)  // Если где-то внутри сервера (обработчика запроса) произойдет паника, приложение не должно упасть
    router.Use(middleware.URLFormat) // Парсер URLов поступающих запросов

    router.Route(
        "/url", 
        func(r chi.Router) {
            r.Use(
                middleware.BasicAuth(
                    "url-shortener", 
                    map[string]string{
                        config.HTTPServer.User: config.HTTPServer.Password,
        }))

        r.Post("/", save.New(log, storage))
    })

    
    router.Get("/{alias}", redirect.New(log, storage))


    server := &http.Server {
        Addr: config.Address,
        Handler: router,
        ReadTimeout: config.HTTPServer.Timeout,
        WriteTimeout: config.HTTPServer.Timeout,
        IdleTimeout: config.HTTPServer.IdleTimeout,
    }

    if err := server.ListenAndServe(); err != nil {
        log.Error("failed to start server")
    }

    log.Error("server stopped")
}

func setupLogger(env string) *slog.Logger {
    var log *slog.Logger

    switch env {
    case envLocal:
        log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
    case envDev:
        log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
    case envProd:
        log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
    }

    return log
}