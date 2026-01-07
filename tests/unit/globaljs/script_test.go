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
