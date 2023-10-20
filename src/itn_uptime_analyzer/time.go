package itn_uptime_analyzer

import (
	"time"
	"strings"
)

// Returns current time in UTC format
func GetCurrentTime() time.Time {
	currentTime := time.Now().UTC()
	return currentTime
}

// If it's before noon, we check the interval between noon and midnight yesterday.
// Otherwise we check between midnight and noon today.
func GetExecutionInterval(currentTime time.Time) (time.Time, time.Time) {
	startDay := currentTime.Day()
	startHour := 0
	endHour := 12
	if currentTime.Hour() < 12 {
		startDay = currentTime.Day() - 1
		startHour = 12
		endHour = 0
	}

	periodStart := time.Date(
		currentTime.Year(),
		currentTime.Month(),
		startDay,
		startHour,
		0,
		0,
		0,
		currentTime.Location())

	periodEnd := time.Date(
		currentTime.Year(),
		currentTime.Month(),
		currentTime.Day(),
		endHour,
		0,
		0,
		0,
		currentTime.Location())

	return periodStart, periodEnd
}

// Decides if the application should check one or multiple buckets
func SubmissionsInMultipleBuckets(currentTime time.Time, executionInterval int) bool {
	if currentTime.Hour() < executionInterval {
		return true
	} else if currentTime.Hour() >= executionInterval {
		return false
	}

	return false
}

// Extract timestamp from S3 bucket key.
func GetSubmissionTime(key string) (time.Time, error) {
	filename := strings.Split(key, "/")[3]
	return time.Parse(time.RFC3339, filename[0:20])
}
