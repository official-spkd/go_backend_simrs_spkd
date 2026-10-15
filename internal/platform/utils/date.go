package utils

import "time"

// Today returns the current calendar date as a midnight timestamp, comparable with parsed
// YYYY-MM-DD values.
func Today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// AgeOn returns the age in whole years of someone born on birthDate, as of the given date.
func AgeOn(birthDate, on time.Time) int {
	age := on.Year() - birthDate.Year()
	if on.Month() < birthDate.Month() || (on.Month() == birthDate.Month() && on.Day() < birthDate.Day()) {
		age--
	}
	return age
}
