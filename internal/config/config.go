package config

import (
	"fmt"
	"os"
	"path/filepath"

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

func SaveExample(path string) error {
	if path == "" {
		path = DefaultPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	example := &Config{
		Panel: PanelConfig{
			URL:         "https://127.0.0.1:2053",
			Token:       "CHANGE_ME_PANEL_API_TOKEN",
			InsecureTLS: true,
		},
		API: APIConfig{
			Listen: "127.0.0.1:9472",
			Key:    "CHANGE_ME_NODE_API_KEY",
		},
	}
	data, err := yaml.Marshal(example)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
