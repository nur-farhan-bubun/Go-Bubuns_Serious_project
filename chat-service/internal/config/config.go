package config

import "os"

// Config holds configuration for the Chat Service.
type Config struct {
	Port            string
	DatabaseURL     string
	RedisURL        string
	KafkaBrokers    string
	UserServiceURL  string
	UserServiceGRPC string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8083"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/chat?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis:6379"),
		KafkaBrokers:    getEnv("KAFKA_BROKERS", "kafka:9092"),
		UserServiceURL:  getEnv("USER_SERVICE_URL", "http://user-service:8081"),
		UserServiceGRPC: getEnv("USER_SERVICE_GRPC", "user-service:50051"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
