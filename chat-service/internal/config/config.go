package config

import "os"

// Config holds configuration for the Chat Service.
type Config struct {
	Port            string
	ScyllaURL       string
	RedisURL        string
	KafkaBrokers    string
	UserServiceURL  string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8083"),
		ScyllaURL:       getEnv("SCYLLA_URL", "scylladb://chat-db:9042/app_chat"),
		RedisURL:        getEnv("REDIS_URL", "redis:6379"),
		KafkaBrokers:    getEnv("KAFKA_BROKERS", "kafka:9092"),
		UserServiceURL:  getEnv("USER_SERVICE_URL", "http://user-service:8081"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
