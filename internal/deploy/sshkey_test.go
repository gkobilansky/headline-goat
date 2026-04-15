package deploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Known-good ed25519 pubkey (generated for tests, not a real key).
// The SHA256 fingerprint for this blob is deterministic.
const testEd25519Pubkey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHvhyaS6MzCfBO8UbjQ6t9Vn3iN5L8DPLqE1k2w3Y4hZ test@localhost"

func TestComputeFingerprint_Ed25519(t *testing.T) {
	fp, err := ComputeFingerprint(testEd25519Pubkey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(fp, "SHA256:") {
		t.Errorf("fingerprint should start with SHA256:, got %q", fp)
	}
	// SHA-256 → 32 bytes → 43 chars base64 (unpadded). "SHA256:" is 7 chars.
	if len(fp) != 7+43 {
		t.Errorf("expected fingerprint length 50, got %d (%q)", len(fp), fp)
	}
}

func TestComputeFingerprint_DifferentKeys(t *testing.T) {
	// Two different keys should produce different fingerprints.
	fp1, err := ComputeFingerprint("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA a")
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := ComputeFingerprint("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB a")
	if err != nil {
		t.Fatal(err)
	}
	if fp1 == fp2 {
		t.Errorf("different keys should produce different fingerprints, both = %s", fp1)
	}
}

func TestComputeFingerprint_StableAcrossComment(t *testing.T) {
	// The comment field (after the blob) must not affect the fingerprint.
	base := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHvhyaS6MzCfBO8UbjQ6t9Vn3iN5L8DPLqE1k2w3Y4hZ"
	fpA, err := ComputeFingerprint(base + " alice@laptop")
	if err != nil {
		t.Fatal(err)
	}
	fpB, err := ComputeFingerprint(base + " bob@desktop")
	if err != nil {
		t.Fatal(err)
	}
	if fpA != fpB {
		t.Errorf("comment should not affect fingerprint, got %s vs %s", fpA, fpB)
	}
}

func TestComputeFingerprintMD5_Format(t *testing.T) {
	fp, err := ComputeFingerprintMD5(testEd25519Pubkey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// MD5 = 16 bytes = 32 hex chars + 15 colons = 47 chars
	if len(fp) != 47 {
		t.Errorf("expected length 47, got %d (%q)", len(fp), fp)
	}
	// Must be all lowercase hex and colons, in aa:bb:cc:... shape.
	for i, c := range fp {
		if (i+1)%3 == 0 {
			if c != ':' {
				t.Errorf("expected colon at position %d, got %c", i, c)
			}
		} else if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("expected hex char at position %d, got %c", i, c)
		}
	}
}

func TestComputeFingerprintMD5_StableAcrossComment(t *testing.T) {
	base := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHvhyaS6MzCfBO8UbjQ6t9Vn3iN5L8DPLqE1k2w3Y4hZ"
	a, err := ComputeFingerprintMD5(base + " alice")
	if err != nil {
		t.Fatal(err)
	}
	b, err := ComputeFingerprintMD5(base + " bob")
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Errorf("comment should not affect fingerprint: %s vs %s", a, b)
	}
}

func TestComputeFingerprint_InvalidInput(t *testing.T) {
	tests := []string{
		"",
		"not-a-key",
		"ssh-ed25519", // missing blob
		"ssh-ed25519 not-base64!@#",
	}
	for _, in := range tests {
		if _, err := ComputeFingerprint(in); err == nil {
			t.Errorf("expected error for input %q, got nil", in)
		}
	}
}

func TestFindLocalPubkey_Ed25519Preferred(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	// Create both; ed25519 should win.
	ed := filepath.Join(sshDir, "id_ed25519.pub")
	rsa := filepath.Join(sshDir, "id_rsa.pub")
	if err := os.WriteFile(ed, []byte("ssh-ed25519 AAA ed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rsa, []byte("ssh-rsa AAA rsa"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := FindLocalPubkey("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != ed {
		t.Errorf("got %s, want %s", got, ed)
	}
}

func TestFindLocalPubkey_FallbackToRSA(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	rsa := filepath.Join(sshDir, "id_rsa.pub")
	if err := os.WriteFile(rsa, []byte("ssh-rsa AAA rsa"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := FindLocalPubkey("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != rsa {
		t.Errorf("got %s, want %s", got, rsa)
	}
}

func TestFindLocalPubkey_NoKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := FindLocalPubkey("")
	if err == nil {
		t.Fatal("expected error when no keys exist, got nil")
	}
}

func TestFindLocalPubkey_Override(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Even if home has other keys, override wins.
	sshDir := filepath.Join(home, ".ssh")
	_ = os.MkdirAll(sshDir, 0700)
	_ = os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"), []byte("ssh-ed25519 xxx"), 0644)

	explicit := filepath.Join(home, "my_key.pub")
	if err := os.WriteFile(explicit, []byte("ssh-ed25519 AAA explicit"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := FindLocalPubkey(explicit)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != explicit {
		t.Errorf("got %s, want %s", got, explicit)
	}
}

func TestFindLocalPubkey_OverrideMissing(t *testing.T) {
	_, err := FindLocalPubkey("/does/not/exist.pub")
	if err == nil {
		t.Fatal("expected error for missing override, got nil")
	}
}
