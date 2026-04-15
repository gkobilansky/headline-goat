// Package deploy provisions a headline-goat VPS via cloud provider CLIs
// (doctl, hcloud) so users never need to SSH into the machine manually.
package deploy

import "context"

// Provider is a cloud provider capable of creating a VPS for hlg.
type Provider interface {
	// Name returns the short identifier ("do", "hetzner").
	Name() string

	// Available reports whether the provider's CLI is installed and
	// authenticated on this machine. Must not error — treat any failure
	// as "not available".
	Available(ctx context.Context) bool

	// HasSSHKey reports whether a pubkey with the given fingerprint is
	// registered with the provider account. The fingerprint must be in
	// the canonical SHA256 form "SHA256:base64hash".
	HasSSHKey(ctx context.Context, fingerprint string) (bool, error)

	// UploadSSHKey registers a new pubkey with the provider account under
	// the given name. Returns the provider-assigned key identifier (which
	// may be the fingerprint, an ID, or the name, depending on provider).
	UploadSSHKey(ctx context.Context, name, pubkey string) (string, error)

	// Deploy creates a VPS. Returns the public IP and provider VM ID on
	// success. Must wait for the VM to reach a running state (IP
	// assigned) before returning — cloud-init completion is NOT required.
	Deploy(ctx context.Context, opts DeployOpts) (*Deployment, error)
}

// DeployOpts describes the VPS to create. All fields are optional; providers
// substitute defaults for empty values.
type DeployOpts struct {
	Name     string // VM name (default: "hlg-<timestamp>")
	Region   string // provider-specific region slug
	Size     string // provider-specific size slug
	Image    string // provider-specific image slug (default: Ubuntu 22.04)
	SSHKeyFP string // SHA256 fingerprint of the local pubkey to authorize
	UserData string // cloud-init script (passed to provider's user_data)
}

// Deployment is the result of a successful Deploy call.
type Deployment struct {
	Provider string // "do", "hetzner"
	VMID     string // provider's VM identifier (for later destroy)
	Host     string // public IPv4 address
	User     string // SSH user to connect as (typically "hlg" after cloud-init)
	Region   string
	Size     string
}
