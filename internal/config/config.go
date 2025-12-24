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
	Env               string
	HTTPPort          string
	DatabaseURL       string
	DefaultCurrency   string
	JWTSecret         string
	PublicBaseURL     string
	UploadDir         string
	AccessTokenTTL    time.Duration
	RefreshTokenTTL   time.Duration
	GoogleClientID    string
	FirebaseProjectID string
	FirebaseCredFile  string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// Load reads environment variables and .env (if present).
func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		Env:               getEnv("APP_ENV", "development"),
		HTTPPort:          getEnv("HTTP_PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		DefaultCurrency:   getEnv("CURRENCY_CODE", "IDR"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		PublicBaseURL:     getEnv("PUBLIC_BASE_URL", ""),
		UploadDir:         getEnv("UPLOAD_DIR", "uploads"),
		AccessTokenTTL:    getDuration("ACCESS_TOKEN_TTL", 30*24*time.Hour),
		RefreshTokenTTL:   getDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		GoogleClientID:    os.Getenv("GOOGLE_CLIENT_ID"),
		FirebaseProjectID: os.Getenv("FIREBASE_PROJECT_ID"),
		FirebaseCredFile:  os.Getenv("FIREBASE_CREDENTIALS"),
		ReadTimeout:       getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:      getDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:       getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout:   getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
	}

	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return cfg, errors.New("JWT_SECRET is required")
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
