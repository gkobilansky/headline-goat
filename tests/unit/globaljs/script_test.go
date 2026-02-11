package globaljs_test

import (
	"strings"
	"testing"

	"github.com/gkobilansky/headline-goat/internal/server"
)

func TestGenerateGlobalScript_ReturnsValidJS(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should be non-empty
	if len(script) == 0 {
		t.Error("expected non-empty script")
	}

	// Should be a self-executing function (IIFE)
	if !strings.Contains(script, "(function()") || !strings.Contains(script, "})();") {
		t.Error("expected script to be an IIFE")
	}
}

func TestGenerateGlobalScript_ContainsBeaconEndpoint(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should contain beacon sending logic
	if !strings.Contains(script, "sendBeacon") {
		t.Error("expected script to use sendBeacon")
	}

	// Should contain the /b endpoint
	if !strings.Contains(script, "/b") {
		t.Error("expected script to contain beacon endpoint '/b'")
	}
}

func TestGenerateGlobalScript_ContainsLocalStorageLogic(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should use localStorage for visitor ID
	if !strings.Contains(script, "localStorage") {
		t.Error("expected script to use localStorage")
	}

	// Should have visitor ID key
	if !strings.Contains(script, "hlg_vid") {
		t.Error("expected script to contain visitor ID key 'hlg_vid'")
	}
}

func TestGenerateGlobalScript_ContainsDataAttributeSelectors(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should select data-hlg-name elements
	if !strings.Contains(script, "data-hlg-name") {
		t.Error("expected script to select data-hlg-name elements")
	}

	// Should select data-hlg-convert elements
	if !strings.Contains(script, "data-hlg-convert") {
		t.Error("expected script to select data-hlg-convert elements")
	}

	// Should handle variants via dataset.hlgVariants (JavaScript camelCase API)
	if !strings.Contains(script, "hlgVariants") {
		t.Error("expected script to handle variants via hlgVariants")
	}
}

func TestGenerateGlobalScript_ContainsVariantAssignment(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should contain random variant assignment logic
	if !strings.Contains(script, "Math.random") || !strings.Contains(script, "Math.floor") {
		t.Error("expected script to contain random variant assignment")
	}

	// Should store variant in localStorage
	if !strings.Contains(script, "hlg_") {
		t.Error("expected script to store variant with 'hlg_' prefix")
	}
}

func TestGenerateGlobalScript_ContainsServerURL(t *testing.T) {
	testURL := "https://ht.example.com"
	script := server.GenerateGlobalScript(testURL)

	if !strings.Contains(script, testURL) {
		t.Errorf("expected script to contain server URL %s", testURL)
	}
}

func TestGenerateGlobalScript_HandlesViewEvents(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should send view event
	if !strings.Contains(script, "'view'") && !strings.Contains(script, "\"view\"") {
		t.Error("expected script to send 'view' events")
	}
}

func TestGenerateGlobalScript_HandlesConvertEvents(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should send convert event
	if !strings.Contains(script, "'convert'") && !strings.Contains(script, "\"convert\"") {
		t.Error("expected script to send 'convert' events")
	}

	// Should add click handler
	if !strings.Contains(script, "click") {
		t.Error("expected script to add click handlers for conversions")
	}
}

// --- GET query param beacon tests ---

func TestGenerateGlobalScript_UsesQueryParamBeacon(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should build URL with query params instead of JSON body
	if !strings.Contains(script, "?t=") || !strings.Contains(script, "&v=") {
		t.Error("expected beacon to use query parameters (e.g., ?t=...&v=...)")
	}
}

func TestGenerateGlobalScript_BeaconFallsBackToImagePixel(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should use Image() as a fallback for environments without sendBeacon
	if !strings.Contains(script, "new Image") {
		t.Error("expected beacon to fall back to image pixel (new Image)")
	}
}

// --- Bot detection tests (client-side) ---

func TestGenerateGlobalScript_ContainsBotDetection(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should check navigator.webdriver (automation detection)
	if !strings.Contains(script, "navigator.webdriver") {
		t.Error("expected script to check navigator.webdriver for bot detection")
	}
}

func TestGenerateGlobalScript_SkipsBeaconForBots(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should check for headless/automation indicators
	if !strings.Contains(script, "webdriver") {
		t.Error("expected script to detect webdriver-based bots")
	}

	// Should have a guard that prevents beacon sending for detected bots
	if !strings.Contains(script, "isBot") && !strings.Contains(script, "bot") {
		t.Error("expected script to have bot detection variable or check")
	}
}

// --- SPA support tests ---

func TestGenerateGlobalScript_HandlesSPANavigation(t *testing.T) {
	script := server.GenerateGlobalScript("http://localhost:8080")

	// Should intercept pushState for SPA route changes
	if !strings.Contains(script, "pushState") {
		t.Error("expected script to intercept history.pushState for SPA support")
	}

	// Should listen for popstate (back/forward navigation)
	if !strings.Contains(script, "popstate") {
		t.Error("expected script to listen for popstate events")
	}
}
