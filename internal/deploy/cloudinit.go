package deploy

import (
	"fmt"
	"strings"
)

// DefaultReleaseURL points at the latest linux/amd64 release binary.
// TODO: gate to a specific tag once releases are stable.
const DefaultReleaseURL = "https://github.com/gkobilansky/headline-goat/releases/latest/download/hlg-linux-amd64"

// CloudInitOpts configures the generated cloud-init script.
type CloudInitOpts struct {
	ReleaseURL string // URL to download the hlg binary from
	Port       int    // listening port (defaults to 8080 if zero)
}

// GenerateCloudInit returns a cloud-init user-data YAML script that:
//  1. Creates a non-root hlg user with sudo disabled
//  2. Downloads the hlg binary to /usr/local/bin/hlg
//  3. Installs a systemd unit that runs hlg as the hlg user
//  4. Opens the listening port through ufw if present
//  5. Enables and starts the service
//
// The user's SSH pubkey is NOT embedded here — providers install it via their
// own ssh_keys mechanism so fingerprints stay canonical per-provider.
func GenerateCloudInit(opts CloudInitOpts) string {
	port := opts.Port
	if port == 0 {
		port = 8080
	}
	url := opts.ReleaseURL
	if url == "" {
		url = DefaultReleaseURL
	}

	var b strings.Builder
	fmt.Fprintln(&b, "#cloud-config")
	fmt.Fprintln(&b, "users:")
	fmt.Fprintln(&b, "  - name: hlg")
	fmt.Fprintln(&b, "    shell: /bin/bash")
	fmt.Fprintln(&b, "    sudo: false")
	fmt.Fprintln(&b, "    lock_passwd: true")
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "write_files:")
	fmt.Fprintln(&b, "  - path: /etc/systemd/system/hlg.service")
	fmt.Fprintln(&b, "    permissions: '0644'")
	fmt.Fprintln(&b, "    content: |")
	fmt.Fprintln(&b, "      [Unit]")
	fmt.Fprintln(&b, "      Description=Headline Goat A/B Testing Server")
	fmt.Fprintln(&b, "      After=network-online.target")
	fmt.Fprintln(&b, "      Wants=network-online.target")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "      [Service]")
	fmt.Fprintln(&b, "      User=hlg")
	fmt.Fprintln(&b, "      WorkingDirectory=/home/hlg")
	fmt.Fprintln(&b, "      Environment=HG_DB_PATH=/home/hlg/hlg.db")
	fmt.Fprintf(&b, "      Environment=HG_PORT=%d\n", port)
	fmt.Fprintln(&b, "      ExecStart=/usr/local/bin/hlg")
	fmt.Fprintln(&b, "      Restart=always")
	fmt.Fprintln(&b, "      RestartSec=3")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "      [Install]")
	fmt.Fprintln(&b, "      WantedBy=multi-user.target")
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "runcmd:")
	fmt.Fprintf(&b, "  - curl -fsSL %s -o /usr/local/bin/hlg\n", url)
	fmt.Fprintln(&b, "  - chmod +x /usr/local/bin/hlg")
	fmt.Fprintln(&b, "  - install -d -o hlg -g hlg /home/hlg")
	fmt.Fprintf(&b, "  - which ufw && ufw allow %d/tcp || true\n", port)
	fmt.Fprintln(&b, "  - systemctl daemon-reload")
	fmt.Fprintln(&b, "  - systemctl enable --now hlg")

	return b.String()
}
