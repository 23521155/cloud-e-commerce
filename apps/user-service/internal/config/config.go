package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the user service,
// loaded from environment variables.
type Config struct {
	AppName string

	Port string

	// DatabaseURL is a sqlserver:// URL, also used by golang-migrate.
	DatabaseURL string

	// JWTSecret and JWTIssuer must equal the gateway's: it verifies the
	// access tokens this service signs.
	JWTSecret string
	JWTIssuer string

	// AccessTokenTTL is how long an access token is accepted.
	AccessTokenTTL time.Duration

	// RefreshTokenTTL is how long a signed-in device stays signed in
	// without being used.
	RefreshTokenTTL time.Duration
}

// Load reads environment variables (via a .env file) into a Config.
func Load() (*Config, error) {

	// Missing file is not fatal: in containers, config comes from
	// env vars injected by the runtime, not a .env baked into the image.
	if err := godotenv.Load("configs/.env"); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	cfg := &Config{
		AppName: getEnv("APP_NAME", "user-service"),

		Port: getEnv("PORT", "8085"),

		DatabaseURL: os.Getenv("DATABASE_URL"),

		JWTSecret: os.Getenv("JWT_SECRET"),

		JWTIssuer: os.Getenv("JWT_ISSUER"),
	}

	ttl, err := time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_TTL: %w", err)
	}
	cfg.AccessTokenTTL = ttl

	refreshTTL, err := time.ParseDuration(getEnv("JWT_REFRESH_TTL", "720h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_TTL: %w", err)
	}
	cfg.RefreshTokenTTL = refreshTTL

	return cfg, nil
}

// getEnv returns the value of key, or fallback when it is unset or empty.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
