package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gkobilansky/headline-goat/internal/deploy"
)

func TestWriteRemoteConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	d := &deploy.Deployment{
		Provider: "hetzner",
		VMID:     "12345",
		Host:     "1.2.3.4",
		User:     "hlg",
		Region:   "nbg1",
		Size:     "cpx11",
	}
	if err := writeRemoteConfig(d); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	path := filepath.Join(home, ".hlg", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}

	var r Remote
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if r.Host != "1.2.3.4" {
		t.Errorf("host = %q, want 1.2.3.4", r.Host)
	}
	if r.User != "hlg" {
		t.Errorf("user = %q, want hlg", r.User)
	}
	if r.Provider != "hetzner" {
		t.Errorf("provider = %q, want hetzner", r.Provider)
	}
	if r.VMID != "12345" {
		t.Errorf("vm_id = %q, want 12345", r.VMID)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permissions = %o, want 0600", perm)
	}
}
