package config

import (
    "fmt"
    "os"
    "strconv"
)

type Config struct {
    HTTPAddr           string
    DatabaseURL        string
    RedisAddr          string
    RedisPassword      string
    RedisDB            int
    RedisKeyPrefix     string
    RedisTTLSeconds    int
    LLMAPIKey          string
    LLMBaseURL         string
    LLMModel           string
    LLMTimeoutSeconds  int
    LLMMaxTokens       int
    LLMMaxRetries      int
    MaxMessages        int
    MaxContentLength   int
    LogLevel           string
    LogFormat          string
    MigrationsDir      string
}

func Load() (Config, error) {
    c := Config{
        HTTPAddr:          envString("HTTP_ADDR", ":8091"),
        DatabaseURL:       os.Getenv("DATABASE_URL"),
        RedisAddr:         envString("REDIS_ADDR", "localhost:6379"),
        RedisPassword:     os.Getenv("REDIS_PASSWORD"),
        RedisDB:           envInt("REDIS_DB", 0),
        RedisKeyPrefix:    envString("REDIS_KEY_PREFIX", "conversation-summary:"),
        RedisTTLSeconds:   envInt("REDIS_TTL_SECONDS", 3600),
        LLMAPIKey:         envString("LLM_API_KEY", "ollama"),
        LLMBaseURL:        envString("LLM_BASE_URL", "http://localhost:11434/v1"),
        LLMModel:          envString("LLM_MODEL", "qwen3:4b"),
        LLMTimeoutSeconds: envInt("LLM_TIMEOUT_SECONDS", 60),
        LLMMaxTokens:      envInt("LLM_MAX_TOKENS", 1000),
        LLMMaxRetries:     envInt("LLM_MAX_RETRIES", 3),
        MaxMessages:       envInt("MAX_MESSAGES", 30),
        MaxContentLength:  envInt("MAX_CONTENT_LENGTH", 20000),
        LogLevel:          envString("LOG_LEVEL", "info"),
        LogFormat:         envString("LOG_FORMAT", "json"),
        MigrationsDir:     envString("MIGRATIONS_DIR", "./migrations"),
    }

    if c.DatabaseURL == "" {
        return Config{}, fmt.Errorf("DATABASE_URL is required")
    }
    if c.RedisTTLSeconds < 1 || c.LLMTimeoutSeconds < 1 || c.LLMMaxTokens < 1 || c.LLMMaxRetries < 1 || c.MaxMessages < 1 || c.MaxContentLength < 1 {
        return Config{}, fmt.Errorf("numeric configuration values must be positive")
    }
    return c, nil
}

func envString(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}

func envInt(key string, fallback int) int {
    value := os.Getenv(key)
    if value == "" {
        return fallback
    }
    parsed, err := strconv.Atoi(value)
    if err != nil {
        return fallback
    }
    return parsed
}
