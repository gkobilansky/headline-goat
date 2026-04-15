package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runner is the subprocess escape hatch — overridden in tests to avoid
// requiring doctl/hcloud to be installed.
type runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// execRunner is the production runner. Uses CombinedOutput so callers get
// stderr messages on failure (doctl's errors are on stderr).
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

// DOProvider creates DigitalOcean droplets via the doctl CLI.
type DOProvider struct {
	run runner
}

func NewDO() *DOProvider { return &DOProvider{run: execRunner} }

func (*DOProvider) Name() string { return "do" }

func (p *DOProvider) Available(ctx context.Context) bool {
	_, err := p.run(ctx, "doctl", "account", "get", "-o", "json")
	return err == nil
}

// doSSHKey mirrors the subset of doctl's ssh-key JSON output we care about.
type doSSHKey struct {
	ID          int    `json:"id"`
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"name"`
}

// HasSSHKey checks if a key with the given MD5 fingerprint is registered.
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

// UploadSSHKey imports a pubkey into the DO account. Writes the pubkey to a
// temp file because doctl's import subcommand requires --public-key-file.
func (p *DOProvider) UploadSSHKey(ctx context.Context, name, pubkey string) (string, error) {
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

	out, err := p.run(ctx, "doctl", "compute", "ssh-key", "import", name,
		"--public-key-file", f.Name(), "-o", "json")
	if err != nil {
		return "", fmt.Errorf("import ssh key: %w", err)
	}
	// doctl returns an array even for single-key output.
	var keys []doSSHKey
	if err := json.Unmarshal(out, &keys); err != nil {
		return "", fmt.Errorf("parse import output: %w", err)
	}
	if len(keys) == 0 {
		return "", fmt.Errorf("doctl returned no key info")
	}
	return keys[0].Fingerprint, nil
}

// doDroplet mirrors the relevant subset of doctl's droplet JSON.
type doDroplet struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Networks struct {
		V4 []struct {
			IPAddress string `json:"ip_address"`
			Type      string `json:"type"` // "public" or "private"
		} `json:"v4"`
	} `json:"networks"`
}

// Deploy creates a droplet and blocks until DO reports it active (--wait).
// Returns the public IPv4 address.
func (p *DOProvider) Deploy(ctx context.Context, opts DeployOpts) (*Deployment, error) {
	region := firstNonEmpty(opts.Region, "nyc1")
	size := firstNonEmpty(opts.Size, "s-1vcpu-1gb")
	image := firstNonEmpty(opts.Image, "ubuntu-22-04-x64")

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
		"compute", "droplet", "create", opts.Name,
		"--region", region,
		"--size", size,
		"--image", image,
		"--ssh-keys", opts.SSHKeyFP,
		"--user-data-file", userDataFile.Name(),
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
		Provider: "do",
		VMID:     fmt.Sprintf("%d", d.ID),
		Host:     ip,
		User:     "hlg",
		Region:   region,
		Size:     size,
	}, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
