package deploy

import (
	"strings"
	"testing"
)

func TestGenerateCloudInit_HasShebangAndHeader(t *testing.T) {
	got := GenerateCloudInit(CloudInitOpts{
		ReleaseURL: "https://example.com/hlg-linux-amd64",
		Port:       8080,
	})
	if !strings.HasPrefix(got, "#cloud-config") {
		t.Errorf("cloud-init must start with #cloud-config, got first line: %q", firstLine(got))
	}
}

func TestGenerateCloudInit_CreatesHLGUser(t *testing.T) {
	got := GenerateCloudInit(CloudInitOpts{ReleaseURL: "x", Port: 8080})
	for _, want := range []string{"users:", "- name: hlg", "shell: /bin/bash"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in cloud-init:\n%s", want, got)
		}
	}
}

func TestGenerateCloudInit_DownloadsBinary(t *testing.T) {
	url := "https://github.com/foo/bar/releases/download/v1.0.0/hlg-linux-amd64"
	got := GenerateCloudInit(CloudInitOpts{ReleaseURL: url, Port: 8080})
	if !strings.Contains(got, url) {
		t.Errorf("release URL %q missing from cloud-init", url)
	}
	// Must chmod +x the binary so systemd can exec it.
	if !strings.Contains(got, "chmod +x /usr/local/bin/hlg") {
		t.Errorf("expected chmod of hlg binary, got:\n%s", got)
	}
}

func TestGenerateCloudInit_WritesSystemdUnit(t *testing.T) {
	got := GenerateCloudInit(CloudInitOpts{ReleaseURL: "x", Port: 8080})
	for _, want := range []string{
		"/etc/systemd/system/hlg.service",
		"ExecStart=/usr/local/bin/hlg",
		"Restart=always",
		"User=hlg",
		"[Install]",
		"WantedBy=multi-user.target",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in cloud-init:\n%s", want, got)
		}
	}
}

func TestGenerateCloudInit_SetsPortEnv(t *testing.T) {
	got := GenerateCloudInit(CloudInitOpts{ReleaseURL: "x", Port: 9090})
	if !strings.Contains(got, "HG_PORT=9090") {
		t.Errorf("expected HG_PORT=9090 in cloud-init:\n%s", got)
	}
}

func TestGenerateCloudInit_EnablesService(t *testing.T) {
	got := GenerateCloudInit(CloudInitOpts{ReleaseURL: "x", Port: 8080})
	if !strings.Contains(got, "systemctl enable --now hlg") {
		t.Errorf("expected systemctl enable --now, got:\n%s", got)
	}
}

func TestGenerateCloudInit_OpensFirewall(t *testing.T) {
	got := GenerateCloudInit(CloudInitOpts{ReleaseURL: "x", Port: 8080})
	// Fresh Ubuntu clouds usually have no firewall, but if ufw is present
	// we should at least open the port so the VM is reachable.
	if !strings.Contains(got, "ufw allow 8080") {
		t.Errorf("expected ufw rule for port, got:\n%s", got)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
