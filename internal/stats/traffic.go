package stats

import "time"

// CalculateTrafficRate computes views per hour based on event data.
//
// Parameters:
// - views: number of views in the time window
// - firstEvent: timestamp of the earliest event in the window
// - windowHours: the query window size in hours
//
// Returns views per hour. Uses minimum 1 hour to avoid division by tiny numbers.
// Caps elapsed time at windowHours if data is older than the window.
func CalculateTrafficRate(views int, firstEvent time.Time, windowHours int) float64 {
	if views == 0 || firstEvent.IsZero() {
		return 0
	}

	elapsed := time.Since(firstEvent).Hours()

	// Minimum 1 hour to avoid division by tiny numbers
	if elapsed < 1 {
		elapsed = 1
	}

	// Cap at window size
	if elapsed > float64(windowHours) {
		elapsed = float64(windowHours)
	}

	return float64(views) / elapsed
}
