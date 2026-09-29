package config

import (
	"log"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServerPort        string
	DatabaseURL       string
	TimepadToken      string
	KulturaAPIKey     string
	MaxBotToken       string
	MiniAppURL        string
	IngestCities      []string
	IngestInterval    time.Duration
	SkipInitDataCheck bool
}

func Load() *Config {
	return &Config{
		ServerPort:        getEnv("SERVER_PORT", ":8080"),
		DatabaseURL:       mustEnv("DATABASE_URL"),
		TimepadToken:      os.Getenv("TIMEPAD_API_TOKEN"),
		KulturaAPIKey:     os.Getenv("KULTURA_API_KEY"),
		IngestCities:      parseCities(os.Getenv("INGEST_CITIES")),
		IngestInterval:    parseDuration(os.Getenv("INGEST_INTERVAL"), 6*time.Hour),
		MaxBotToken:       os.Getenv("MAX_BOT_TOKEN"),
		MiniAppURL:        os.Getenv("MINI_APP_URL"),
		SkipInitDataCheck: os.Getenv("SKIP_INIT_DATA_CHECK") == "true",
	}
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("config: mustEnv (environmental value not write) %s", key)
	}
	return val
}

func getEnv(key string, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return defaultValue
}

func parseCities(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	cities := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			cities = append(cities, p)
		}
	}
	if len(cities) == 0 {
		return nil
	}
	return cities
}

func parseDuration(raw string, def time.Duration) time.Duration {
	if raw == "" {
		return def
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("config: INGEST_INTERVAL=%q env value incorrect, %s will be provided by default", raw, def)
		return def
	}
	return d
}
