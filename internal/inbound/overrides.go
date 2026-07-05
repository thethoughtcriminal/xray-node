package inbound

import (
	"fmt"
	"strings"
)

// Overrides are optional values collected interactively or from CLI flags.
type Overrides struct {
	Port int
	SNI  string
}

func (s *Spec) IsRealityVLESS() bool {
	if s.Protocol != "vless" {
		return false
	}
	security, _ := s.StreamSettings["security"].(string)
	return security == "reality"
}

func (s *Spec) DefaultSNI() string {
	rs := realitySettings(s)
	if rs == nil {
		return ""
	}
	if sns, ok := rs["serverNames"].([]any); ok && len(sns) > 0 {
		if name, ok := sns[0].(string); ok && name != "" {
			return name
		}
	}
	for _, key := range []string{"target", "dest"} {
		value, _ := rs[key].(string)
		if value == "" {
			continue
		}
		if host, _, ok := strings.Cut(value, ":"); ok && host != "" {
			return host
		}
		return value
	}
	return ""
}

func (s *Spec) ApplyOverrides(o Overrides) error {
	if o.Port > 0 {
		if o.Port > 65535 {
			return fmt.Errorf("port must be between 1 and 65535")
		}
		s.Port = o.Port
		s.Tag = tagForPort(s)
	}
	if o.SNI != "" {
		if !s.IsRealityVLESS() {
			return fmt.Errorf("sni override is only supported for vless reality inbounds")
		}
		if err := applySNI(s, o.SNI); err != nil {
			return err
		}
	}
	return s.Validate()
}

func tagForPort(s *Spec) string {
	network, _ := s.StreamSettings["network"].(string)
	if network == "hysteria" {
		return fmt.Sprintf("in-%d-udp", s.Port)
	}
	return fmt.Sprintf("in-%d-tcp", s.Port)
}

func applySNI(s *Spec, sni string) error {
	sni = strings.TrimSpace(sni)
	if sni == "" {
		return fmt.Errorf("sni is required")
	}
	if strings.Contains(sni, ":") {
		return fmt.Errorf("sni must be a hostname without port")
	}

	rs := realitySettings(s)
	if rs == nil {
		return fmt.Errorf("realitySettings not found in inbound spec")
	}

	target := sni + ":443"
	rs["target"] = target
	rs["dest"] = target
	rs["serverNames"] = []any{sni}
	s.StreamSettings["realitySettings"] = rs
	return nil
}

func realitySettings(s *Spec) map[string]any {
	raw, ok := s.StreamSettings["realitySettings"]
	if !ok || raw == nil {
		return nil
	}
	rs, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return rs
}
