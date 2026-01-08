package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gkobilansky/headline-goat/internal/server"
	"github.com/gkobilansky/headline-goat/internal/store"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var port int

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Start the server and show setup instructions",
	Long: `Start the Headline Goat server and show integration instructions.

The server provides:
  - Global script at /hlg.js
  - Beacon endpoint for tracking events
  - Dashboard for viewing results

Tests auto-create when the first beacon arrives - no explicit setup needed.

Examples:
  hlg
  hlg init
  hlg init --port 3000`,
	RunE: runInit,
}

func init() {
	defaultPort := 8080
	if p := os.Getenv("HG_PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			defaultPort = parsed
		}
	}

	initCmd.Flags().IntVarP(&port, "port", "p", defaultPort, "port to listen on")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	// Open database first to check for existing settings
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer s.Close()

	ctx := cmd.Context()

	// Check for existing settings
	existingURL, _ := s.GetSetting(ctx, "server_url")
	existingFramework, _ := s.GetSetting(ctx, "framework")

	// Prompt for server URL
	serverURL, err := promptServerURL(existingURL, port)
	if err != nil {
		return err
	}

	// Prompt for framework
	framework, err := promptFramework(existingFramework)
	if err != nil {
		return err
	}

	// Store settings
	if err := s.SetSetting(ctx, "server_url", serverURL); err != nil {
		return fmt.Errorf("failed to save server URL: %w", err)
	}
	if err := s.SetSetting(ctx, "framework", framework); err != nil {
		return fmt.Errorf("failed to save framework: %w", err)
	}

	// Token file path (alongside database)
	tokenFile := filepath.Join(filepath.Dir(dbPath), ".hlg-token")

	// Create server
	srv := server.New(s, port, tokenFile)

	// Print startup message with instructions
	printStartupInstructions(framework, serverURL, port, srv.Token())

	// Start server quietly (we printed our own message)
	return srv.StartQuiet()
}

func promptServerURL(existing string, port int) (string, error) {
	defaultURL := fmt.Sprintf("http://localhost:%d", port)
	if existing != "" {
		defaultURL = existing
	}

	prompt := promptui.Prompt{
		Label:   fmt.Sprintf("Server URL for script tag [%s]", defaultURL),
		Default: defaultURL,
		Validate: func(input string) error {
			if input == "" {
				return nil // Allow empty, will use default
			}
			if !strings.HasPrefix(input, "http://") && !strings.HasPrefix(input, "https://") {
				return fmt.Errorf("must start with http:// or https://")
			}
			return nil
		},
	}

	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}

	// Use default if empty
	if result == "" {
		result = defaultURL
	}

	// Remove trailing slash if present
	return strings.TrimSuffix(result, "/"), nil
}

func promptFramework(existing string) (string, error) {
	frameworks := []string{
		"HTML (vanilla JavaScript)",
		"React / Next.js",
		"Vue",
		"Svelte",
		"Laravel (Blade)",
		"Django",
		"Other",
	}

	// Find cursor position for existing selection
	cursorPos := 0
	if existing != "" {
		switch existing {
		case "html":
			cursorPos = 0
		case "react":
			cursorPos = 1
		case "vue":
			cursorPos = 2
		case "svelte":
			cursorPos = 3
		case "laravel":
			cursorPos = 4
		case "django":
			cursorPos = 5
		case "other":
			cursorPos = 6
		}
	}

	prompt := promptui.Select{
		Label:     "Your framework",
		Items:     frameworks,
		Size:      7,
		CursorPos: cursorPos,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}

	return frameworkFromIndex(idx), nil
}

func printStartupInstructions(framework, serverURL string, port int, token string) {
	fmt.Println()
	fmt.Printf("Server running at http://localhost:%d\n", port)
	fmt.Printf("Dashboard: http://localhost:%d/dashboard?token=%s\n", port, token)
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()

	// Integration instructions
	fmt.Println("Add the script and test to your site:")
	fmt.Println()
	printFrameworkSnippet(framework, serverURL)
	fmt.Println()

	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list             List all tests")
	fmt.Println("  results <name>   View detailed test results")
	fmt.Println("  winner <name>    Declare a winning variant")
	fmt.Println("  create <name>    Create a test via CLI")
	fmt.Println("  export <name>    Export raw event data")
	fmt.Println("  token            Show dashboard URL")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
}

func frameworkFromIndex(idx int) string {
	switch idx {
	case 0:
		return "html"
	case 1:
		return "react"
	case 2:
		return "vue"
	case 3:
		return "svelte"
	case 4:
		return "laravel"
	case 5:
		return "django"
	default:
		return "other"
	}
}

