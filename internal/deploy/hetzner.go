package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// HetznerProvider creates Hetzner Cloud servers via the hcloud CLI.
type HetznerProvider struct {
	run runner
}

func NewHetzner() *HetznerProvider { return &HetznerProvider{run: execRunner} }

func (*HetznerProvider) Name() string { return "hetzner" }

// Available reports whether hcloud is installed and has an active context.
// `hcloud context active` prints the context name and exits 0 on success.
func (p *HetznerProvider) Available(ctx context.Context) bool {
	out, err := p.run(ctx, "hcloud", "context", "active")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

// hcloudSSHKey mirrors the relevant fields of hcloud's ssh-key JSON.
type hcloudSSHKey struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
}

// listSSHKeys fetches all registered keys. Shared by HasSSHKey and Deploy
// (Deploy needs the name because hcloud's --ssh-key flag doesn't accept
// fingerprints directly).
func (p *HetznerProvider) listSSHKeys(ctx context.Context) ([]hcloudSSHKey, error) {
	out, err := p.run(ctx, "hcloud", "ssh-key", "list", "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("list ssh keys: %w", err)
	}
	var keys []hcloudSSHKey
	if err := json.Unmarshal(out, &keys); err != nil {
		return nil, fmt.Errorf("parse ssh-key list: %w", err)
	}
	return keys, nil
}

func (p *HetznerProvider) HasSSHKey(ctx context.Context, fingerprint string) (bool, error) {
	keys, err := p.listSSHKeys(ctx)
	if err != nil {
		return false, err
	}
	for _, k := range keys {
		if k.Fingerprint == fingerprint {
			return true, nil
		}
	}
	return false, nil
}

func (p *HetznerProvider) UploadSSHKey(ctx context.Context, name, pubkey string) (string, error) {
	f, err := os.CreateTemp("", "hlg-pubkey-*.pub")
	if err != nil {
		return "", fmt.Errorf("tempfile: %w", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(pubkey + "\n"); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	out, err := p.run(ctx, "hcloud", "ssh-key", "create",
		"--name", name,
		"--public-key-from-file", f.Name(),
		"-o", "json")
	if err != nil {
		return "", fmt.Errorf("create ssh key: %w", err)
	}
	var key hcloudSSHKey
	if err := json.Unmarshal(out, &key); err != nil {
		return "", fmt.Errorf("parse ssh-key create: %w", err)
	}
	if key.Fingerprint == "" {
		// Some hcloud versions return the fingerprint separately; fall back
		// to MD5-hashing the submitted pubkey.
		fp, ferr := ComputeFingerprintMD5(pubkey)
		if ferr != nil {
			return "", fmt.Errorf("parse ssh-key create: no fingerprint and %w", ferr)
		}
		return fp, nil
	}
	return key.Fingerprint, nil
}

// hcloudServerCreate mirrors the JSON returned by `hcloud server create -o json`.
// Note the outer "server" key — this differs from doctl's array-of-droplets.
type hcloudServerCreate struct {
	Server struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Status   string `json:"status"`
		PublicNet struct {
			IPv4 struct {
				IP string `json:"ip"`
			} `json:"ipv4"`
		} `json:"public_net"`
	} `json:"server"`
}

func (p *HetznerProvider) Deploy(ctx context.Context, opts DeployOpts) (*Deployment, error) {
	location := firstNonEmpty(opts.Region, "nbg1")
	serverType := firstNonEmpty(opts.Size, "cpx11")
	image := firstNonEmpty(opts.Image, "ubuntu-22.04")

	// Translate fingerprint → key name (hcloud --ssh-key requires name/ID).
	keys, err := p.listSSHKeys(ctx)
	if err != nil {
		return nil, err
	}
	var sshKeyName string
	for _, k := range keys {
		if k.Fingerprint == opts.SSHKeyFP {
			sshKeyName = k.Name
			break
		}
	}
	if sshKeyName == "" {
		return nil, fmt.Errorf("ssh key with fingerprint %s not found on hetzner account", opts.SSHKeyFP)
	}

	userDataFile, err := os.CreateTemp("", "hlg-userdata-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("tempfile: %w", err)
	}
	defer os.Remove(userDataFile.Name())
	if _, err := userDataFile.WriteString(opts.UserData); err != nil {
		userDataFile.Close()
		return nil, err
	}
	userDataFile.Close()

	args := []string{
		"server", "create",
		"--name", opts.Name,
		"--type", serverType,
		"--image", image,
		"--location", location,
		"--ssh-key", sshKeyName,
		"--user-data-from-file", userDataFile.Name(),
		"-o", "json",
	}

	out, err := p.run(ctx, "hcloud", args...)
	if err != nil {
		return nil, fmt.Errorf("create server: %w", err)
	}

	var result hcloudServerCreate
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("parse server create: %w", err)
	}
	if result.Server.PublicNet.IPv4.IP == "" {
		return nil, fmt.Errorf("server %d has no public IPv4 address", result.Server.ID)
	}

	return &Deployment{
		Provider: "hetzner",
		VMID:     fmt.Sprintf("%d", result.Server.ID),
		Host:     result.Server.PublicNet.IPv4.IP,
		User:     "hlg",
		Region:   location,
		Size:     serverType,
	}, nil
}
