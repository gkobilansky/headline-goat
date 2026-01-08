package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintFrameworkSnippet_HTML(t *testing.T) {
	output := captureOutput(func() {
		printFrameworkSnippet("html", "http://localhost:8080")
	})

	expectations := []string{
		"index.html",
		`<script src="http://localhost:8080/hlg.js" defer></script>`,
		`data-hlg-name="hero"`,
		`data-hlg-variants='["Ship Faster","Build Better"]'`,
	}

	for _, expected := range expectations {
		if !strings.Contains(output, expected) {
			t.Errorf("HTML output missing expected content: %s\n\nGot:\n%s", expected, output)
		}
	}
}

func TestPrintFrameworkSnippet_React(t *testing.T) {
	output := captureOutput(func() {
		printFrameworkSnippet("react", "http://localhost:8080")
	})

	expectations := []string{
		"app/layout.tsx",
		"next/script",
		`<Script src="http://localhost:8080/hlg.js"`,
		`strategy="afterInteractive"`,
		`data-hlg-name="hero"`,
	}

	for _, expected := range expectations {
		if !strings.Contains(output, expected) {
			t.Errorf("React output missing expected content: %s\n\nGot:\n%s", expected, output)
		}
	}
}

func TestPrintFrameworkSnippet_Vue(t *testing.T) {
	output := captureOutput(func() {
		printFrameworkSnippet("vue", "http://localhost:8080")
	})

	expectations := []string{
		"index.html",
		"nuxt.config.ts",
		`:data-hlg-variants="JSON.stringify`,
	}

	for _, expected := range expectations {
		if !strings.Contains(output, expected) {
			t.Errorf("Vue output missing expected content: %s\n\nGot:\n%s", expected, output)
		}
	}
}

func TestPrintFrameworkSnippet_Svelte(t *testing.T) {
	output := captureOutput(func() {
		printFrameworkSnippet("svelte", "http://localhost:8080")
	})

	expectations := []string{
		"src/app.html",
		"+layout.svelte",
		`data-hlg-variants={JSON.stringify`,
	}

	for _, expected := range expectations {
		if !strings.Contains(output, expected) {
			t.Errorf("Svelte output missing expected content: %s\n\nGot:\n%s", expected, output)
		}
	}
}

func TestPrintFrameworkSnippet_Laravel(t *testing.T) {
	output := captureOutput(func() {
		printFrameworkSnippet("laravel", "http://localhost:8080")
	})

	expectations := []string{
		"resources/views/layouts/app.blade.php",
		`@json(["Ship Faster", "Build Better"])`,
	}

	for _, expected := range expectations {
		if !strings.Contains(output, expected) {
			t.Errorf("Laravel output missing expected content: %s\n\nGot:\n%s", expected, output)
		}
	}
}

func TestPrintFrameworkSnippet_Django(t *testing.T) {
	output := captureOutput(func() {
		printFrameworkSnippet("django", "http://localhost:8080")
	})

	expectations := []string{
		"templates/base.html",
		`data-hlg-variants='["Ship Faster", "Build Better"]'`,
	}

	for _, expected := range expectations {
		if !strings.Contains(output, expected) {
			t.Errorf("Django output missing expected content: %s\n\nGot:\n%s", expected, output)
		}
	}
}

func TestPrintFrameworkSnippet_Other(t *testing.T) {
	output := captureOutput(func() {
		printFrameworkSnippet("other", "http://localhost:8080")
	})

	expectations := []string{
		"Add to your HTML <head>",
		`<script src="http://localhost:8080/hlg.js" defer></script>`,
	}

	for _, expected := range expectations {
		if !strings.Contains(output, expected) {
			t.Errorf("Other output missing expected content: %s\n\nGot:\n%s", expected, output)
		}
	}
}

func TestPrintFrameworkSnippet_UsesServerURL(t *testing.T) {
	customURL := "https://ab.example.com"
	output := captureOutput(func() {
		printFrameworkSnippet("html", customURL)
	})

	if !strings.Contains(output, customURL+"/hlg.js") {
		t.Errorf("Expected output to use custom server URL %s, got:\n%s", customURL, output)
	}
}

func TestFrameworkMapping(t *testing.T) {
	tests := []struct {
		index    int
		expected string
	}{
		{0, "html"},
		{1, "react"},
		{2, "vue"},
		{3, "svelte"},
		{4, "laravel"},
		{5, "django"},
		{6, "other"},
	}

	for _, tc := range tests {
		result := frameworkFromIndex(tc.index)
		if result != tc.expected {
			t.Errorf("frameworkFromIndex(%d) = %s, want %s", tc.index, result, tc.expected)
		}
	}
}
