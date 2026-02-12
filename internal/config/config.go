package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	BotToken      string
	DatabaseURL   string
	PollerTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		BotToken:      os.Getenv("BOT_TOKEN"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		PollerTimeout: 10 * time.Second,
	}

	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = defaultPostgresDSN()
	}

	return cfg, nil
}

func defaultPostgresDSN() string {
	host := getEnvOrDefault("POSTGRES_HOST", "127.0.0.1")
	port := getEnvOrDefault("POSTGRES_PORT", "5432")
	user := getEnvOrDefault("POSTGRES_USER", "teamtimebot")
	password := getEnvOrDefault("POSTGRES_PASSWORD", "teamtimebot")
	db := getEnvOrDefault("POSTGRES_DB", "teamtimebot")
	sslmode := getEnvOrDefault("POSTGRES_SSLMODE", "disable")
	timezone := getEnvOrDefault("TZ", "UTC")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		host,
		port,
		user,
		password,
		db,
		sslmode,
		timezone,
	)
}

func getEnvOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
