package stats

import (
	"fmt"
	"time"
)

// StatusEstimate contains predictions for when a test will be ready.
type StatusEstimate struct {
	TotalViewsNeeded   int           // Per variant
	CurrentMinViews    int           // Minimum across variants
	ViewsRemaining     int           // To reach significance
	TrafficRatePerHour float64       // 0 if unknown
	EstimatedHours     float64       // 0 if can't estimate
	CheckBackTime      time.Time     // Zero if can't estimate
	Message            string        // Human-readable status
	Ready              bool          // True if significant
	RecommendedVariant int           // -1 if not ready
}

// EstimateStatus calculates time-to-significance estimates for a test.
//
// Parameters:
// - result: the current statistical analysis
// - viewsNeeded: required views per variant for significance
// - trafficRate: current traffic rate in views/hour (0 if unknown)
func EstimateStatus(result *Result, viewsNeeded int, trafficRate float64) StatusEstimate {
	status := StatusEstimate{
		TotalViewsNeeded:   viewsNeeded,
		RecommendedVariant: -1,
	}

	// Find minimum views across variants
	if len(result.Variants) > 0 {
		status.CurrentMinViews = result.Variants[0].Views
		for _, v := range result.Variants[1:] {
			if v.Views < status.CurrentMinViews {
				status.CurrentMinViews = v.Views
			}
		}
	}

	// Calculate views remaining
	if status.CurrentMinViews < viewsNeeded {
		status.ViewsRemaining = viewsNeeded - status.CurrentMinViews
	}

	// If already significant
	if result.Confident {
		status.Ready = true
		status.RecommendedVariant = result.LeadingVariant
		status.Message = fmt.Sprintf("Ready to declare winner. Run: hlg winner <test> --variant %d", result.LeadingVariant)
		return status
	}

	// Calculate time estimate if we have traffic
	status.TrafficRatePerHour = trafficRate
	if trafficRate > 0 && status.ViewsRemaining > 0 {
		status.EstimatedHours = float64(status.ViewsRemaining) / trafficRate
		status.CheckBackTime = time.Now().Add(time.Duration(status.EstimatedHours * float64(time.Hour)))
	}

	// Build message
	confPct := result.ConfidenceLevel * 100
	if confPct >= 90 {
		// Close to significance
		if status.EstimatedHours > 0 {
			hours := int(status.EstimatedHours + 0.5) // Round
			status.Message = fmt.Sprintf("%.0f%% confident. Need ~%d more views. Check back in ~%d hours.",
				confPct, status.ViewsRemaining, hours)
		} else {
			status.Message = fmt.Sprintf("%.0f%% confident. Need ~%d more views.", confPct, status.ViewsRemaining)
		}
	} else if status.CurrentMinViews < viewsNeeded/10 {
		// Very early - not enough data
		if status.EstimatedHours > 0 {
			hours := int(status.EstimatedHours + 0.5)
			status.Message = fmt.Sprintf("Not enough data. Need ~%d views per variant. Check back in ~%d hours.",
				viewsNeeded, hours)
		} else {
			status.Message = fmt.Sprintf("Not enough data. Need ~%d views per variant.", viewsNeeded)
		}
	} else if trafficRate == 0 {
		status.Message = "Collecting data. Traffic rate unknown, check back later."
	} else {
		// Normal progress
		hours := int(status.EstimatedHours + 0.5)
		progress := (float64(status.CurrentMinViews) / float64(viewsNeeded)) * 100
		status.Message = fmt.Sprintf("%.0f%% progress. Check back in ~%d hours.", progress, hours)
	}

	return status
}
