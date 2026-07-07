package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "/etc/xray-node/config.yaml"

type Config struct {
	Panel PanelConfig `yaml:"panel"`
	API   APIConfig   `yaml:"api"`
}

type PanelConfig struct {
	URL         string `yaml:"url"`
	Token       string `yaml:"token"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	InsecureTLS bool   `yaml:"insecure_tls"`
}

type APIConfig struct {
	Listen string `yaml:"listen"`
	Key    string `yaml:"key"`
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Panel.URL == "" {
		cfg.Panel.URL = "https://127.0.0.1:2053"
	}
	if cfg.API.Listen == "" {
		cfg.API.Listen = "127.0.0.1:9472"
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.API.Key == "" {
		return fmt.Errorf("api.key is required")
	}
	if c.Panel.Token == "" && (c.Panel.Username == "" || c.Panel.Password == "") {
		return fmt.Errorf("panel.token or panel username/password is required")
	}
	return nil
}
