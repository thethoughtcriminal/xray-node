package inbound

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Spec is a declarative inbound definition applied to 3x-ui.
type Spec struct {
	Remark         string         `yaml:"remark"`
	Protocol       string         `yaml:"protocol"`
	Listen         string         `yaml:"listen"`
	Port           int            `yaml:"port"`
	Enable         *bool          `yaml:"enable"`
	Tag            string         `yaml:"tag"`
	Settings       map[string]any `yaml:"settings"`
	StreamSettings map[string]any `yaml:"streamSettings"`
	Sniffing       map[string]any `yaml:"sniffing"`
	Allocate       map[string]any `yaml:"allocate"`
}

func LoadFile(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse inbound yaml: %w", err)
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return &spec, nil
}

func (s *Spec) Validate() error {
	if s.Remark == "" {
		return fmt.Errorf("remark is required")
	}
	if s.Protocol == "" {
		return fmt.Errorf("protocol is required")
	}
	if s.Port <= 0 || s.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if s.Enable == nil {
		v := true
		s.Enable = &v
	}
	if s.Settings == nil {
		s.Settings = map[string]any{}
	}
	if s.StreamSettings == nil {
		s.StreamSettings = map[string]any{}
	}
	return nil
}

func (s *Spec) SettingsJSON() (string, error) {
	b, err := json.Marshal(s.Settings)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Spec) StreamSettingsJSON() (string, error) {
	b, err := json.Marshal(s.StreamSettings)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Spec) SniffingJSON() (string, error) {
	if s.Sniffing == nil {
		return "", nil
	}
	b, err := json.Marshal(s.Sniffing)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Spec) AllocateJSON() (string, error) {
	if s.Allocate == nil {
		return "", nil
	}
	b, err := json.Marshal(s.Allocate)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
