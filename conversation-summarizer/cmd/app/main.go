package main

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/Garmoro/conversation-summarizer/internal/config"
    "github.com/Garmoro/conversation-summarizer/internal/handler"
    "github.com/Garmoro/conversation-summarizer/internal/llm"
    "github.com/Garmoro/conversation-summarizer/internal/repository/postgres"
    "github.com/Garmoro/conversation-summarizer/internal/repository/redisrepo"
    "github.com/Garmoro/conversation-summarizer/internal/service"
)

func main() {
    cfg, err := config.Load()
    if err != nil { panic(err) }
    logger := newLogger(cfg)
    if err := run(context.Background(), cfg, logger); err != nil { logger.Error("application stopped", "error", err); os.Exit(1) }
}

func run(parent context.Context, cfg config.Config, logger *slog.Logger) error {
    ctx, stop := signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
    defer stop()
    logger.Info("application starting", "http_addr", cfg.HTTPAddr, "llm_model", cfg.LLMModel)
    if err := postgres.RunMigrations(ctx, cfg.DatabaseURL, cfg.MigrationsDir); err != nil { return err }
    repo, err := postgres.New(ctx, cfg.DatabaseURL)
    if err != nil { return err }
    defer repo.Close()
    cache := redisrepo.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, cfg.RedisKeyPrefix, time.Duration(cfg.RedisTTLSeconds)*time.Second)
    defer cache.Close()
    if err := cache.Ping(ctx); err != nil { return fmt.Errorf("redis unavailable: %w", err) }
    llmClient := llm.NewHTTPClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMMaxTokens, cfg.LLMMaxRetries, time.Duration(cfg.LLMTimeoutSeconds)*time.Second, logger)
    summaryService := service.New(repo, cache, llmClient, cfg.MaxMessages, cfg.MaxContentLength, logger)
    server := &http.Server{Addr: cfg.HTTPAddr, Handler: handler.New(summaryService, repo, cache, logger).Routes(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: time.Duration(cfg.LLMTimeoutSeconds+10) * time.Second, IdleTimeout: 60 * time.Second}
    serverErrors := make(chan error, 1)
    go func() { logger.Info("HTTP server started", "addr", cfg.HTTPAddr); serverErrors <- server.ListenAndServe() }()
    select {
    case err := <-serverErrors:
        if errors.Is(err, http.ErrServerClosed) { return nil }
        return fmt.Errorf("HTTP server: %w", err)
    case <-ctx.Done():
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        logger.Info("application shutting down")
        return server.Shutdown(shutdownCtx)
    }
}

func newLogger(cfg config.Config) *slog.Logger {
    level := new(slog.LevelVar)
    switch cfg.LogLevel { case "debug": level.Set(slog.LevelDebug); case "warn": level.Set(slog.LevelWarn); case "error": level.Set(slog.LevelError); default: level.Set(slog.LevelInfo) }
    options := &slog.HandlerOptions{Level: level}
    if cfg.LogFormat == "text" { return slog.New(slog.NewTextHandler(os.Stdout, options)) }
    return slog.New(slog.NewJSONHandler(os.Stdout, options))
}
