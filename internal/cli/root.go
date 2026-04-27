package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath string

	// Flags affecting remote-mode dispatch. Parsed by cobra for help/discovery,
	// but actual interception happens in Execute() before cobra runs.
	remoteFlag bool
	localFlag  bool
)

// remoteCapableCommands lists subcommands whose data lives in the SQLite
// store and therefore make sense to forward over SSH to a remote server.
// Commands like "init" (start server), "deploy" (provision VPS), and
// "help"/"version" are intentionally excluded.
var remoteCapableCommands = map[string]bool{
	"create":  true,
	"list":    true,
	"results": true,
	"winner":  true,
	"export":  true,
	"token":   true,
}

var rootCmd = &cobra.Command{
	Use:   "hlg",
	Short: "Headline Goat - A minimal, self-hosted A/B testing tool for headlines",
	Long: `🐐 Headline Goat is a minimal, self-hosted A/B testing tool for headlines.
Single Go binary, embedded SQLite, no external dependencies.

Running without a subcommand starts the server (same as 'hlg init').

Remote mode:
  Data commands (list, results, create, winner, export, token) can run against
  a remote server over SSH. Remote is activated automatically when hlg.json
  exists in the current repo, or via --remote / HLG_REMOTE=user@host.
  Pass --local to force local execution.`,
	RunE: runInit, // Default action is to start server
}

// Execute is the CLI entry point. Before delegating to cobra it checks
// whether the invocation should be forwarded to a remote host over SSH.
func Execute() error {
	args := os.Args[1:]

	if shouldRunRemote(args) {
		remote, err := ResolveRemote()
		if err != nil {
			fmt.Fprintf(os.Stderr, "hlg: resolve remote: %v\n", err)
			return err
		}
		if remote == nil {
			if hasFlag(args, "--remote") {
				err := fmt.Errorf("--remote set but no hlg.json, ~/.hlg/config.json, or HLG_REMOTE env found")
				fmt.Fprintf(os.Stderr, "hlg: %v\n", err)
				return err
			}
			// No remote configured — fall through to local execution.
		} else {
			return runRemote(remote, stripRemoteFlags(args))
		}
	}

	return rootCmd.Execute()
}

// shouldRunRemote decides whether the current invocation is a candidate for
// SSH forwarding. It does NOT read config files — that happens later in
// ResolveRemote() — so this is purely based on argv.
func shouldRunRemote(args []string) bool {
	if hasFlag(args, "--local") {
		return false
	}

	cmd, _, err := rootCmd.Find(args)
	if err != nil || cmd == nil {
		return false
	}
	if !remoteCapableCommands[cmd.Name()] {
		return false
	}

	// Explicit opt-in always wins.
	if hasFlag(args, "--remote") || os.Getenv("HLG_REMOTE") != "" {
		return true
	}

	// Implicit: if any config file is discoverable we assume remote intent.
	// ResolveRemote() will return nil if nothing is actually configured, in
	// which case Execute() falls back to local.
	return true
}

// hasFlag reports whether name appears in args as a standalone flag or as
// "name=value". Matches both "--remote" and "--remote=foo".
func hasFlag(args []string, name string) bool {
	prefix := name + "="
	for _, a := range args {
		if a == name || (len(a) > len(prefix) && a[:len(prefix)] == prefix) {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", getEnvOrDefault("HG_DB_PATH", "./hlg.db"), "database path")
	rootCmd.PersistentFlags().BoolVar(&remoteFlag, "remote", false, "run against a remote hlg server over SSH (uses hlg.json or ~/.hlg/config.json)")
	rootCmd.PersistentFlags().BoolVar(&localFlag, "local", false, "force local execution even if hlg.json is present")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
