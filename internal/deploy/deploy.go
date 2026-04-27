// Package deploy provisions a headline-goat VPS via cloud provider CLIs
// (doctl, hcloud) so users never need to SSH into the machine manually.
package deploy

import "context"

const (
	ProviderDO      = "do"
	ProviderHetzner = "hetzner"
)

type Provider interface {
	Name() string

	// Available reports whether the provider's CLI is installed and
	// authenticated. Must not error — treat any failure as "not available".
	Available(ctx context.Context) bool

	// HasSSHKey reports whether a pubkey with the given MD5 fingerprint
	// (colon-separated hex, as returned by doctl/hcloud) is registered.
	HasSSHKey(ctx context.Context, fingerprint string) (bool, error)

	// UploadSSHKey registers a pubkey under name and returns its MD5
	// fingerprint. Safe to call when the key already exists only if the
	// provider deduplicates by name — otherwise callers must HasSSHKey first.
	UploadSSHKey(ctx context.Context, name, pubkey string) (string, error)

	// Deploy creates a VPS and blocks until the VM has a public IP. Does
	// NOT wait for cloud-init to finish installing hlg.
	Deploy(ctx context.Context, opts DeployOpts) (*Deployment, error)
}

type DeployOpts struct {
	Name     string
	Region   string
	Size     string
	Image    string
	SSHKeyFP string // MD5 fingerprint of an already-registered pubkey
	UserData string // cloud-init script
}

type Deployment struct {
	Provider string
	VMID     string
	Host     string
	User     string
	Region   string
	Size     string
}
