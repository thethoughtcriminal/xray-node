package inbound

import "testing"

func TestApplyOverridesPortAndSNI(t *testing.T) {
	spec := &Spec{
		Remark:   "vless-reality",
		Protocol: "vless",
		Port:     443,
		StreamSettings: map[string]any{
			"network":  "tcp",
			"security": "reality",
			"realitySettings": map[string]any{
				"dest": "www.microsoft.com:443",
				"serverNames": []any{
					"www.microsoft.com",
				},
			},
		},
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	if err := spec.ApplyOverrides(Overrides{Port: 8443, SNI: "deepl.com"}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if spec.Port != 8443 {
		t.Fatalf("port=%d", spec.Port)
	}
	if spec.Tag != "in-8443-tcp" {
		t.Fatalf("tag=%q", spec.Tag)
	}
	if spec.DefaultSNI() != "deepl.com" {
		t.Fatalf("sni=%q", spec.DefaultSNI())
	}
	rs := realitySettings(spec)
	if rs["target"] != "deepl.com:443" {
		t.Fatalf("target=%v", rs["target"])
	}
}
