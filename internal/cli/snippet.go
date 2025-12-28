package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/headline-goat/headline-goat/internal/snippets"
	"github.com/headline-goat/headline-goat/internal/store"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newSnippetCmd())
}

func newSnippetCmd() *cobra.Command {
	var framework string
	var animation string
	var serverURL string

	cmd := &cobra.Command{
		Use:   "snippet <name>",
		Short: "Generate integration code for a test",
		Long:  "Generate copy-paste-ready code snippets for integrating a headline test into your application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			testName := args[0]

			dbPath, _ := cmd.Flags().GetString("db")
			s, err := store.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer s.Close()

			ctx := context.Background()
			test, err := s.GetTest(ctx, testName)
			if err != nil {
				return fmt.Errorf("test not found: %s", testName)
			}

			// Determine framework
			fw := snippets.Framework(framework)
			if framework == "" {
				fw, err = promptFramework()
				if err != nil {
					return err
				}
			}

			// Determine animation (only for React/Vue/Svelte)
			anim := snippets.Animation(animation)
			if animation == "" && needsAnimation(fw) {
				anim, err = promptAnimation()
				if err != nil {
					return err
				}
			}

			// Determine server URL
			url := serverURL
			if url == "" {
				url, err = promptServerURL()
				if err != nil {
					return err
				}
			}

			// Build config
			config := snippets.Config{
				TestName:  test.Name,
				Variants:  test.Variants,
				ServerURL: url,
				Animation: anim,
			}

			// If test has a winner, set it
			if test.State == store.StateCompleted && test.WinnerVariant != nil {
				config.WinnerVariant = test.WinnerVariant
			}

			// Generate snippets
			files, err := snippets.Generate(fw, config)
			if err != nil {
				return fmt.Errorf("failed to generate snippet: %w", err)
			}

			// Print output
			printSnippets(files)

			return nil
		},
	}

	cmd.Flags().StringVarP(&framework, "framework", "f", "", "framework (html, nextjs, react, vue, svelte, laravel, django)")
	cmd.Flags().StringVarP(&animation, "animation", "a", "", "animation type (scramble, pixel, typewriter, none)")
	cmd.Flags().StringVarP(&serverURL, "server-url", "s", "", "server URL (e.g., https://ht.example.com)")

	return cmd
}

func needsAnimation(fw snippets.Framework) bool {
	return fw == snippets.FrameworkReact ||
		fw == snippets.FrameworkNextJS ||
		fw == snippets.FrameworkVue ||
		fw == snippets.FrameworkSvelte
}

func promptFramework() (snippets.Framework, error) {
	frameworks := []struct {
		Name      string
		Framework snippets.Framework
	}{
		{"HTML (vanilla JavaScript)", snippets.FrameworkHTML},
		{"Next.js", snippets.FrameworkNextJS},
		{"React", snippets.FrameworkReact},
		{"Vue", snippets.FrameworkVue},
		{"Svelte", snippets.FrameworkSvelte},
		{"Laravel", snippets.FrameworkLaravel},
		{"Django", snippets.FrameworkDjango},
	}

	items := make([]string, len(frameworks))
	for i, f := range frameworks {
		items[i] = f.Name
	}

	prompt := promptui.Select{
		Label: "Select framework",
		Items: items,
		Size:  7,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}

	return frameworks[idx].Framework, nil
}

func promptAnimation() (snippets.Animation, error) {
	animations := []struct {
		Name      string
		Animation snippets.Animation
	}{
		{"Scramble (letters randomize then resolve)", snippets.AnimationScramble},
		{"Pixel (characters fade in with pixelated effect)", snippets.AnimationPixel},
		{"Typewriter (characters appear one at a time)", snippets.AnimationTypewriter},
		{"None (instant display)", snippets.AnimationNone},
	}

	items := make([]string, len(animations))
	for i, a := range animations {
		items[i] = a.Name
	}

	prompt := promptui.Select{
		Label: "Select animation",
		Items: items,
		Size:  4,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}

	return animations[idx].Animation, nil
}

func promptServerURL() (string, error) {
	defaultURL := os.Getenv("HEADLINE_GOAT_URL")
	if defaultURL == "" {
		defaultURL = "http://localhost:8080"
	}

	prompt := promptui.Prompt{
		Label:   "Server URL",
		Default: defaultURL,
	}

	result, err := prompt.Run()
	if err != nil {
		if err == promptui.ErrInterrupt {
			os.Exit(0)
		}
		return "", err
	}

	return strings.TrimRight(result, "/"), nil
}

func printSnippets(files []snippets.SnippetFile) {
	for i, file := range files {
		if i > 0 {
			fmt.Println()
		}
		fmt.Println(strings.Repeat("=", 62))
		fmt.Printf(" %s\n", file.Filename)
		fmt.Println(strings.Repeat("=", 62))
		fmt.Println()
		fmt.Println(file.Content)
	}
}

// RunSnippetFlow runs the interactive snippet generation flow for a test
func RunSnippetFlow(test *store.Test) error {
	// Determine framework
	fw, err := promptFramework()
	if err != nil {
		return err
	}

	// Determine animation (only for React/Vue/Svelte)
	var anim snippets.Animation
	if needsAnimation(fw) {
		anim, err = promptAnimation()
		if err != nil {
			return err
		}
	}

	// Determine server URL
	url, err := promptServerURL()
	if err != nil {
		return err
	}

	// Build config
	config := snippets.Config{
		TestName:  test.Name,
		Variants:  test.Variants,
		ServerURL: url,
		Animation: anim,
	}

	// If test has a winner, set it
	if test.State == store.StateCompleted && test.WinnerVariant != nil {
		config.WinnerVariant = test.WinnerVariant
	}

	// Generate snippets
	files, err := snippets.Generate(fw, config)
	if err != nil {
		return fmt.Errorf("failed to generate snippet: %w", err)
	}

	// Print output
	fmt.Println()
	printSnippets(files)

	return nil
}
