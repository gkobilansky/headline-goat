package deploy

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestHetzner_Available_TrueWhenContextActive(t *testing.T) {
	f := &fakeRunner{responses: map[string]fakeResponse{
		"hcloud context active": {stdout: []byte("default\n")},
	}}
	p := &HetznerProvider{run: f.run}
	if !p.Available(context.Background()) {
		t.Error("expected Available=true when context is active")
	}
}

func TestHetzner_Available_FalseOnEmptyOutput(t *testing.T) {
	f := &fakeRunner{responses: map[string]fakeResponse{
		"hcloud context active": {stdout: []byte("")},
	}}
	p := &HetznerProvider{run: f.run}
	if p.Available(context.Background()) {
		t.Error("expected Available=false with empty context")
	}
}

func TestHetzner_Available_FalseOnError(t *testing.T) {
	f := &fakeRunner{responses: map[string]fakeResponse{
		"hcloud context active": {err: fmt.Errorf("not found")},
	}}
	p := &HetznerProvider{run: f.run}
	if p.Available(context.Background()) {
		t.Error("expected Available=false on error")
	}
}

func TestHetzner_HasSSHKey_Match(t *testing.T) {
	listJSON := `[
		{"id":1,"name":"alice","fingerprint":"aa:bb:cc"},
		{"id":2,"name":"bob","fingerprint":"dd:ee:ff"}
	]`
	f := &fakeRunner{responses: map[string]fakeResponse{
		"hcloud ssh-key list": {stdout: []byte(listJSON)},
	}}
	p := &HetznerProvider{run: f.run}

	found, err := p.HasSSHKey(context.Background(), "dd:ee:ff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected match")
	}
}

func TestHetzner_UploadSSHKey_ReturnsFingerprint(t *testing.T) {
	createJSON := `{"id":99,"name":"hlg-key","fingerprint":"11:22:33"}`
	f := &fakeRunner{responses: map[string]fakeResponse{
		"hcloud ssh-key create": {stdout: []byte(createJSON)},
	}}
	p := &HetznerProvider{run: f.run}

	fp, err := p.UploadSSHKey(context.Background(), "hlg-key", "ssh-ed25519 AAAA test")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if fp != "11:22:33" {
		t.Errorf("got %q, want 11:22:33", fp)
	}
}

func TestHetzner_Deploy_TranslatesFingerprintToKeyName(t *testing.T) {
	listJSON := `[{"id":7,"name":"my-laptop","fingerprint":"aa:bb:cc"}]`
	createJSON := `{"server":{"id":12345,"status":"running","public_net":{"ipv4":{"ip":"5.6.7.8"}}}}`

	f := &fakeRunner{responses: map[string]fakeResponse{
		"hcloud ssh-key list":   {stdout: []byte(listJSON)},
		"hcloud server create": {stdout: []byte(createJSON)},
	}}
	p := &HetznerProvider{run: f.run}

	got, err := p.Deploy(context.Background(), DeployOpts{
		Name:     "hlg-test",
		SSHKeyFP: "aa:bb:cc",
		UserData: "#cloud-config",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if got.Host != "5.6.7.8" {
		t.Errorf("got host %q, want 5.6.7.8", got.Host)
	}
	if got.VMID != "12345" {
		t.Errorf("got VMID %q, want 12345", got.VMID)
	}
	if got.Provider != "hetzner" {
		t.Errorf("got provider %q, want hetzner", got.Provider)
	}

	// Second call = server create. Must reference the key by NAME, not fingerprint.
	createCall := f.calls[1]
	args := strings.Join(createCall.args, " ")
	if !strings.Contains(args, "--ssh-key my-laptop") {
		t.Errorf("expected --ssh-key my-laptop (translated), got: %s", args)
	}
	// Must pass the default Hetzner type/location/image.
	for _, want := range []string{"--type cpx11", "--location nbg1", "--image ubuntu-22.04"} {
		if !strings.Contains(args, want) {
			t.Errorf("missing %q in args: %s", want, args)
		}
	}
}

func TestHetzner_Deploy_FingerprintNotRegistered(t *testing.T) {
	f := &fakeRunner{responses: map[string]fakeResponse{
		"hcloud ssh-key list": {stdout: []byte(`[]`)},
	}}
	p := &HetznerProvider{run: f.run}

	_, err := p.Deploy(context.Background(), DeployOpts{
		Name:     "x",
		SSHKeyFP: "aa:bb:cc",
	})
	if err == nil {
		t.Fatal("expected error when fingerprint is not in account")
	}
	if !strings.Contains(err.Error(), "aa:bb:cc") {
		t.Errorf("error should include fingerprint: %v", err)
	}
}

func TestHetzner_HasSSHKeyThenDeploy_SingleListCall(t *testing.T) {
	listJSON := `[{"id":7,"name":"my-laptop","fingerprint":"aa:bb:cc"}]`
	createJSON := `{"server":{"id":1,"public_net":{"ipv4":{"ip":"1.1.1.1"}}}}`

	f := &fakeRunner{responses: map[string]fakeResponse{
		"hcloud ssh-key list":  {stdout: []byte(listJSON)},
		"hcloud server create": {stdout: []byte(createJSON)},
	}}
	p := &HetznerProvider{run: f.run}

	if _, err := p.HasSSHKey(context.Background(), "aa:bb:cc"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Deploy(context.Background(), DeployOpts{
		Name: "hlg", SSHKeyFP: "aa:bb:cc",
	}); err != nil {
		t.Fatal(err)
	}

	// The cache set up by HasSSHKey must let Deploy skip a second ssh-key list.
	listCalls := 0
	for _, c := range f.calls {
		if len(c.args) >= 2 && c.args[0] == "ssh-key" && c.args[1] == "list" {
			listCalls++
		}
	}
	if listCalls != 1 {
		t.Errorf("expected 1 ssh-key list call (cache reused), got %d", listCalls)
	}
}

func TestHetzner_Deploy_UsesCustomValues(t *testing.T) {
	listJSON := `[{"id":1,"name":"k","fingerprint":"x"}]`
	createJSON := `{"server":{"id":1,"public_net":{"ipv4":{"ip":"1.1.1.1"}}}}`
	f := &fakeRunner{responses: map[string]fakeResponse{
		"hcloud ssh-key list":   {stdout: []byte(listJSON)},
		"hcloud server create": {stdout: []byte(createJSON)},
	}}
	p := &HetznerProvider{run: f.run}

	_, err := p.Deploy(context.Background(), DeployOpts{
		Name:     "hlg",
		Region:   "hel1",
		Size:     "cx21",
		Image:    "debian-12",
		SSHKeyFP: "x",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	args := strings.Join(f.calls[1].args, " ")
	for _, want := range []string{"hel1", "cx21", "debian-12"} {
		if !strings.Contains(args, want) {
			t.Errorf("expected %q in args: %s", want, args)
		}
	}
}
