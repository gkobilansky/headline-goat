package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/headline-goat/headline-goat/internal/server"
	"github.com/headline-goat/headline-goat/internal/store"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var port int

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Start headline-goat server",
	Long: `Start the headline-goat server and show integration instructions.

The server provides:
  - Global script at /ht.js
  - Beacon endpoint for tracking events
  - Dashboard for viewing results

Tests auto-create when the first beacon arrives - no explicit setup needed.

Example:
  headline-goat init
  headline-goat init --port 8080`,
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
	// Prompt for framework to show appropriate instructions
	framework, err := promptFramework()
	if err != nil {
		return err
	}

	// Open database
	s, err := store.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer s.Close()

	// Token file path (alongside database)
	tokenFile := filepath.Join(filepath.Dir(dbPath), ".headline-goat-token")

	// Create server
	srv := server.New(s, port, tokenFile)

	// Print startup message with instructions
	printStartupInstructions(framework, port, srv.Token())

	// Start server quietly (we printed our own message)
	return srv.StartQuiet()
}

func promptFramework() (string, error) {
	frameworks := []string{
		"HTML (vanilla JavaScript)",
		"React / Next.js",
		"Vue",
		"Svelte",
		"Laravel / Django / Other",
	}

	prompt := promptui.Select{
		Label: "Your framework",
		Items: frameworks,
		Size:  5,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}

	switch idx {
	case 0:
		return "html", nil
	case 1:
		return "react", nil
	case 2:
		return "vue", nil
	case 3:
		return "svelte", nil
	default:
		return "other", nil
	}
}

func printStartupInstructions(framework string, port int, token string) {
	fmt.Println()
	fmt.Printf("Server running at http://localhost:%d\n", port)
	fmt.Printf("Dashboard: http://localhost:%d/dashboard?token=%s\n", port, token)
	fmt.Println()
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()

	// Step 1: Deploy
	fmt.Println("1. Deploy headline-goat to get a public URL")
	fmt.Println()
	fmt.Println("   Options: Fly.io, Cloudflare Tunnel, VPS with Caddy")
	fmt.Println("   Docs: https://github.com/headline-goat/headline-goat#deployment")
	fmt.Println()

	// Step 2: Add script
	fmt.Println("2. Add the script to your site")
	fmt.Println()
	fmt.Println("   <script src=\"https://YOUR-URL/ht.js\" defer></script>")
	fmt.Println()

	// Step 3: Create test
	fmt.Println("3. Add a test with data attributes")
	fmt.Println()
	printFrameworkExample(framework)
	fmt.Println()

	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  results <name>   Show test statistics")
	fmt.Println("  winner <name>    Declare a winner")
	fmt.Println("  list             List all tests")
	fmt.Println("  otp              Show dashboard URL")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
}

func printFrameworkExample(framework string) {
	switch framework {
	case "react":
		fmt.Println(`   <h1
     data-ht-name="hero"
     data-ht-variants='["Ship Faster","Build Better"]'
   >
     Ship Faster
   </h1>
   <button data-ht-convert="hero">Sign Up</button>`)
	case "vue":
		fmt.Println(`   <h1
     data-ht-name="hero"
     :data-ht-variants='JSON.stringify(["Ship Faster","Build Better"])'
   >
     Ship Faster
   </h1>
   <button data-ht-convert="hero">Sign Up</button>`)
	case "svelte":
		fmt.Println(`   <h1
     data-ht-name="hero"
     data-ht-variants={JSON.stringify(["Ship Faster","Build Better"])}
   >
     Ship Faster
   </h1>
   <button data-ht-convert="hero">Sign Up</button>`)
	default:
		fmt.Println(`   <h1 data-ht-name="hero" data-ht-variants='["A","B"]'>A</h1>
   <button data-ht-convert="hero">Sign Up</button>`)
	}
}
