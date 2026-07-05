package inbound

import "testing"

func TestSpecValidate(t *testing.T) {
	spec := &Spec{
		Remark:   "test",
		Protocol: "vless",
		Port:     443,
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if spec.Enable == nil || !*spec.Enable {
		t.Fatalf("expected enable default true")
	}
}

func TestSpecValidateMissingRemark(t *testing.T) {
	spec := &Spec{Protocol: "vless", Port: 443}
	if err := spec.Validate(); err == nil {
		t.Fatal("expected error")
	}
}
