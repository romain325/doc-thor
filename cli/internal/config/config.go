package config

import (
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

type Server struct {
	URL    string `toml:"url"`
	APIKey string `toml:"api_key,omitempty"`
}

type Config struct {
	Server Server `toml:"server"`
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "doc-thor", "config.toml"), nil
}

func Load() (Config, error) {
	var cfg Config
	path, err := configPath()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	err = toml.Unmarshal(data, &cfg)
	return cfg, err
}

func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
