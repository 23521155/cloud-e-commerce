package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config holds all runtime configuration for the API Gateway,
// loaded from environment variables.
type Config struct {
	AppName string

	Port string

	OTelEndpoint string

	JWTSecret string

	JWTIssuer string

	AccessTokenExpire string

	RoutesConfig []RouteConfig

	// TrustedProxies are the proxies (IPs or CIDRs) whose X-Forwarded-For the
	// gateway believes when working out the client IP. Empty: trust none.
	TrustedProxies []string
}

// Load reads environment variables (via a .env file) into a Config.
func Load() (*Config, error) {	

	// Load .env into the process environment before reading values.
	// Missing file is not fatal: in containers, config comes from
	// env vars injected by the runtime (e.g. Docker Compose), not a
	// physical .env file baked into the image.
	if err := godotenv.Load("configs/.env"); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	cfg := &Config{
		AppName: os.Getenv("APP_NAME"),

		Port: os.Getenv("PORT"),

		OTelEndpoint: os.Getenv("OTEL_ENDPOINT"),

		JWTSecret: os.Getenv("JWT_SECRET"),

		JWTIssuer: os.Getenv("JWT_ISSUER"),

		AccessTokenExpire: os.Getenv("ACCESS_TOKEN_EXPIRE"),

		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
	}

	routesData, err := os.ReadFile("configs/routes.yaml")
	if err != nil {
		return nil, err
	}

	var wrapper struct {
		Routes []RouteConfig `yaml:"routes"`
	}

	if err := yaml.Unmarshal(routesData, &wrapper); err != nil {
		return nil, err
	}

	for i := range wrapper.Routes {
		wrapper.Routes[i].Upstream = os.ExpandEnv(wrapper.Routes[i].Upstream)
	}

	cfg.RoutesConfig = wrapper.Routes
	
	return cfg, nil
}

// splitList turns "a, b,,c" into [a b c]; an empty string gives nil.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
