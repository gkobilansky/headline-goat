package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type HetznerProvider struct {
	run runner
	// keyCache maps MD5 fingerprint → key name. hcloud's server create
	// requires a key name/ID rather than a fingerprint, so we list once and
	// cache the mapping across HasSSHKey → Deploy.
	keyCache map[string]string
}

func NewHetzner() *HetznerProvider { return &HetznerProvider{run: execRunner} }

func (*HetznerProvider) Name() string { return ProviderHetzner }

func (p *HetznerProvider) Available(ctx context.Context) bool {
	out, err := p.run(ctx, "hcloud", "context", "active")
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

type hcloudSSHKey struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
}

func (p *HetznerProvider) refreshKeyCache(ctx context.Context) error {
	out, err := p.run(ctx, "hcloud", "ssh-key", "list", "-o", "json")
	if err != nil {
		return fmt.Errorf("list ssh keys: %w", err)
	}
	var keys []hcloudSSHKey
	if err := json.Unmarshal(out, &keys); err != nil {
		return fmt.Errorf("parse ssh-key list: %w", err)
	}
	p.keyCache = make(map[string]string, len(keys))
	for _, k := range keys {
		p.keyCache[k.Fingerprint] = k.Name
	}
	return nil
}

func (p *HetznerProvider) HasSSHKey(ctx context.Context, fingerprint string) (bool, error) {
	if err := p.refreshKeyCache(ctx); err != nil {
		return false, err
	}
	_, ok := p.keyCache[fingerprint]
	return ok, nil
}

func (p *HetznerProvider) UploadSSHKey(ctx context.Context, name, pubkey string) (string, error) {
	path, cleanup, err := writeTempFile("hlg-pubkey-*.pub", pubkey+"\n")
	if err != nil {
		return "", err
	}
	defer cleanup()

	out, err := p.run(ctx, "hcloud", "ssh-key", "create",
		"--name", name,
		"--public-key-from-file", path,
		"-o", "json")
	if err != nil {
		return "", fmt.Errorf("create ssh key: %w", err)
	}
	var key hcloudSSHKey
	if err := json.Unmarshal(out, &key); err != nil {
		return "", fmt.Errorf("parse ssh-key create: %w", err)
	}

	fp := key.Fingerprint
	if fp == "" {
		// Some hcloud versions omit the fingerprint from create output.
		fp, err = ComputeFingerprintMD5(pubkey)
		if err != nil {
			return "", fmt.Errorf("parse ssh-key create: no fingerprint and %w", err)
		}
	}
	if p.keyCache == nil {
		p.keyCache = map[string]string{}
	}
	p.keyCache[fp] = key.Name
	return fp, nil
}

type hcloudServerCreate struct {
	Server struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
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

	keyName, ok := p.keyCache[opts.SSHKeyFP]
	if !ok {
		// Cold path: caller went straight to Deploy without HasSSHKey/Upload.
		if err := p.refreshKeyCache(ctx); err != nil {
			return nil, err
		}
		keyName = p.keyCache[opts.SSHKeyFP]
	}
	if keyName == "" {
		return nil, fmt.Errorf("ssh key with fingerprint %s not found on hetzner account", opts.SSHKeyFP)
	}

	userDataPath, cleanup, err := writeTempFile("hlg-userdata-*.yaml", opts.UserData)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	args := []string{
		"server", "create",
		"--name", opts.Name,
		"--type", serverType,
		"--image", image,
		"--location", location,
		"--ssh-key", keyName,
		"--user-data-from-file", userDataPath,
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
		Provider: ProviderHetzner,
		VMID:     fmt.Sprintf("%d", result.Server.ID),
		Host:     result.Server.PublicNet.IPv4.IP,
		User:     "hlg",
		Region:   location,
		Size:     serverType,
	}, nil
}
