package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gkobilansky/headline-goat/internal/deploy"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	// Don't dump the flag usage on runtime errors (missing CLIs, network
	// failures, etc.) — those aren't misuse.
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

	// Local pubkey discovery.
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

	// Ensure the key is registered with the provider.
	has, err := p.HasSSHKey(ctx, fingerprint)
	if err != nil {
		return err
	}
	if !has {
		keyName := fmt.Sprintf("hlg-%s", hostname())
		if !deployOpts.yes {
			prompt := fmt.Sprintf("Pubkey %s is not registered with %s. Upload it as %q?", keyPath, p.Name(), keyName)
			if !confirm(prompt, true) {
				return fmt.Errorf("aborted by user")
			}
		}
		newFP, err := p.UploadSSHKey(ctx, keyName, pubkey)
		if err != nil {
			return fmt.Errorf("upload ssh key: %w", err)
		}
		fmt.Printf("Uploaded ssh key (%s)\n", newFP)
		fingerprint = newFP
	}

	// Build VM opts.
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

	// Persist so subsequent hlg commands dispatch over SSH automatically.
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

// writeRemoteConfig persists the deployment to ~/.hlg/config.json so the
// remote-dispatch logic in Execute() picks it up. Extra fields (provider,
// vm_id) are ignored by SSH dispatch but used by future lifecycle commands.
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

// confirm prompts on stdin. In non-interactive environments returns def.
func confirm(prompt string, def bool) bool {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return def
	}
	suffix := " [Y/n] "
	if !def {
		suffix = " [y/N] "
	}
	fmt.Print(prompt + suffix)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return def
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	if ans == "" {
		return def
	}
	return ans == "y" || ans == "yes"
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "deploy"
	}
	// Cloud provider key names often forbid dots and non-alphanum.
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
