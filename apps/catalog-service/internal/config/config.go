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

	// DatabaseURL is a sqlserver:// URL, also used by golang-migrate.
	DatabaseURL string

	// InternalToken is the shared secret for the gRPC calls made by other
	// services. When empty the gRPC server is not started at all.
	InternalToken string

	// GRPCPort is where the gRPC server listens, next to the HTTP Port.
	GRPCPort string
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

		DatabaseURL: os.Getenv("DATABASE_URL"),

		InternalToken: os.Getenv("INTERNAL_TOKEN"),

		GRPCPort: getEnv("GRPC_PORT", "9083"),
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
