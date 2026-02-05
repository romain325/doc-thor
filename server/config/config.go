package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port             string
	DatabaseURL      string
	NginxConfigDir   string
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageUseSSL    bool
	BuilderEndpoints []string
	SessionTTLHours  int
	InitialUser      string
	InitialPassword  string
}

func Load() Config {
	return Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "./data.db"),
		NginxConfigDir:   getEnv("NGINX_CONFIG_DIR", "/etc/nginx/sites-enabled"),
		StorageEndpoint:  getEnv("STORAGE_ENDPOINT", ""),
		StorageAccessKey: getEnv("STORAGE_ACCESS_KEY", ""),
		StorageSecretKey: getEnv("STORAGE_SECRET_KEY", ""),
		StorageUseSSL:    getEnvBool("STORAGE_USE_SSL", false),
		BuilderEndpoints: getEnvList("BUILDER_ENDPOINTS"),
		SessionTTLHours:  getEnvInt("SESSION_TTL_HOURS", 24),
		InitialUser:      getEnv("INITIAL_USER", ""),
		InitialPassword:  getEnv("INITIAL_PASSWORD", ""),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getEnvList(key string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return nil
	}
	return strings.Split(v, ",")
}
