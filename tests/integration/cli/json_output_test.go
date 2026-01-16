package cli_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gkobilansky/headline-goat/internal/cli"
	"github.com/gkobilansky/headline-goat/tests/testutil"
)

func TestResultsJSON_Structure(t *testing.T) {
	s := testutil.SetupTestStore(t)
	ctx := context.Background()

	// Create test
	test, _ := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")

	// Add events
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v1")
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v2")
	_ = s.RecordEvent(ctx, "hero", 1, "view", "v3")
	_ = s.RecordEvent(ctx, "hero", 0, "convert", "v1")

	// Get JSON output using the formatting function
	output, err := cli.FormatResultsJSON(ctx, s, test.Name)
	if err != nil {
		t.Fatalf("FormatResultsJSON failed: %v", err)
	}

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nOutput: %s", err, output)
	}

	// Check required fields
	if result["name"] != "hero" {
		t.Errorf("expected name=hero, got %v", result["name"])
	}

	if result["state"] != "running" {
		t.Errorf("expected state=running, got %v", result["state"])
	}

	if _, ok := result["variants"]; !ok {
		t.Error("expected variants field in JSON")
	}

	if _, ok := result["significance"]; !ok {
		t.Error("expected significance field in JSON")
	}

	if _, ok := result["status"]; !ok {
		t.Error("expected status field in JSON")
	}
}

func TestResultsJSON_VariantFields(t *testing.T) {
	s := testutil.SetupTestStore(t)
	ctx := context.Background()

	// Create test
	test, _ := s.CreateTest(ctx, "hero", []string{"Ship Faster", "Build Better"}, nil, "")

	// Add events
	_ = s.RecordEvent(ctx, "hero", 0, "view", "v1")
	_ = s.RecordEvent(ctx, "hero", 0, "convert", "v1")

	output, err := cli.FormatResultsJSON(ctx, s, test.Name)
	if err != nil {
		t.Fatalf("FormatResultsJSON failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(output), &result)

	variants := result["variants"].([]interface{})
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}

	v0 := variants[0].(map[string]interface{})
	if v0["name"] != "Ship Faster" {
		t.Errorf("expected first variant name 'Ship Faster', got %v", v0["name"])
	}

	// Check numeric fields exist
	if _, ok := v0["views"]; !ok {
		t.Error("expected views field in variant")
	}
	if _, ok := v0["conversions"]; !ok {
		t.Error("expected conversions field in variant")
	}
	if _, ok := v0["rate"]; !ok {
		t.Error("expected rate field in variant")
	}
}

func TestResultsJSON_StatusFields(t *testing.T) {
	s := testutil.SetupTestStore(t)
	ctx := context.Background()

	test, _ := s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")

	output, err := cli.FormatResultsJSON(ctx, s, test.Name)
	if err != nil {
		t.Fatalf("FormatResultsJSON failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(output), &result)

	status := result["status"].(map[string]interface{})

	// Check status fields
	if _, ok := status["ready"]; !ok {
		t.Error("expected ready field in status")
	}
	if _, ok := status["views_needed"]; !ok {
		t.Error("expected views_needed field in status")
	}
	if _, ok := status["message"]; !ok {
		t.Error("expected message field in status")
	}
}

func TestListJSON_Structure(t *testing.T) {
	s := testutil.SetupTestStore(t)
	ctx := context.Background()

	// Create tests
	s.CreateTest(ctx, "hero", []string{"A", "B"}, nil, "")
	s.CreateTest(ctx, "cta", []string{"X", "Y", "Z"}, nil, "")

	output, err := cli.FormatListJSON(ctx, s)
	if err != nil {
		t.Fatalf("FormatListJSON failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	tests, ok := result["tests"].([]interface{})
	if !ok {
		t.Fatal("expected tests array")
	}

	if len(tests) != 2 {
		t.Errorf("expected 2 tests, got %d", len(tests))
	}

	// Check first test has required fields
	test0 := tests[0].(map[string]interface{})
	requiredFields := []string{"name", "state", "variants", "total_views", "total_conversions", "created_at"}
	for _, field := range requiredFields {
		if _, ok := test0[field]; !ok {
			t.Errorf("expected %s field in test", field)
		}
	}
}

func TestCreateJSON_Structure(t *testing.T) {
	output := cli.FormatCreateJSON("hero", []string{"A", "B"})

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result["created"] != true {
		t.Error("expected created=true")
	}
	if result["name"] != "hero" {
		t.Errorf("expected name=hero, got %v", result["name"])
	}

	variants := result["variants"].([]interface{})
	if len(variants) != 2 {
		t.Errorf("expected 2 variants, got %d", len(variants))
	}
}
