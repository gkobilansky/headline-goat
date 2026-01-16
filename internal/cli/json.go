package cli

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gkobilansky/headline-goat/internal/stats"
	"github.com/gkobilansky/headline-goat/internal/store"
)

// ResultsJSON represents the JSON output structure for results
type ResultsJSON struct {
	Name         string              `json:"name"`
	State        string              `json:"state"`
	CreatedAt    time.Time           `json:"created_at"`
	Variants     []VariantJSON       `json:"variants"`
	Significance SignificanceJSON    `json:"significance"`
	Status       StatusJSON          `json:"status"`
}

// VariantJSON represents a single variant in JSON output
type VariantJSON struct {
	Index       int     `json:"index"`
	Name        string  `json:"name"`
	Views       int     `json:"views"`
	Conversions int     `json:"conversions"`
	Rate        float64 `json:"rate"`
	CILower     float64 `json:"ci_lower"`
	CIUpper     float64 `json:"ci_upper"`
	Leading     bool    `json:"leading,omitempty"`
}

// SignificanceJSON represents statistical significance in JSON output
type SignificanceJSON struct {
	Confident       bool    `json:"confident"`
	ConfidenceLevel float64 `json:"confidence_level"`
	LeadingVariant  int     `json:"leading_variant"`
}

// StatusJSON represents the status estimate in JSON output
type StatusJSON struct {
	Ready               bool      `json:"ready"`
	ViewsNeeded         int       `json:"views_needed"`
	ViewsCurrent        int       `json:"views_current"`
	ProgressPercent     int       `json:"progress_percent"`
	TrafficRatePerHour  float64   `json:"traffic_rate_per_hour"`
	EstimatedHours      float64   `json:"estimated_hours"`
	CheckBackAt         time.Time `json:"check_back_at,omitempty"`
	Message             string    `json:"message"`
	RecommendedVariant  int       `json:"recommended_variant"`
}

// ListJSON represents the JSON output structure for list
type ListJSON struct {
	Tests []TestSummaryJSON `json:"tests"`
}

// TestSummaryJSON represents a test summary in the list
type TestSummaryJSON struct {
	Name             string    `json:"name"`
	State            string    `json:"state"`
	Variants         []string  `json:"variants"`
	TotalViews       int       `json:"total_views"`
	TotalConversions int       `json:"total_conversions"`
	CreatedAt        time.Time `json:"created_at"`
}

// CreateJSON represents the JSON output structure for create
type CreateJSON struct {
	Created  bool     `json:"created"`
	Name     string   `json:"name"`
	Variants []string `json:"variants"`
	Message  string   `json:"message"`
}

// FormatResultsJSON formats test results as JSON
func FormatResultsJSON(ctx context.Context, s *store.SQLiteStore, testName string) (string, error) {
	// Get test
	test, err := s.GetTest(ctx, testName)
	if err != nil {
		return "", err
	}

	// Get stats
	variantStats, err := s.GetVariantStats(ctx, testName)
	if err != nil {
		return "", err
	}

	// Analyze
	result := stats.Analyze(test, variantStats)

	// Get traffic rate
	viewCount, firstEvent, err := s.GetRecentViewCount(ctx, testName, 24)
	if err != nil {
		return "", err
	}
	trafficRate := stats.CalculateTrafficRate(viewCount, firstEvent, 24)

	// Calculate sample size needed
	baselineRate := 0.0
	if len(result.Variants) > 0 && result.Variants[0].Views > 0 {
		baselineRate = result.Variants[0].Rate
	}
	viewsNeeded := stats.RequiredSampleSize(baselineRate, 0.20)

	// Get status estimate
	status := stats.EstimateStatus(result, viewsNeeded, trafficRate)

	// Build JSON structure
	output := ResultsJSON{
		Name:      test.Name,
		State:     string(test.State),
		CreatedAt: test.CreatedAt,
		Variants:  make([]VariantJSON, len(result.Variants)),
		Significance: SignificanceJSON{
			Confident:       result.Confident,
			ConfidenceLevel: result.ConfidenceLevel,
			LeadingVariant:  result.LeadingVariant,
		},
	}

	// Calculate total views for progress
	totalViews := 0
	for i, v := range result.Variants {
		totalViews += v.Views
		output.Variants[i] = VariantJSON{
			Index:       v.Index,
			Name:        v.Name,
			Views:       v.Views,
			Conversions: v.Conversions,
			Rate:        v.Rate,
			CILower:     v.CILower,
			CIUpper:     v.CIUpper,
			Leading:     v.Index == result.LeadingVariant,
		}
	}

	totalNeeded := viewsNeeded * len(result.Variants)
	progressPct := 0
	if totalNeeded > 0 {
		progressPct = (totalViews * 100) / totalNeeded
	}

	output.Status = StatusJSON{
		Ready:              status.Ready,
		ViewsNeeded:        viewsNeeded,
		ViewsCurrent:       totalViews,
		ProgressPercent:    progressPct,
		TrafficRatePerHour: trafficRate,
		EstimatedHours:     status.EstimatedHours,
		CheckBackAt:        status.CheckBackTime,
		Message:            status.Message,
		RecommendedVariant: status.RecommendedVariant,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// FormatListJSON formats the test list as JSON
func FormatListJSON(ctx context.Context, s *store.SQLiteStore) (string, error) {
	tests, err := s.ListTests(ctx)
	if err != nil {
		return "", err
	}

	output := ListJSON{
		Tests: make([]TestSummaryJSON, len(tests)),
	}

	for i, test := range tests {
		// Get stats for this test
		variantStats, err := s.GetVariantStats(ctx, test.Name)
		if err != nil {
			return "", err
		}

		totalViews := 0
		totalConversions := 0
		for _, stat := range variantStats {
			totalViews += stat.Views
			totalConversions += stat.Conversions
		}

		output.Tests[i] = TestSummaryJSON{
			Name:             test.Name,
			State:            string(test.State),
			Variants:         test.Variants,
			TotalViews:       totalViews,
			TotalConversions: totalConversions,
			CreatedAt:        test.CreatedAt,
		}
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// FormatCreateJSON formats the create result as JSON
func FormatCreateJSON(name string, variants []string) string {
	output := CreateJSON{
		Created:  true,
		Name:     name,
		Variants: variants,
		Message:  "Test '" + name + "' created",
	}

	jsonBytes, _ := json.MarshalIndent(output, "", "  ")
	return string(jsonBytes)
}
