package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gkobilansky/headline-goat/internal/dashboard"
	"github.com/gkobilansky/headline-goat/internal/stats"
)

// Dashboard template data structures
type layoutData struct {
	Title   string
	CSS     template.CSS
	Content template.HTML
}

type listData struct {
	Tests []testListItem
}

type testListItem struct {
	Name              string
	State             string
	VariantCount      int
	TotalViews        int
	AvgConversionRate string
	Goal              string
	CreatedAt         string
	HasSourceConflict bool
}

type detailData struct {
	Test               testDetailItem
	Result             *detailResult
	ConfidencePercent  float64
	LeadingVariantName string
}

type testDetailItem struct {
	Name              string
	State             string
	Goal              string
	CreatedAt         string
	Source            string
	HasSourceConflict bool
}

type detailResult struct {
	Variants       []detailVariant
	Confident      bool
	LeadingVariant int
}

type detailVariant struct {
	Index          int
	Name           string
	Views          int
	Conversions    int
	RatePercent    float64
	CILowerPercent float64
	CIUpperPercent float64
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Handle logout
	if r.URL.Query().Get("logout") == "1" {
		http.SetCookie(w, &http.Cookie{
			Name:   tokenCookieName,
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	ctx := context.Background()

	tests, err := s.store.ListTests(ctx)
	if err != nil {
		http.Error(w, "Failed to load tests", http.StatusInternalServerError)
		return
	}

	// Build list items
	items := make([]testListItem, len(tests))
	for i, t := range tests {
		variantStats, _ := s.store.GetVariantStats(ctx, t.Name)

		totalViews := 0
		totalConversions := 0
		for _, vs := range variantStats {
			totalViews += vs.Views
			totalConversions += vs.Conversions
		}

		avgRate := "0%"
		if totalViews > 0 {
			avgRate = formatPercentage(float64(totalConversions) / float64(totalViews) * 100)
		}

		items[i] = testListItem{
			Name:              t.Name,
			State:             string(t.State),
			VariantCount:      len(t.Variants),
			TotalViews:        totalViews,
			AvgConversionRate: avgRate,
			Goal:              t.ConversionGoal,
			CreatedAt:         t.CreatedAt.Format("Jan 2, 2006"),
			HasSourceConflict: t.HasSourceConflict,
		}
	}

	s.renderDashboard(w, "Dashboard", "list.html", listData{Tests: items})
}

func (s *Server) handleDashboardTest(w http.ResponseWriter, r *http.Request) {
	// Extract test name from path: /dashboard/test/<name>
	name := r.URL.Path[len("/dashboard/test/"):]
	if name == "" {
		http.NotFound(w, r)
		return
	}

	ctx := context.Background()

	test, err := s.store.GetTest(ctx, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	variantStats, err := s.store.GetVariantStats(ctx, name)
	if err != nil {
		http.Error(w, "Failed to load stats", http.StatusInternalServerError)
		return
	}

	result := stats.Analyze(test, variantStats)

	// Build detail variants
	variants := make([]detailVariant, len(result.Variants))
	for i, v := range result.Variants {
		variants[i] = detailVariant{
			Index:          v.Index,
			Name:           v.Name,
			Views:          v.Views,
			Conversions:    v.Conversions,
			RatePercent:    v.Rate * 100,
			CILowerPercent: v.CILower * 100,
			CIUpperPercent: v.CIUpper * 100,
		}
	}

	leadingName := ""
	if len(result.Variants) > 0 {
		leadingName = result.Variants[result.LeadingVariant].Name
	}

	data := detailData{
		Test: testDetailItem{
			Name:              test.Name,
			State:             string(test.State),
			Goal:              test.ConversionGoal,
			CreatedAt:         test.CreatedAt.Format("Jan 2, 2006"),
			Source:            test.Source,
			HasSourceConflict: test.HasSourceConflict,
		},
		Result: &detailResult{
			Variants:       variants,
			Confident:      result.Confident,
			LeadingVariant: result.LeadingVariant,
		},
		ConfidencePercent:  result.ConfidenceLevel * 100,
		LeadingVariantName: leadingName,
	}

	s.renderDashboard(w, test.Name, "detail.html", data)
}

func (s *Server) handleDashboardAPI(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	tests, err := s.store.ListTests(ctx)
	if err != nil {
		http.Error(w, "Failed to load tests", http.StatusInternalServerError)
		return
	}

	type apiVariantResult struct {
		Variant     int     `json:"variant"`
		VariantName string  `json:"variant_name"`
		Views       int     `json:"views"`
		Conversions int     `json:"conversions"`
		Rate        float64 `json:"rate"`
		CILower     float64 `json:"ci_lower"`
		CIUpper     float64 `json:"ci_upper"`
	}

	type apiSignificance struct {
		Confident          bool    `json:"confident"`
		ConfidenceLevel    float64 `json:"confidence_level"`
		LeadingVariant     int     `json:"leading_variant"`
		LeadingVariantName string  `json:"leading_variant_name"`
	}

	type apiTest struct {
		Name           string             `json:"name"`
		State          string             `json:"state"`
		Variants       []string           `json:"variants"`
		ConversionGoal string             `json:"conversion_goal,omitempty"`
		CreatedAt      string             `json:"created_at"`
		Results        []apiVariantResult `json:"results"`
		Significance   apiSignificance    `json:"significance"`
	}

	apiTests := make([]apiTest, len(tests))
	for i, t := range tests {
		variantStats, _ := s.store.GetVariantStats(ctx, t.Name)
		result := stats.Analyze(t, variantStats)

		results := make([]apiVariantResult, len(result.Variants))
		for j, v := range result.Variants {
			results[j] = apiVariantResult{
				Variant:     v.Index,
				VariantName: v.Name,
				Views:       v.Views,
				Conversions: v.Conversions,
				Rate:        v.Rate,
				CILower:     v.CILower,
				CIUpper:     v.CIUpper,
			}
		}

		leadingName := ""
		if len(result.Variants) > 0 {
			leadingName = result.Variants[result.LeadingVariant].Name
		}

		apiTests[i] = apiTest{
			Name:           t.Name,
			State:          string(t.State),
			Variants:       t.Variants,
			ConversionGoal: t.ConversionGoal,
			CreatedAt:      t.CreatedAt.Format("2006-01-02T15:04:05Z"),
			Results:        results,
			Significance: apiSignificance{
				Confident:          result.Confident,
				ConfidenceLevel:    result.ConfidenceLevel,
				LeadingVariant:     result.LeadingVariant,
				LeadingVariantName: leadingName,
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tests": apiTests,
	})
}

func (s *Server) renderDashboard(w http.ResponseWriter, title, contentTemplate string, data interface{}) {
	// Load CSS
	cssBytes, err := dashboard.Assets.ReadFile("assets/style.css")
	if err != nil {
		http.Error(w, "Failed to load styles", http.StatusInternalServerError)
		return
	}

	// Load and execute content template
	contentTmplBytes, err := dashboard.Templates.ReadFile("templates/" + contentTemplate)
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		return
	}

	contentTmpl, err := template.New("content").Parse(string(contentTmplBytes))
	if err != nil {
		http.Error(w, "Failed to parse template", http.StatusInternalServerError)
		return
	}

	var contentBuf bytes.Buffer
	if err := contentTmpl.Execute(&contentBuf, data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to render template: %v", err), http.StatusInternalServerError)
		return
	}

	// Load and execute layout template
	layoutTmplBytes, err := dashboard.Templates.ReadFile("templates/layout.html")
	if err != nil {
		http.Error(w, "Failed to load layout", http.StatusInternalServerError)
		return
	}

	layoutTmpl, err := template.New("layout").Parse(string(layoutTmplBytes))
	if err != nil {
		http.Error(w, "Failed to parse layout", http.StatusInternalServerError)
		return
	}

	layoutData := layoutData{
		Title:   title,
		CSS:     template.CSS(cssBytes),
		Content: template.HTML(contentBuf.String()),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := layoutTmpl.Execute(w, layoutData); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		return
	}
}

func formatPercentage(p float64) string {
	if p < 0.01 {
		return "0%"
	}
	return fmt.Sprintf("%.1f%%", p)
}
