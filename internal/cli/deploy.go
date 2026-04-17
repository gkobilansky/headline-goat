package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gkobilansky/headline-goat/internal/deploy"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Provision a VPS and deploy hlg to it",
	Long: `Provision a fresh VPS via your authenticated provider CLI (doctl or hcloud),
install hlg as a systemd service, and save the connection info to
~/.hlg/config.json so subsequent commands dispatch over SSH automatically.

Requires: either doctl or hcloud CLI installed and authenticated. If both
are authenticated, pass --provider to choose.

You will be prompted to confirm uploading your local SSH pubkey if it's
not already registered with the chosen provider.`,
	SilenceUsage: true,
	RunE:         runDeploy,
}

type deployFlags struct {
	provider string
	name     string
	region   string
	size     string
	sshKey   string
	port     int
	yes      bool
}

var deployOpts deployFlags

func init() {
	deployCmd.Flags().StringVar(&deployOpts.provider, "provider", "", "provider to use (do|hetzner); auto-detected if only one CLI is authenticated")
	deployCmd.Flags().StringVar(&deployOpts.name, "name", "", "VM name (default: hlg-<timestamp>)")
	deployCmd.Flags().StringVar(&deployOpts.region, "region", "", "provider region/location (default: nyc1 for DO, nbg1 for Hetzner)")
	deployCmd.Flags().StringVar(&deployOpts.size, "size", "", "VM size/type (default: s-1vcpu-1gb for DO, cpx11 for Hetzner)")
	deployCmd.Flags().StringVar(&deployOpts.sshKey, "ssh-key", "", "path to SSH pubkey to authorize (default: ~/.ssh/id_ed25519.pub or id_rsa.pub)")
	deployCmd.Flags().IntVar(&deployOpts.port, "port", 8080, "hlg server port")
	deployCmd.Flags().BoolVarP(&deployOpts.yes, "yes", "y", false, "skip confirmation prompts (for scripts/CI)")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	providers := []deploy.Provider{deploy.NewDO(), deploy.NewHetzner()}

	p, err := deploy.SelectProvider(ctx, deployOpts.provider, providers)
	if err != nil {
		return err
	}
	fmt.Printf("Using provider: %s\n", p.Name())

	keyPath, err := deploy.FindLocalPubkey(deployOpts.sshKey)
	if err != nil {
		return err
	}
	pubkey, err := deploy.ReadPubkey(keyPath)
	if err != nil {
		return fmt.Errorf("read ssh key: %w", err)
	}
	fingerprint, err := deploy.ComputeFingerprintMD5(pubkey)
	if err != nil {
		return fmt.Errorf("fingerprint ssh key: %w", err)
	}
	fmt.Printf("Using SSH key: %s (%s)\n", keyPath, fingerprint)

	has, err := p.HasSSHKey(ctx, fingerprint)
	if err != nil {
		return err
	}
	if !has {
		keyName := fmt.Sprintf("hlg-%s", hostname())
		if !deployOpts.yes {
			prompt := promptui.Prompt{
				Label:     fmt.Sprintf("Upload %s to %s as %q", keyPath, p.Name(), keyName),
				IsConfirm: true,
				Default:   "y",
			}
			if _, err := prompt.Run(); err != nil {
				if errors.Is(err, promptui.ErrAbort) {
					return fmt.Errorf("aborted by user")
				}
				return err
			}
		}
		newFP, err := p.UploadSSHKey(ctx, keyName, pubkey)
		if err != nil {
			return fmt.Errorf("upload ssh key: %w", err)
		}
		fmt.Printf("Uploaded ssh key (%s)\n", newFP)
		fingerprint = newFP
	}

	name := deployOpts.name
	if name == "" {
		name = fmt.Sprintf("hlg-%d", time.Now().Unix())
	}
	userData := deploy.GenerateCloudInit(deploy.CloudInitOpts{
		ReleaseURL: deploy.DefaultReleaseURL,
		Port:       deployOpts.port,
	})

	fmt.Printf("Creating VM %q on %s... (this may take a minute)\n", name, p.Name())
	dep, err := p.Deploy(ctx, deploy.DeployOpts{
		Name:     name,
		Region:   deployOpts.region,
		Size:     deployOpts.size,
		SSHKeyFP: fingerprint,
		UserData: userData,
	})
	if err != nil {
		return err
	}

	if err := writeRemoteConfig(dep); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write ~/.hlg/config.json: %v\n", err)
	}

	fmt.Println()
	fmt.Printf("Deployed: %s (%s, id %s, %s/%s)\n", dep.Host, dep.Provider, dep.VMID, dep.Region, dep.Size)
	fmt.Printf("Saved to ~/.hlg/config.json. Cloud-init needs ~60s to finish installing hlg.\n")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  hlg list                           # works from any dir (global config)\n")
	fmt.Printf("  cp ~/.hlg/config.json ./hlg.json   # pin this VPS to a project\n")
	fmt.Printf("  curl http://%s:%d/health           # verify it's up\n", dep.Host, deployOpts.port)
	return nil
}

func writeRemoteConfig(d *deploy.Deployment) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".hlg")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	r := Remote{
		Host:     d.Host,
		User:     d.User,
		Provider: d.Provider,
		VMID:     d.VMID,
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0600)
}

// hostname returns os.Hostname with non-alphanum chars replaced by '-',
// since cloud provider SSH key names forbid dots and most punctuation.
func hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "deploy"
	}
	var b strings.Builder
	for _, r := range h {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
