package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration for the catalog service,
// loaded from environment variables.
type Config struct {
	AppName string

	Port string
}

// Load reads environment variables (via a .env file) into a Config.
func Load() (*Config, error) {

	// Missing file is not fatal: in containers, config comes from
	// env vars injected by the runtime, not a .env baked into the image.
	if err := godotenv.Load("configs/.env"); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	cfg := &Config{
		AppName: getEnv("APP_NAME", "catalog-service"),

		Port: getEnv("PORT", "8083"),
	}

	return cfg, nil
}

// getEnv returns the value of key, or fallback when it is unset or empty.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
