package deploy

import (
	"context"
	"strings"
	"testing"
)

// stubProvider satisfies Provider for selection tests. Only Name and
// Available are exercised here; other methods panic to make misuse obvious.
type stubProvider struct {
	name      string
	available bool
}

func (s *stubProvider) Name() string                            { return s.name }
func (s *stubProvider) Available(context.Context) bool          { return s.available }
func (s *stubProvider) HasSSHKey(context.Context, string) (bool, error) {
	panic("unexpected HasSSHKey call in selection test")
}
func (s *stubProvider) UploadSSHKey(context.Context, string, string) (string, error) {
	panic("unexpected UploadSSHKey call in selection test")
}
func (s *stubProvider) Deploy(context.Context, DeployOpts) (*Deployment, error) {
	panic("unexpected Deploy call in selection test")
}

func TestSelectProvider_ExplicitMatch(t *testing.T) {
	ctx := context.Background()
	do := &stubProvider{name: "do", available: true}
	hz := &stubProvider{name: "hetzner", available: true}

	got, err := SelectProvider(ctx, "hetzner", []Provider{do, hz})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "hetzner" {
		t.Errorf("got %s, want hetzner", got.Name())
	}
}

func TestSelectProvider_ExplicitUnknown(t *testing.T) {
	ctx := context.Background()
	do := &stubProvider{name: "do", available: true}

	_, err := SelectProvider(ctx, "aws", []Provider{do})
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if !strings.Contains(err.Error(), "aws") {
		t.Errorf("error should mention unknown name, got: %v", err)
	}
}

func TestSelectProvider_ExplicitUnauthenticated(t *testing.T) {
	ctx := context.Background()
	do := &stubProvider{name: "do", available: false}

	_, err := SelectProvider(ctx, "do", []Provider{do})
	if err == nil {
		t.Fatal("expected error when explicit provider is not authenticated, got nil")
	}
	if !strings.Contains(err.Error(), "doctl") {
		t.Errorf("error should mention doctl install/auth, got: %v", err)
	}
}

func TestSelectProvider_AutoSingleAvailable(t *testing.T) {
	ctx := context.Background()
	do := &stubProvider{name: "do", available: false}
	hz := &stubProvider{name: "hetzner", available: true}

	got, err := SelectProvider(ctx, "", []Provider{do, hz})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "hetzner" {
		t.Errorf("got %s, want hetzner", got.Name())
	}
}

func TestSelectProvider_AutoBothAvailable_Errors(t *testing.T) {
	ctx := context.Background()
	do := &stubProvider{name: "do", available: true}
	hz := &stubProvider{name: "hetzner", available: true}

	_, err := SelectProvider(ctx, "", []Provider{do, hz})
	if err == nil {
		t.Fatal("expected error when both providers are available and none requested")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--provider") || !strings.Contains(msg, "do") || !strings.Contains(msg, "hetzner") {
		t.Errorf("error should list ambiguous providers and mention --provider flag, got: %v", err)
	}
}

func TestSelectProvider_AutoNoneAvailable(t *testing.T) {
	ctx := context.Background()
	do := &stubProvider{name: "do", available: false}
	hz := &stubProvider{name: "hetzner", available: false}

	_, err := SelectProvider(ctx, "", []Provider{do, hz})
	if err == nil {
		t.Fatal("expected error when no providers are available")
	}
	if !strings.Contains(err.Error(), "doctl") || !strings.Contains(err.Error(), "hcloud") {
		t.Errorf("error should mention both CLIs to install, got: %v", err)
	}
}
