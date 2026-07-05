package panel

import (
	"encoding/json"
	"testing"
)

func TestJSONFieldUnmarshalString(t *testing.T) {
	var inbound Inbound
	raw := `{"id":1,"settings":"{\"clients\":[]}"}`
	if err := json.Unmarshal([]byte(raw), &inbound); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if inbound.Settings.String() != `{"clients":[]}` {
		t.Fatalf("settings=%q", inbound.Settings)
	}
}

func TestJSONFieldUnmarshalObject(t *testing.T) {
	var inbound Inbound
	raw := `{"id":1,"settings":{"clients":[],"decryption":"none"}}`
	if err := json.Unmarshal([]byte(raw), &inbound); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if inbound.Settings.String() == "" {
		t.Fatal("expected settings")
	}
	var settings ClientSettings
	if err := json.Unmarshal([]byte(inbound.Settings.String()), &settings); err != nil {
		t.Fatalf("parse settings: %v", err)
	}
}
