package config

import (
	"net"
	"os"
)

type Config struct {
	SupabaseURL      string
	SupabaseKey      string
	TargetForwardURL string
	Port             string
	ServerID         string // ✅ Campo agregado
	ServerIP         string // ✅ Campo agregado
}

func New() *Config {
	return &Config{
		SupabaseURL:      os.Getenv("SUPABASE_URL"),
		SupabaseKey:      os.Getenv("SUPABASE_KEY"),
		TargetForwardURL: os.Getenv("TARGET_FORWARD_URL"),
		Port:             getEnvOrDefault("PORT", "3000"),
		ServerID:         getEnvOrDefault("SERVER_ID", "srs-paris-01"),
		ServerIP:         getEnvOrDefault("SERVER_IP", getOutboundIP()),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ✅ Obtener IP del servidor automáticamente
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "unknown"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}