package deploy

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ComputeFingerprint returns the canonical SSH SHA-256 fingerprint
// ("SHA256:<base64>", unpadded) of the pubkey's binary blob.
//
// The input is the one-line authorized_keys form:
//   ssh-ed25519 <base64blob> [comment]
//
// Only the blob participates in the hash — the comment is ignored, matching
// `ssh-keygen -lf` behavior.
func ComputeFingerprint(pubkey string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(pubkey))
	if len(fields) < 2 {
		return "", fmt.Errorf("invalid pubkey: expected \"<type> <blob> [comment]\"")
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return "", fmt.Errorf("invalid base64 in pubkey: %w", err)
	}
	sum := sha256.Sum256(blob)
	// OpenSSH uses unpadded base64.
	return "SHA256:" + strings.TrimRight(base64.StdEncoding.EncodeToString(sum[:]), "="), nil
}

// FindLocalPubkey returns the path to a local SSH pubkey to authorize on
// new VMs. Explicit path wins; otherwise tries ~/.ssh/id_ed25519.pub then
// ~/.ssh/id_rsa.pub.
func FindLocalPubkey(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("ssh key %s: %w", explicit, err)
		}
		return explicit, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}

	// Prefer ed25519 (modern, shorter, faster) over RSA.
	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519.pub"),
		filepath.Join(home, ".ssh", "id_rsa.pub"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no SSH pubkey found at %s. Generate one with `ssh-keygen -t ed25519` or pass --ssh-key", strings.Join(candidates, " or "))
}

// ComputeFingerprintMD5 returns the colon-separated MD5 hex fingerprint of
// the pubkey blob. This is the format both `doctl` and `hcloud` report in
// their `fingerprint` field.
func ComputeFingerprintMD5(pubkey string) (string, error) {
	fields := strings.Fields(strings.TrimSpace(pubkey))
	if len(fields) < 2 {
		return "", fmt.Errorf("invalid pubkey: expected \"<type> <blob> [comment]\"")
	}
	blob, err := base64.StdEncoding.DecodeString(fields[1])
	if err != nil {
		return "", fmt.Errorf("invalid base64 in pubkey: %w", err)
	}
	sum := md5.Sum(blob)
	hex := hex.EncodeToString(sum[:])
	var b strings.Builder
	for i := 0; i < len(hex); i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteString(hex[i : i+2])
	}
	return b.String(), nil
}

// ReadPubkey loads and trims a pubkey file from disk.
func ReadPubkey(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
