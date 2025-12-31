package store_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/headline-goat/headline-goat/internal/store"
)

func TestTestState_Constants(t *testing.T) {
	tests := []struct {
		state store.TestState
		want  string
	}{
		{store.StateRunning, "running"},
		{store.StatePaused, "paused"},
		{store.StateCompleted, "completed"},
	}

	for _, tt := range tests {
		if string(tt.state) != tt.want {
			t.Errorf("got %s, want %s", tt.state, tt.want)
		}
	}
}

func TestTest_Struct(t *testing.T) {
	now := time.Now()
	winner := 1
	test := store.Test{
		ID:             1,
		Name:           "hero",
		Variants:       []string{"A", "B", "C"},
		Weights:        []float64{0.5, 0.3, 0.2},
		ConversionGoal: "Signup button click",
		State:          store.StateRunning,
		WinnerVariant:  &winner,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if test.Name != "hero" {
		t.Errorf("got Name %s, want hero", test.Name)
	}
	if len(test.Variants) != 3 {
		t.Errorf("got %d variants, want 3", len(test.Variants))
	}
	if test.ConversionGoal != "Signup button click" {
		t.Errorf("got ConversionGoal %s, want 'Signup button click'", test.ConversionGoal)
	}
	if *test.WinnerVariant != 1 {
		t.Errorf("got WinnerVariant %d, want 1", *test.WinnerVariant)
	}
}

func TestEvent_Struct(t *testing.T) {
	now := time.Now()
	event := store.Event{
		ID:        1,
		TestName:  "hero",
		Variant:   0,
		EventType: "view",
		VisitorID: "abc123",
		CreatedAt: now,
	}

	if event.TestName != "hero" {
		t.Errorf("got TestName %s, want hero", event.TestName)
	}
	if event.EventType != "view" {
		t.Errorf("got EventType %s, want view", event.EventType)
	}
}

func TestVariantStats_Struct(t *testing.T) {
	stats := store.VariantStats{
		Variant:     0,
		Views:       100,
		Conversions: 10,
	}

	if stats.Views != 100 {
		t.Errorf("got Views %d, want 100", stats.Views)
	}
}

func TestVariants_JSONEncoding(t *testing.T) {
	variants := []string{"Ship Faster", "Build Better", "Scale Smart"}
	data, err := json.Marshal(variants)
	if err != nil {
		t.Fatalf("failed to marshal variants: %v", err)
	}

	expected := `["Ship Faster","Build Better","Scale Smart"]`
	if string(data) != expected {
		t.Errorf("got %s, want %s", string(data), expected)
	}

	var decoded []string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal variants: %v", err)
	}

	if len(decoded) != 3 {
		t.Errorf("got %d variants, want 3", len(decoded))
	}
}

func TestWeights_JSONEncoding(t *testing.T) {
	weights := []float64{0.5, 0.3, 0.2}
	data, err := json.Marshal(weights)
	if err != nil {
		t.Fatalf("failed to marshal weights: %v", err)
	}

	var decoded []float64
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal weights: %v", err)
	}

	if len(decoded) != 3 {
		t.Errorf("got %d weights, want 3", len(decoded))
	}

	sum := 0.0
	for _, w := range decoded {
		sum += w
	}
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("weights sum to %f, want 1.0", sum)
	}
}
