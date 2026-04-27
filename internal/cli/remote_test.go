package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseRemoteString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Remote
		wantErr bool
	}{
		{
			name:  "host only",
			input: "example.com",
			want:  &Remote{Host: "example.com"},
		},
		{
			name:  "user@host",
			input: "hlg@example.com",
			want:  &Remote{Host: "example.com", User: "hlg"},
		},
		{
			name:  "user@host:port",
			input: "hlg@example.com:2222",
			want:  &Remote{Host: "example.com", User: "hlg", Port: 2222},
		},
		{
			name:  "host:port",
			input: "example.com:2222",
			want:  &Remote{Host: "example.com", Port: 2222},
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid port",
			input:   "example.com:abc",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseRemoteString(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (result: %+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestLoadRemoteConfig_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hlg.json")
	content := `{"host":"vps.example.com","user":"hlg","port":22}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := loadRemoteConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := &Remote{Host: "vps.example.com", User: "hlg", Port: 22}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadRemoteConfig_MissingHost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hlg.json")
	if err := os.WriteFile(path, []byte(`{"user":"hlg"}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadRemoteConfig(path)
	if err == nil {
		t.Fatal("expected error for config missing host, got nil")
	}
}

func TestLoadRemoteConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hlg.json")
	if err := os.WriteFile(path, []byte(`not json`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := loadRemoteConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestFindLocalConfig_InCurrentDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hlg.json")
	if err := os.WriteFile(path, []byte(`{"host":"x"}`), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := findLocalConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != path {
		t.Errorf("got %s, want %s", got, path)
	}
}

func TestFindLocalConfig_InParentDir(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "hlg.json")
	if err := os.WriteFile(configPath, []byte(`{"host":"x"}`), 0644); err != nil {
		t.Fatal(err)
	}

	child := filepath.Join(root, "sub", "nested")
	if err := os.MkdirAll(child, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := findLocalConfig(child)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != configPath {
		t.Errorf("got %s, want %s", got, configPath)
	}
}

func TestFindLocalConfig_NotFound(t *testing.T) {
	dir := t.TempDir()
	got, err := findLocalConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty path, got %s", got)
	}
}

func TestBuildSSHArgs_HostOnly(t *testing.T) {
	r := &Remote{Host: "vps.example.com"}
	got := buildSSHArgs(r, []string{"results", "hero", "--json"})
	want := []string{"vps.example.com", "hlg results hero --json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildSSHArgs_WithUser(t *testing.T) {
	r := &Remote{Host: "vps.example.com", User: "hlg"}
	got := buildSSHArgs(r, []string{"list"})
	want := []string{"hlg@vps.example.com", "hlg list"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildSSHArgs_WithPort(t *testing.T) {
	r := &Remote{Host: "vps.example.com", User: "hlg", Port: 2222}
	got := buildSSHArgs(r, []string{"list"})
	want := []string{"-p", "2222", "hlg@vps.example.com", "hlg list"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildSSHArgs_WithIdentity(t *testing.T) {
	r := &Remote{Host: "vps.example.com", Identity: "/home/me/.ssh/hlg_key"}
	got := buildSSHArgs(r, []string{"list"})
	want := []string{"-i", "/home/me/.ssh/hlg_key", "vps.example.com", "hlg list"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildSSHArgs_QuotesArgsWithSpaces(t *testing.T) {
	r := &Remote{Host: "vps.example.com"}
	got := buildSSHArgs(r, []string{"create", "hero", "--variants", "A,B C,D"})
	// The arg "A,B C,D" contains a space and must be shell-quoted.
	want := []string{"vps.example.com", `hlg create hero --variants 'A,B C,D'`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestStripRemoteFlags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "no flag",
			in:   []string{"results", "hero"},
			want: []string{"results", "hero"},
		},
		{
			name: "--remote alone",
			in:   []string{"results", "--remote", "hero"},
			want: []string{"results", "hero"},
		},
		{
			name: "--remote=value",
			in:   []string{"results", "--remote=user@host", "hero"},
			want: []string{"results", "hero"},
		},
		{
			name: "--local also stripped",
			in:   []string{"results", "--local", "hero"},
			want: []string{"results", "hero"},
		},
		{
			name: "preserves other flags",
			in:   []string{"results", "hero", "--json", "--remote"},
			want: []string{"results", "hero", "--json"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripRemoteFlags(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveRemote_Precedence(t *testing.T) {
	// Set up: global config + local config. Local should win.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HLG_REMOTE", "")

	globalDir := filepath.Join(home, ".hlg")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "config.json"), []byte(`{"host":"global.example.com"}`), 0644); err != nil {
		t.Fatal(err)
	}

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "hlg.json"), []byte(`{"host":"local.example.com"}`), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveRemoteFrom(repo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Host != "local.example.com" {
		t.Errorf("expected local.example.com to win, got %+v", got)
	}

	// Env var beats local config
	got, err = resolveRemoteFrom(repo, "env.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Host != "env.example.com" {
		t.Errorf("expected env.example.com to win, got %+v", got)
	}
}

func TestResolveRemote_None(t *testing.T) {
	home := t.TempDir() // empty home, no .hlg/config.json
	t.Setenv("HOME", home)

	repo := t.TempDir() // empty repo, no hlg.json

	got, err := resolveRemoteFrom(repo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil remote, got %+v", got)
	}
}

func TestHasFlag(t *testing.T) {
	tests := []struct {
		args []string
		name string
		want bool
	}{
		{[]string{"results", "hero"}, "--remote", false},
		{[]string{"results", "--remote", "hero"}, "--remote", true},
		{[]string{"results", "--remote=user@host"}, "--remote", true},
		{[]string{"results", "--local"}, "--local", true},
		{[]string{"results", "--json"}, "--remote", false},
	}
	for _, tc := range tests {
		got := hasFlag(tc.args, tc.name)
		if got != tc.want {
			t.Errorf("hasFlag(%v, %q) = %v, want %v", tc.args, tc.name, got, tc.want)
		}
	}
}

func TestShouldRunRemote(t *testing.T) {
	// Isolate from real env / filesystem.
	t.Setenv("HLG_REMOTE", "")
	t.Setenv("HOME", t.TempDir())
	t.Chdir(t.TempDir())

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"local flag forces off", []string{"results", "hero", "--local"}, false},
		{"unknown command is not remote", []string{"bogus"}, false},
		{"init command never remote", []string{"init"}, false},
		{"help not remote", []string{"--help"}, false},
		{"results with --remote", []string{"results", "hero", "--remote"}, true},
		{"results without flag (auto-detect)", []string{"results", "hero"}, true},
		{"create is remote-capable", []string{"create", "hero", "--variants", "A,B"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldRunRemote(tc.args)
			if got != tc.want {
				t.Errorf("shouldRunRemote(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

func TestResolveRemote_GlobalOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	globalDir := filepath.Join(home, ".hlg")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "config.json"), []byte(`{"host":"global.example.com"}`), 0644); err != nil {
		t.Fatal(err)
	}

	repo := t.TempDir() // no local config
	got, err := resolveRemoteFrom(repo, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.Host != "global.example.com" {
		t.Errorf("expected global.example.com, got %+v", got)
	}
}
