package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// runner is the subprocess escape hatch — overridden in tests to avoid
// requiring doctl/hcloud to be installed.
type runner func(ctx context.Context, name string, args ...string) ([]byte, error)

func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return out, fmt.Errorf("%s: %s", name, strings.TrimSpace(string(ee.Stderr)))
		}
		return out, err
	}
	return out, nil
}

type DOProvider struct {
	run runner
}

func NewDO() *DOProvider { return &DOProvider{run: execRunner} }

func (*DOProvider) Name() string { return ProviderDO }

func (p *DOProvider) Available(ctx context.Context) bool {
	_, err := p.run(ctx, "doctl", "account", "get", "-o", "json")
	return err == nil
}

type doSSHKey struct {
	ID          int    `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"name"`
}

func (p *DOProvider) HasSSHKey(ctx context.Context, fingerprint string) (bool, error) {
	out, err := p.run(ctx, "doctl", "compute", "ssh-key", "list", "-o", "json")
	if err != nil {
		return false, fmt.Errorf("list ssh keys: %w", err)
	}
	var keys []doSSHKey
	if err := json.Unmarshal(out, &keys); err != nil {
		return false, fmt.Errorf("parse ssh-key list: %w", err)
	}
	for _, k := range keys {
		if k.Fingerprint == fingerprint {
			return true, nil
		}
	}
	return false, nil
}

func (p *DOProvider) UploadSSHKey(ctx context.Context, name, pubkey string) (string, error) {
	path, cleanup, err := writeTempFile("hlg-pubkey-*.pub", pubkey+"\n")
	if err != nil {
		return "", err
	}
	defer cleanup()

	out, err := p.run(ctx, "doctl", "compute", "ssh-key", "import", name,
		"--public-key-file", path, "-o", "json")
	if err != nil {
		return "", fmt.Errorf("import ssh key: %w", err)
	}
	// doctl wraps even a single result in an array.
	var keys []doSSHKey
	if err := json.Unmarshal(out, &keys); err != nil {
		return "", fmt.Errorf("parse import output: %w", err)
	}
	if len(keys) == 0 {
		return "", fmt.Errorf("doctl returned no key info")
	}
	return keys[0].Fingerprint, nil
}

type doDroplet struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Networks struct {
		V4 []struct {
			IPAddress string `json:"ip_address"`
			Type      string `json:"type"`
		} `json:"v4"`
	} `json:"networks"`
}

func (p *DOProvider) Deploy(ctx context.Context, opts DeployOpts) (*Deployment, error) {
	region := firstNonEmpty(opts.Region, "nyc1")
	size := firstNonEmpty(opts.Size, "s-1vcpu-1gb")
	image := firstNonEmpty(opts.Image, "ubuntu-22-04-x64")

	userDataPath, cleanup, err := writeTempFile("hlg-userdata-*.yaml", opts.UserData)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	args := []string{
		"compute", "droplet", "create", opts.Name,
		"--region", region,
		"--size", size,
		"--image", image,
		"--ssh-keys", opts.SSHKeyFP,
		"--user-data-file", userDataPath,
		"--wait",
		"-o", "json",
	}

	out, err := p.run(ctx, "doctl", args...)
	if err != nil {
		return nil, fmt.Errorf("create droplet: %w", err)
	}

	var droplets []doDroplet
	if err := json.Unmarshal(out, &droplets); err != nil {
		return nil, fmt.Errorf("parse droplet output: %w", err)
	}
	if len(droplets) == 0 {
		return nil, fmt.Errorf("doctl returned no droplet info")
	}
	d := droplets[0]

	var ip string
	for _, n := range d.Networks.V4 {
		if n.Type == "public" {
			ip = n.IPAddress
			break
		}
	}
	if ip == "" {
		return nil, fmt.Errorf("droplet %d has no public IPv4 address", d.ID)
	}

	return &Deployment{
		Provider: ProviderDO,
		VMID:     fmt.Sprintf("%d", d.ID),
		Host:     ip,
		User:     "hlg",
		Region:   region,
		Size:     size,
	}, nil
}
