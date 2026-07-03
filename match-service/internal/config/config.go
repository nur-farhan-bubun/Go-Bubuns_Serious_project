package config

import "os"

// Config holds configuration for the Match Service.
type Config struct {
	Port           string
	DatabaseURL    string
	UserServiceURL string
	KafkaBrokers   string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8082"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/matches?sslmode=disable"),
		UserServiceURL: getEnv("USER_SERVICE_URL", "http://user-service:8081"),
		KafkaBrokers:   getEnv("KAFKA_BROKERS", "kafka:9092"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
