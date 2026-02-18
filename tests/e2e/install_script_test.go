package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallScript_Exists(t *testing.T) {
	scriptPath := filepath.Join(getProjectRoot(t), "scripts", "install.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatal("install.sh script does not exist at scripts/install.sh")
	}
}

func TestInstallScript_ValidBashSyntax(t *testing.T) {
	scriptPath := filepath.Join(getProjectRoot(t), "scripts", "install.sh")

	// Check syntax with bash -n
	cmd := exec.Command("bash", "-n", scriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("install.sh has syntax errors: %v\n%s", err, output)
	}
}

func TestInstallScript_DetectsOSArch(t *testing.T) {
	scriptPath := filepath.Join(getProjectRoot(t), "scripts", "install.sh")

	// Source the script and test the detect functions
	cmd := exec.Command("bash", "-c", `
		source "`+scriptPath+`" --source-only
		os=$(detect_os)
		arch=$(detect_arch)
		echo "$os/$arch"
	`)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to detect OS/arch: %v\n%s", err, output)
	}

	result := strings.TrimSpace(string(output))
	parts := strings.Split(result, "/")
	if len(parts) != 2 {
		t.Fatalf("expected os/arch format, got: %s", result)
	}

	detectedOS := parts[0]
	detectedArch := parts[1]

	// Verify OS detection
	validOS := map[string]bool{"linux": true, "darwin": true}
	if !validOS[detectedOS] {
		t.Errorf("unexpected OS detected: %s", detectedOS)
	}

	// Verify arch detection
	validArch := map[string]bool{"amd64": true, "arm64": true}
	if !validArch[detectedArch] {
		t.Errorf("unexpected arch detected: %s", detectedArch)
	}

	// Cross-check with Go's runtime detection
	expectedOS := runtime.GOOS
	if detectedOS != expectedOS {
		t.Errorf("OS mismatch: script detected %s, Go reports %s", detectedOS, expectedOS)
	}

	expectedArch := runtime.GOARCH
	if detectedArch != expectedArch {
		t.Errorf("arch mismatch: script detected %s, Go reports %s", detectedArch, expectedArch)
	}
}

func TestInstallScript_BuildsCorrectURL(t *testing.T) {
	scriptPath := filepath.Join(getProjectRoot(t), "scripts", "install.sh")

	testCases := []struct {
		os       string
		arch     string
		expected string
	}{
		{"linux", "amd64", "hlg-linux-amd64"},
		{"linux", "arm64", "hlg-linux-arm64"},
		{"darwin", "amd64", "hlg-darwin-amd64"},
		{"darwin", "arm64", "hlg-darwin-arm64"},
	}

	for _, tc := range testCases {
		t.Run(tc.os+"-"+tc.arch, func(t *testing.T) {
			cmd := exec.Command("bash", "-c", `
				source "`+scriptPath+`" --source-only
				get_binary_name "`+tc.os+`" "`+tc.arch+`"
			`)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("failed to get binary name: %v\n%s", err, output)
			}

			result := strings.TrimSpace(string(output))
			if result != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestInstallScript_HasRequiredFunctions(t *testing.T) {
	scriptPath := filepath.Join(getProjectRoot(t), "scripts", "install.sh")

	requiredFunctions := []string{
		"detect_os",
		"detect_arch",
		"get_binary_name",
		"download_binary",
		"install_binary",
	}

	for _, fn := range requiredFunctions {
		t.Run(fn, func(t *testing.T) {
			cmd := exec.Command("bash", "-c", `
				source "`+scriptPath+`" --source-only
				type `+fn+` &>/dev/null && echo "exists" || echo "missing"
			`)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("failed to check function %s: %v\n%s", fn, err, output)
			}

			result := strings.TrimSpace(string(output))
			if result != "exists" {
				t.Errorf("required function %s is missing", fn)
			}
		})
	}
}

func getProjectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from test file to find project root (where go.mod is)
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}
