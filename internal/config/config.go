package config

import (
    "errors"
    "os"
    "strconv"
    "time"

    "github.com/joho/godotenv"
)

// Config holds application runtime configuration.
type Config struct {
    Env             string
    HTTPPort        string
    DatabaseURL     string
    DefaultCurrency string
    ReadTimeout     time.Duration
    WriteTimeout    time.Duration
    IdleTimeout     time.Duration
    ShutdownTimeout time.Duration
}

// Load reads environment variables and .env (if present).
func Load() (Config, error) {
    _ = godotenv.Load()

    cfg := Config{
        Env:             getEnv("APP_ENV", "development"),
        HTTPPort:        getEnv("HTTP_PORT", "8080"),
        DatabaseURL:     os.Getenv("DATABASE_URL"),
        DefaultCurrency: getEnv("CURRENCY_CODE", "IDR"),
        ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
        WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
        IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
        ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
    }

    if cfg.DatabaseURL == "" {
        return cfg, errors.New("DATABASE_URL is required")
    }
    return cfg, nil
}

func getEnv(key, fallback string) string {
    val := os.Getenv(key)
    if val == "" {
        return fallback
    }
    return val
}

func getDuration(key string, fallback time.Duration) time.Duration {
    val := os.Getenv(key)
    if val == "" {
        return fallback
    }
    d, err := time.ParseDuration(val)
    if err != nil {
        // Support seconds as integer without suffix.
        if secs, convErr := strconv.Atoi(val); convErr == nil {
            return time.Duration(secs) * time.Second
        }
        return fallback
    }
    return d
}
