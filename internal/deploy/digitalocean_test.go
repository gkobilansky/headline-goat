package deploy

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// fakeRunner captures each call and returns predetermined responses.
type fakeRunner struct {
	// responses maps the first argument ("doctl"/"hcloud" + first sub-arg key)
	// to (stdout, error). Lookup is by joining args with space.
	responses map[string]fakeResponse
	calls     []fakeCall
}

type fakeResponse struct {
	stdout []byte
	err    error
}

type fakeCall struct {
	name string
	args []string
}

func (f *fakeRunner) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, fakeCall{name: name, args: append([]string(nil), args...)})

	key := name + " " + strings.Join(args, " ")
	for k, v := range f.responses {
		if strings.HasPrefix(key, k) {
			return v.stdout, v.err
		}
	}
	return nil, fmt.Errorf("fakeRunner: no response for %q", key)
}

func TestDO_Available_TrueWhenAccountGetSucceeds(t *testing.T) {
	f := &fakeRunner{responses: map[string]fakeResponse{
		"doctl account get": {stdout: []byte(`{"email":"x"}`)},
	}}
	p := &DOProvider{run: f.run}
	if !p.Available(context.Background()) {
		t.Error("expected Available=true when doctl account get succeeds")
	}
}

func TestDO_Available_FalseOnError(t *testing.T) {
	f := &fakeRunner{responses: map[string]fakeResponse{
		"doctl account get": {err: fmt.Errorf("not authenticated")},
	}}
	p := &DOProvider{run: f.run}
	if p.Available(context.Background()) {
		t.Error("expected Available=false on auth error")
	}
}

func TestDO_HasSSHKey_Match(t *testing.T) {
	listJSON := `[
		{"id":111,"fingerprint":"aa:bb:cc","name":"one"},
		{"id":222,"fingerprint":"dd:ee:ff","name":"two"}
	]`
	f := &fakeRunner{responses: map[string]fakeResponse{
		"doctl compute ssh-key list": {stdout: []byte(listJSON)},
	}}
	p := &DOProvider{run: f.run}

	found, err := p.HasSSHKey(context.Background(), "dd:ee:ff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected HasSSHKey=true for matching fingerprint")
	}
}

func TestDO_HasSSHKey_NoMatch(t *testing.T) {
	f := &fakeRunner{responses: map[string]fakeResponse{
		"doctl compute ssh-key list": {stdout: []byte(`[]`)},
	}}
	p := &DOProvider{run: f.run}

	found, err := p.HasSSHKey(context.Background(), "aa:bb:cc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected HasSSHKey=false for missing fingerprint")
	}
}

func TestDO_UploadSSHKey_ReturnsFingerprint(t *testing.T) {
	response := `[{"id":999,"fingerprint":"11:22:33","name":"hlg-key"}]`
	f := &fakeRunner{responses: map[string]fakeResponse{
		"doctl compute ssh-key import": {stdout: []byte(response)},
	}}
	p := &DOProvider{run: f.run}

	fp, err := p.UploadSSHKey(context.Background(), "hlg-key", "ssh-ed25519 AAAA test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp != "11:22:33" {
		t.Errorf("got %q, want %q", fp, "11:22:33")
	}
	// Sanity check: a temp file path should have been passed to doctl.
	call := f.calls[0]
	foundFlag := false
	for i, a := range call.args {
		if a == "--public-key-file" && i+1 < len(call.args) {
			foundFlag = true
			if !strings.HasSuffix(call.args[i+1], ".pub") {
				t.Errorf("pubkey file path should end in .pub, got %s", call.args[i+1])
			}
		}
	}
	if !foundFlag {
		t.Error("expected --public-key-file in doctl args")
	}
}

func TestDO_Deploy_ExtractsPublicIP(t *testing.T) {
	dropletJSON := `[{
		"id": 12345,
		"name": "hlg-test",
		"status": "active",
		"networks": {
			"v4": [
				{"ip_address": "10.0.0.1", "type": "private"},
				{"ip_address": "1.2.3.4", "type": "public"}
			]
		}
	}]`
	f := &fakeRunner{responses: map[string]fakeResponse{
		"doctl compute droplet create": {stdout: []byte(dropletJSON)},
	}}
	p := &DOProvider{run: f.run}

	got, err := p.Deploy(context.Background(), DeployOpts{
		Name:     "hlg-test",
		SSHKeyFP: "aa:bb:cc",
		UserData: "#cloud-config",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Host != "1.2.3.4" {
		t.Errorf("got host %q, want 1.2.3.4", got.Host)
	}
	if got.VMID != "12345" {
		t.Errorf("got VMID %q, want 12345", got.VMID)
	}
	if got.Provider != "do" {
		t.Errorf("got provider %q, want do", got.Provider)
	}
	if got.User != "hlg" {
		t.Errorf("got user %q, want hlg", got.User)
	}

	// Verify the args we passed to doctl.
	if len(f.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(f.calls))
	}
	args := strings.Join(f.calls[0].args, " ")
	for _, want := range []string{"--wait", "--ssh-keys aa:bb:cc", "-o json", "nyc1", "s-1vcpu-1gb", "ubuntu-22-04-x64"} {
		if !strings.Contains(args, want) {
			t.Errorf("doctl args missing %q: %s", want, args)
		}
	}
}

func TestDO_Deploy_UsesCustomRegionAndSize(t *testing.T) {
	f := &fakeRunner{responses: map[string]fakeResponse{
		"doctl compute droplet create": {stdout: []byte(`[{"id":1,"networks":{"v4":[{"ip_address":"1.1.1.1","type":"public"}]}}]`)},
	}}
	p := &DOProvider{run: f.run}

	_, err := p.Deploy(context.Background(), DeployOpts{
		Name:     "hlg",
		Region:   "sfo3",
		Size:     "s-2vcpu-2gb",
		SSHKeyFP: "x",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	args := strings.Join(f.calls[0].args, " ")
	if !strings.Contains(args, "sfo3") {
		t.Errorf("expected sfo3 in args: %s", args)
	}
	if !strings.Contains(args, "s-2vcpu-2gb") {
		t.Errorf("expected custom size in args: %s", args)
	}
}

func TestDO_Deploy_NoPublicIP(t *testing.T) {
	f := &fakeRunner{responses: map[string]fakeResponse{
		"doctl compute droplet create": {stdout: []byte(`[{"id":1,"networks":{"v4":[]}}]`)},
	}}
	p := &DOProvider{run: f.run}

	_, err := p.Deploy(context.Background(), DeployOpts{Name: "hlg", SSHKeyFP: "x"})
	if err == nil {
		t.Fatal("expected error when droplet has no public IP")
	}
}
