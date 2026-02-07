package config

import "os"

type Config struct {
	SupabaseURL      string
	SupabaseKey      string
	TargetForwardURL string
	Port             string
}

func New() *Config {
	return &Config{
		SupabaseURL:      os.Getenv("SUPABASE_URL"),
		SupabaseKey:      os.Getenv("SUPABASE_KEY"),
		TargetForwardURL: os.Getenv("TARGET_FORWARD_URL"),
		Port:             getEnvOrDefault("PORT", "3000"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}