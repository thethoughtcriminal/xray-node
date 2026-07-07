package inbound

import "testing"

func TestParseXrayX25519Output(t *testing.T) {
	text := `PrivateKey: gHz35R_Uip69Rcqw8FtupsZRhxG4XX_HLvk2MWYwYlk
Password (PublicKey): ujoInMV7niI43wRJWEM_hm-kkbebhc-42ZoubuTBzTQ
Hash32: UKfuBdWalFSyAA0_9xPjYJbs7CUXCm7e1PcIWGkZaC8`
	priv, pub, err := parseXrayX25519Output(text)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if priv != "gHz35R_Uip69Rcqw8FtupsZRhxG4XX_HLvk2MWYwYlk" {
		t.Fatalf("private=%q", priv)
	}
	if pub != "ujoInMV7niI43wRJWEM_hm-kkbebhc-42ZoubuTBzTQ" {
		t.Fatalf("public=%q", pub)
	}
}

func TestEnsureRealityKeysPreservesExisting(t *testing.T) {
	spec := &Spec{
		Remark:   "vless-reality",
		Protocol: "vless",
		Port:     443,
		StreamSettings: map[string]any{
			"security": "reality",
			"realitySettings": map[string]any{
				"privateKey": "existing-private",
				"settings": map[string]any{
					"publicKey": "existing-public",
				},
			},
		},
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if err := spec.EnsureRealityKeys(); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	rs := realitySettings(spec)
	if rs["privateKey"] != "existing-private" {
		t.Fatalf("private overwritten")
	}
}
