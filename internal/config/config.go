package config

import (
	"os"
	"strings"
)

// Config contains runtime settings loaded from environment variables.
type Config struct {
	Environment string
	Port        string
	DatabaseURL string
	AutoMigrate bool
}

// Load reads application configuration from environment variables with sane defaults.
func Load() Config {
	return Config{
		Environment: getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/subscription_management?sslmode=disable"),
		AutoMigrate: strings.EqualFold(getEnv("AUTO_MIGRATE", "true"), "true"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
