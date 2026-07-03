package config

import "os"

// Config holds configuration for the User Service.
type Config struct {
	Port              string
	DatabaseURL       string
	S3Bucket          string
	S3Region          string
	GoogleClientID    string
	GoogleClientSecret string
	GoogleRedirectURL string
	JWTSecret         string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8081"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/users?sslmode=disable"),
		S3Bucket:           getEnv("S3_BUCKET", ""),
		S3Region:           getEnv("S3_REGION", ""),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/v1/auth/google/callback"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
