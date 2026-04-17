package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	shellquote "github.com/kballard/go-shellquote"
)

// Remote describes how to reach a headline-goat server over SSH.
// Loaded from hlg.json (per-repo) or ~/.hlg/config.json (global).
type Remote struct {
	Host     string `json:"host"`
	User     string `json:"user,omitempty"`
	Port     int    `json:"port,omitempty"`
	Identity string `json:"identity,omitempty"` // private key path (ssh -i)

	// Populated by `hlg deploy`. Not used for SSH dispatch; here so future
	// lifecycle commands (destroy, status) can target the underlying VM
	// without a separate state file.
	Provider string `json:"provider,omitempty"`
	VMID     string `json:"vm_id,omitempty"`
}

const (
	localConfigName  = "hlg.json"
	globalConfigPath = ".hlg/config.json" // relative to $HOME
)

// parseRemoteString parses "[user@]host[:port]" into a Remote.
func parseRemoteString(s string) (*Remote, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty remote string")
	}

	r := &Remote{}
	if at := strings.Index(s, "@"); at >= 0 {
		r.User = s[:at]
		s = s[at+1:]
	}
	if colon := strings.Index(s, ":"); colon >= 0 {
		port, err := strconv.Atoi(s[colon+1:])
		if err != nil {
			return nil, fmt.Errorf("invalid port in %q: %w", s, err)
		}
		r.Port = port
		s = s[:colon]
	}
	if s == "" {
		return nil, fmt.Errorf("empty host")
	}
	r.Host = s
	return r, nil
}

// loadRemoteConfig reads and validates a config file at path.
func loadRemoteConfig(path string) (*Remote, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var r Remote
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if strings.TrimSpace(r.Host) == "" {
		return nil, fmt.Errorf("%s: \"host\" is required", path)
	}
	return &r, nil
}

// findLocalConfig walks up from startDir looking for hlg.json.
// Returns the full path if found, or "" if none exists.
func findLocalConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, localConfigName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil // reached filesystem root
		}
		dir = parent
	}
}

// globalConfigFile returns ~/.hlg/config.json (or "" if $HOME unset).
func globalConfigFile() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, globalConfigPath)
}

// ResolveRemote determines the effective remote config based on precedence:
//  1. HLG_REMOTE env var (parsed as "[user@]host[:port]")
//  2. ./hlg.json walking up from cwd
//  3. ~/.hlg/config.json
//
// Returns nil, nil when no remote is configured (i.e. run locally).
func ResolveRemote() (*Remote, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return resolveRemoteFrom(cwd, os.Getenv("HLG_REMOTE"))
}

// resolveRemoteFrom is the testable core of ResolveRemote.
func resolveRemoteFrom(cwd, envVal string) (*Remote, error) {
	if strings.TrimSpace(envVal) != "" {
		return parseRemoteString(envVal)
	}

	if path, err := findLocalConfig(cwd); err != nil {
		return nil, err
	} else if path != "" {
		return loadRemoteConfig(path)
	}

	if gp := globalConfigFile(); gp != "" {
		r, err := loadRemoteConfig(gp)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return r, err
	}

	return nil, nil
}

// buildSSHArgs constructs the argv (after "ssh") for invoking `hlg` on a
// remote host. The remote command is assembled as a single shell-quoted
// string so the remote shell receives the exact same argv we were given.
func buildSSHArgs(r *Remote, hlgArgs []string) []string {
	args := []string{}
	if r.Identity != "" {
		args = append(args, "-i", r.Identity)
	}
	if r.Port != 0 {
		args = append(args, "-p", strconv.Itoa(r.Port))
	}

	target := r.Host
	if r.User != "" {
		target = r.User + "@" + r.Host
	}
	args = append(args, target)

	args = append(args, shellquote.Join(append([]string{"hlg"}, hlgArgs...)...))
	return args
}

// stripRemoteFlags removes --remote/--local (and --remote=... / --local=...)
// from the argv before forwarding to the remote host. Without this, the
// remote `hlg` invocation would try to recursively ssh to itself.
func stripRemoteFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--remote" || a == "--local" {
			continue
		}
		if strings.HasPrefix(a, "--remote=") || strings.HasPrefix(a, "--local=") {
			continue
		}
		out = append(out, a)
	}
	return out
}

// runRemote execs `ssh ... 'hlg <args>'` with stdio streamed through.
// Returns the remote process's exit code behavior via cmd.Run's error.
func runRemote(r *Remote, hlgArgs []string) error {
	sshArgs := buildSSHArgs(r, hlgArgs)
	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Surface invocation problems (e.g. ssh not installed) since main.go
		// doesn't print cobra errors for bypassed commands. For remote exit
		// codes, ssh already printed the remote stderr to our stderr.
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Fprintf(os.Stderr, "hlg: remote exec failed: %v\n", err)
		}
		return err
	}
	return nil
}
