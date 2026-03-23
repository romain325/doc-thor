package config

import (
	"context"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
	"log"
	"log/slog"
	"os"
)

type StorageConfig struct {
	Endpoint  string `env:"STORAGE_ENDPOINT" yaml:"endpoint"`
	AccessKey string `env:"STORAGE_ACCESS_KEY" yaml:"access_key"`
	SecretKey string `env:"STORAGE_SECRET_KEY" yaml:"secret_key"`
	UseSSL    bool   `env:"STORAGE_USE_SSL" yaml:"use_ssl"`
}

type Config struct {
	ConfigFile      string        `env:"DOCTHOR_CONFIG_FILE"`
	Port            string        `env:"PORT" yaml:"port"`
	DatabaseURL     string        `env:"DATABASE_URL" yaml:"database_url"`
	NginxConfigDir  string        `env:"NGINX_CONFIG_DIR" yaml:"nginx_config_dir"`
	StorageConfig   StorageConfig `yaml:"storage"`
	SessionTTLHours int           `env:"SESSION_TTL_HOURS" yaml:"session_ttl_hours"`
	InitialUser     string        `env:"INITIAL_USER" yaml:"initial_user"`
	InitialPassword string        `env:"INITIAL_PASSWORD" yaml:"inital_password"`
}

// Conf priority: default -> config file -> env variables
func Load() Config {
	ctx := context.Background()
	cfg := Config{
		Port:           "8080",
		DatabaseURL:    "./data.db",
		NginxConfigDir: "/etc/nginx/sites-enabled",
		StorageConfig: StorageConfig{
			UseSSL: false,
		},
		SessionTTLHours: 24,
	}

	confFile, err := os.ReadFile(getEnv("DOCTHOR_CONFIG_FILE", "./config.yaml"))
	if err != nil {
		slog.Warn("No config file found", slog.Any("err", err))
	} else {
		if err := yaml.Unmarshal(confFile, &cfg); err != nil {
			slog.Error("Error loading config file", slog.Any("err", err))
		}
	}

	if err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:           &cfg,
		DefaultOverwrite: true,
	}); err != nil {
		log.Fatal(err)
	}

	slog.Debug("Loaded Config", slog.Any("config", cfg))

	return cfg

}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
