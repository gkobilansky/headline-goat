package cli_test

import (
	"context"
	"strings"
	"testing"

	"github.com/headline-goat/headline-goat/internal/snippets"
	"github.com/headline-goat/headline-goat/internal/store"
)

func TestSnippetGeneration_HTML(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Setup: create a test
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, err = s.CreateTest(context.Background(), "hero", []string{"Ship Faster", "Build Better"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	test, _ := s.GetTest(context.Background(), "hero")
	s.Close()

	// Generate snippet (simulating what CLI does)
	config := snippets.Config{
		TestName:  test.Name,
		Variants:  test.Variants,
		ServerURL: "http://localhost:8080",
	}

	files, err := snippets.Generate(snippets.FrameworkHTML, config)
	if err != nil {
		t.Fatalf("snippet generation failed: %v", err)
	}

	outputStr := files[0].Content

	// Verify output contains expected elements
	if !strings.Contains(outputStr, "hero") {
		t.Error("expected output to contain test name 'hero'")
	}
	if !strings.Contains(outputStr, "data-ht-test") {
		t.Error("expected output to contain 'data-ht-test' attribute")
	}
	if !strings.Contains(outputStr, "http://localhost:8080") {
		t.Error("expected output to contain server URL")
	}
}

func TestSnippetGeneration_React(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Setup: create a test
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, err = s.CreateTest(context.Background(), "pricing", []string{"A", "B", "C"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}

	test, _ := s.GetTest(context.Background(), "pricing")
	s.Close()

	// Generate snippet
	config := snippets.Config{
		TestName:  test.Name,
		Variants:  test.Variants,
		ServerURL: "https://ht.example.com",
		Animation: snippets.AnimationScramble,
	}

	files, err := snippets.Generate(snippets.FrameworkReact, config)
	if err != nil {
		t.Fatalf("snippet generation failed: %v", err)
	}

	// Verify React files are generated
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

	// Check server URL is in the output
	found := false
	for _, f := range files {
		if strings.Contains(f.Content, "https://ht.example.com") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected output to contain server URL")
	}
}

func TestSnippetGeneration_NextJS(t *testing.T) {
	config := snippets.Config{
		TestName:  "hero",
		Variants:  []string{"A", "B"},
		ServerURL: "http://localhost:8080",
		Animation: snippets.AnimationPixel,
	}

	files, err := snippets.Generate(snippets.FrameworkNextJS, config)
	if err != nil {
		t.Fatalf("snippet generation failed: %v", err)
	}

	// Next.js should include middleware
	fileNames := make(map[string]bool)
	for _, f := range files {
		fileNames[f.Filename] = true
	}

	if !fileNames["middleware.ts"] {
		t.Error("expected middleware.ts to be generated for Next.js")
	}
}

func TestSnippetGeneration_StaticWinner(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Setup: create a test and set winner
	s, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, err = s.CreateTest(context.Background(), "hero", []string{"Ship Faster", "Build Better"}, nil, "")
	if err != nil {
		t.Fatalf("failed to create test: %v", err)
	}
	err = s.SetWinner(context.Background(), "hero", 0)
	if err != nil {
		t.Fatalf("failed to set winner: %v", err)
	}

	test, _ := s.GetTest(context.Background(), "hero")
	s.Close()

	// Generate snippet with winner
	winnerIdx := 0
	config := snippets.Config{
		TestName:      test.Name,
		Variants:      test.Variants,
		ServerURL:     "http://localhost:8080",
		WinnerVariant: &winnerIdx,
	}

	files, err := snippets.Generate(snippets.FrameworkHTML, config)
	if err != nil {
		t.Fatalf("snippet generation failed: %v", err)
	}

	outputStr := files[0].Content

	// Static winner should contain the winning variant
	if !strings.Contains(outputStr, "Ship Faster") {
		t.Error("expected output to contain winning variant 'Ship Faster'")
	}

	// Static winner should NOT contain A/B testing attributes
	if strings.Contains(outputStr, "data-ht-test") {
		t.Error("static winner should not contain A/B testing attributes")
	}
}

func TestSnippetGeneration_AllFrameworks(t *testing.T) {
	config := snippets.Config{
		TestName:  "test",
		Variants:  []string{"A", "B"},
		ServerURL: "http://localhost:8080",
		Animation: snippets.AnimationScramble,
	}

	frameworks := snippets.AllFrameworks()
	for _, fw := range frameworks {
		t.Run(string(fw), func(t *testing.T) {
			files, err := snippets.Generate(fw, config)
			if err != nil {
				t.Fatalf("failed to generate %s snippet: %v", fw, err)
			}
			if len(files) == 0 {
				t.Errorf("expected at least 1 file for %s", fw)
			}
		})
	}
}