func printFrameworkSnippet(framework, serverURL string) {
	switch framework {
	case "html":
		fmt.Println("   File: index.html (in <head>)")
		fmt.Println()
		fmt.Printf("   <script src=\"%s/hlg.js\" defer></script>\n", serverURL)
		fmt.Println()
		fmt.Println(`   <h1 data-hlg-name="hero" data-hlg-variants='["Ship Faster","Build Better"]'>`)
		fmt.Println("     Ship Faster")
		fmt.Println("   </h1>")
		fmt.Println(`   <button data-hlg-convert="hero">Sign Up</button>`)

	case "react":
		fmt.Println("   File: app/layout.tsx or pages/_app.tsx (use next/script)")
		fmt.Println()
		fmt.Printf("   <Script src=\"%s/hlg.js\" strategy=\"afterInteractive\" />\n", serverURL)
		fmt.Println()
		fmt.Println("   <h1")
		fmt.Println(`     data-hlg-name="hero"`)
		fmt.Println(`     data-hlg-variants='["Ship Faster","Build Better"]'`)
		fmt.Println("   >")
		fmt.Println("     Ship Faster")
		fmt.Println("   </h1>")
		fmt.Println(`   <button data-hlg-convert="hero">Sign Up</button>`)

	case "vue":
		fmt.Println("   File: index.html or nuxt.config.ts (head section)")
		fmt.Println()
		fmt.Printf("   <script src=\"%s/hlg.js\" defer></script>\n", serverURL)
		fmt.Println()
		fmt.Println("   <h1")
		fmt.Println(`     data-hlg-name="hero"`)
		fmt.Println(`     :data-hlg-variants="JSON.stringify(['Ship Faster','Build Better'])"`)
		fmt.Println("   >")
		fmt.Println("     Ship Faster")
		fmt.Println("   </h1>")
		fmt.Println(`   <button data-hlg-convert="hero">Sign Up</button>`)

	case "svelte":
		fmt.Println("   File: src/app.html or +layout.svelte")
		fmt.Println()
		fmt.Printf("   <script src=\"%s/hlg.js\" defer></script>\n", serverURL)
		fmt.Println()
		fmt.Println("   <h1")
		fmt.Println(`     data-hlg-name="hero"`)
		fmt.Println(`     data-hlg-variants={JSON.stringify(['Ship Faster','Build Better'])}`)
		fmt.Println("   >")
		fmt.Println("     Ship Faster")
		fmt.Println("   </h1>")
		fmt.Println(`   <button data-hlg-convert="hero">Sign Up</button>`)

	case "laravel":
		fmt.Println("   File: resources/views/layouts/app.blade.php (in <head>)")
		fmt.Println()
		fmt.Printf("   <script src=\"%s/hlg.js\" defer></script>\n", serverURL)
		fmt.Println()
		fmt.Println("   <h1")
		fmt.Println(`     data-hlg-name="hero"`)
		fmt.Println(`     data-hlg-variants='@json(["Ship Faster", "Build Better"])'`)
		fmt.Println("   >")
		fmt.Println("     Ship Faster")
		fmt.Println("   </h1>")
		fmt.Println(`   <button data-hlg-convert="hero">Sign Up</button>`)

	case "django":
		fmt.Println("   File: templates/base.html (in <head>)")
		fmt.Println()
		fmt.Printf("   <script src=\"%s/hlg.js\" defer></script>\n", serverURL)
		fmt.Println()
		fmt.Println("   <h1")
		fmt.Println(`     data-hlg-name="hero"`)
		fmt.Println(`     data-hlg-variants='["Ship Faster", "Build Better"]'`)
		fmt.Println("   >")
		fmt.Println("     Ship Faster")
		fmt.Println("   </h1>")
		fmt.Println(`   <button data-hlg-convert="hero">Sign Up</button>`)

	default: // other
		fmt.Println("   Add to your HTML <head>:")
		fmt.Println()
		fmt.Printf("   <script src=\"%s/hlg.js\" defer></script>\n", serverURL)
		fmt.Println()
		fmt.Println(`   <h1 data-hlg-name="hero" data-hlg-variants='["Ship Faster","Build Better"]'>`)
		fmt.Println("     Ship Faster")
		fmt.Println("   </h1>")
		fmt.Println(`   <button data-hlg-convert="hero">Sign Up</button>`)
	}
}
