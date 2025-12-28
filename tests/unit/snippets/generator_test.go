package snippets_test

import (
	"strings"
	"testing"

	"github.com/headline-goat/headline-goat/internal/snippets"
)

func TestGenerate_HTML(t *testing.T) {
	config := snippets.Config{
		TestName:  "hero",
		Variants:  []string{"Ship Faster", "Build Better"},
		ServerURL: "http://localhost:8080",
	}

	files, err := snippets.Generate(snippets.FrameworkHTML, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := files[0].Content

	// Should contain script tag with server URL
	if !strings.Contains(content, "http://localhost:8080") {
		t.Error("expected content to contain server URL")
	}

	// Should contain test name
	if !strings.Contains(content, "hero") {
		t.Error("expected content to contain test name")
	}

	// Should contain data attributes
	if !strings.Contains(content, "data-ht-test") {
		t.Error("expected content to contain data-ht-test attribute")
	}
}

func TestGenerate_React(t *testing.T) {
	config := snippets.Config{
		TestName:  "hero",
		Variants:  []string{"Ship Faster", "Build Better"},
		ServerURL: "http://localhost:8080",
		Animation: snippets.AnimationScramble,
	}

	files, err := snippets.Generate(snippets.FrameworkReact, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// React should generate multiple files
	if len(files) < 4 {
		t.Fatalf("expected at least 4 files, got %d", len(files))
	}

	// Check for expected files
	fileNames := make(map[string]bool)
	for _, f := range files {
		fileNames[f.Filename] = true
	}

	expectedFiles := []string{"TrackImpression.tsx", "ConvertButton.tsx", "useConvert.ts", "useVisitorId.ts"}
	for _, name := range expectedFiles {
		if !fileNames[name] {
			t.Errorf("expected file %s to be generated", name)
		}
	}
}

func TestGenerate_NextJS(t *testing.T) {
	config := snippets.Config{
		TestName:  "hero",
		Variants:  []string{"A", "B", "C"},
		ServerURL: "https://ht.example.com",
		Animation: snippets.AnimationPixel,
	}

	files, err := snippets.Generate(snippets.FrameworkNextJS, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Next.js should include middleware
	if len(files) < 5 {
		t.Fatalf("expected at least 5 files, got %d", len(files))
	}

	fileNames := make(map[string]bool)
	for _, f := range files {
		fileNames[f.Filename] = true
	}

	if !fileNames["middleware.ts"] {
		t.Error("expected middleware.ts to be generated")
	}
}

func TestGenerate_Vue(t *testing.T) {
	config := snippets.Config{
		TestName:  "pricing",
		Variants:  []string{"Free Trial", "Get Started"},
		ServerURL: "http://localhost:8080",
		Animation: snippets.AnimationTypewriter,
	}

	files, err := snippets.Generate(snippets.FrameworkVue, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if files[0].Filename != "HeadlineTest.vue" {
		t.Errorf("expected HeadlineTest.vue, got %s", files[0].Filename)
	}

	// Should contain Vue template syntax
	if !strings.Contains(files[0].Content, "<template>") {
		t.Error("expected Vue template syntax")
	}
}

func TestGenerate_Svelte(t *testing.T) {
	config := snippets.Config{
		TestName:  "cta",
		Variants:  []string{"Buy Now", "Start Free"},
		ServerURL: "http://localhost:8080",
		Animation: snippets.AnimationNone,
	}

	files, err := snippets.Generate(snippets.FrameworkSvelte, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	if files[0].Filename != "HeadlineTest.svelte" {
		t.Errorf("expected HeadlineTest.svelte, got %s", files[0].Filename)
	}

	// Should contain Svelte script tag
	if !strings.Contains(files[0].Content, "<script") {
		t.Error("expected Svelte script syntax")
	}
}

func TestGenerate_Laravel(t *testing.T) {
	config := snippets.Config{
		TestName:  "hero",
		Variants:  []string{"A", "B"},
		ServerURL: "http://localhost:8080",
	}

	files, err := snippets.Generate(snippets.FrameworkLaravel, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Laravel should generate multiple files
	if len(files) < 2 {
		t.Fatalf("expected at least 2 files, got %d", len(files))
	}

	fileNames := make(map[string]bool)
	for _, f := range files {
		fileNames[f.Filename] = true
	}

	if !fileNames["HeadlineTestMiddleware.php"] {
		t.Error("expected HeadlineTestMiddleware.php to be generated")
	}
}

func TestGenerate_Django(t *testing.T) {
	config := snippets.Config{
		TestName:  "hero",
		Variants:  []string{"A", "B"},
		ServerURL: "http://localhost:8080",
	}

	files, err := snippets.Generate(snippets.FrameworkDjango, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Django should generate multiple files
	if len(files) >= 1 {
		// Should contain Python syntax
		hasPython := false
		for _, f := range files {
			if strings.Contains(f.Content, "def ") || strings.Contains(f.Content, "class ") {
				hasPython = true
				break
			}
		}
		if !hasPython {
			t.Error("expected Python syntax in Django files")
		}
	}
}

func TestGenerate_AllAnimations(t *testing.T) {
	animations := []snippets.Animation{
		snippets.AnimationScramble,
		snippets.AnimationPixel,
		snippets.AnimationTypewriter,
		snippets.AnimationNone,
	}

	for _, anim := range animations {
		t.Run(string(anim), func(t *testing.T) {
			config := snippets.Config{
				TestName:  "hero",
				Variants:  []string{"A", "B"},
				ServerURL: "http://localhost:8080",
				Animation: anim,
			}

			// Test with React (supports animations)
			files, err := snippets.Generate(snippets.FrameworkReact, config)
			if err != nil {
				t.Fatalf("unexpected error for animation %s: %v", anim, err)
			}

			if len(files) == 0 {
				t.Errorf("expected files for animation %s", anim)
			}
		})
	}
}

func TestGenerate_StaticWinner(t *testing.T) {
	config := snippets.Config{
		TestName:      "hero",
		Variants:      []string{"Ship Faster", "Build Better"},
		ServerURL:     "http://localhost:8080",
		WinnerVariant: intPtr(0),
	}

	files, err := snippets.Generate(snippets.FrameworkHTML, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	content := files[0].Content

	// Static winner should contain the winning variant directly
	if !strings.Contains(content, "Ship Faster") {
		t.Error("expected content to contain winning variant")
	}

	// Should NOT contain A/B testing logic
	if strings.Contains(content, "data-ht-test") {
		t.Error("static winner should not contain A/B testing attributes")
	}
}

func intPtr(i int) *int {
	return &i
}
