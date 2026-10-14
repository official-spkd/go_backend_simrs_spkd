package utils

import "time"

// Today returns the current calendar date as a midnight timestamp, comparable with parsed
// YYYY-MM-DD values.
func Today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
