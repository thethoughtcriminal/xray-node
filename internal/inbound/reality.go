package inbound

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	realityPrivateKeyLine = regexp.MustCompile(`(?m)^PrivateKey:\s*(\S+)`)
	realityPublicKeyLine  = regexp.MustCompile(`(?m)^Password \(PublicKey\):\s*(\S+)`)
)

func (s *Spec) HasRealityKeys() bool {
	rs := realitySettings(s)
	if rs == nil {
		return false
	}
	if pk, _ := rs["privateKey"].(string); pk != "" {
		return true
	}
	settings, _ := rs["settings"].(map[string]any)
	if pub, _ := settings["publicKey"].(string); pub != "" {
		return true
	}
	return false
}

func (s *Spec) MergeRealityKeysFromStream(streamJSON string) error {
	if streamJSON == "" {
		return nil
	}
	var stream map[string]any
	if err := decodeJSONField(streamJSON, &stream); err != nil {
		return err
	}
	raw, ok := stream["realitySettings"]
	if !ok || raw == nil {
		return nil
	}
	existing, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	return mergeRealityKeys(s, existing)
}

func mergeRealityKeys(s *Spec, existing map[string]any) error {
	rs := realitySettings(s)
	if rs == nil {
		return nil
	}
	if pk, _ := rs["privateKey"].(string); pk == "" {
		if v, _ := existing["privateKey"].(string); v != "" {
			rs["privateKey"] = v
		}
	}
	dstSettings, _ := rs["settings"].(map[string]any)
	if dstSettings == nil {
		dstSettings = map[string]any{}
		rs["settings"] = dstSettings
	}
	if pub, _ := dstSettings["publicKey"].(string); pub == "" {
		if srcSettings, ok := existing["settings"].(map[string]any); ok {
			if v, _ := srcSettings["publicKey"].(string); v != "" {
				dstSettings["publicKey"] = v
			}
		}
	}
	s.StreamSettings["realitySettings"] = rs
	return nil
}

// EnsureRealityKeys generates x25519 keys via xray when missing.
func (s *Spec) EnsureRealityKeys() error {
	if !s.IsRealityVLESS() || s.HasRealityKeys() {
		return nil
	}
	privateKey, publicKey, err := generateRealityKeyPair()
	if err != nil {
		return err
	}
	rs := realitySettings(s)
	if rs == nil {
		return fmt.Errorf("realitySettings not found in inbound spec")
	}
	rs["privateKey"] = privateKey
	settings, _ := rs["settings"].(map[string]any)
	if settings == nil {
		settings = map[string]any{}
		rs["settings"] = settings
	}
	settings["publicKey"] = publicKey
	s.StreamSettings["realitySettings"] = rs
	return nil
}

func generateRealityKeyPair() (privateKey, publicKey string, err error) {
	xray, err := findXrayBinary()
	if err != nil {
		return "", "", err
	}
	out, err := exec.Command(xray, "x25519").CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("xray x25519: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return parseXrayX25519Output(string(out))
}

func findXrayBinary() (string, error) {
	if p, err := exec.LookPath("xray"); err == nil {
		return p, nil
	}
	for _, candidate := range []string{
		"/usr/local/x-ui/bin/xray-linux-amd64",
		"/usr/local/x-ui/bin/xray-linux-arm64",
		"/usr/local/x-ui/bin/xray-linux-armv7",
	} {
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("xray binary not found (install 3x-ui first)")
}

func decodeJSONField(raw string, out any) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal([]byte(raw), &s); err != nil {
			return err
		}
		raw = s
	}
	return json.Unmarshal([]byte(raw), out)
}

func parseXrayX25519Output(text string) (privateKey, publicKey string, err error) {
	if m := realityPrivateKeyLine.FindStringSubmatch(text); len(m) == 2 {
		privateKey = m[1]
	}
	if m := realityPublicKeyLine.FindStringSubmatch(text); len(m) == 2 {
		publicKey = m[1]
	}
	if privateKey == "" || publicKey == "" {
		return "", "", fmt.Errorf("missing keys in xray output")
	}
	return privateKey, publicKey, nil
}
