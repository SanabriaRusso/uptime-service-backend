package itn_uptime_analyzer

import (
	"time"
	"strings"

	logging "github.com/ipfs/go-log/v2"
)

type PeriodConfig struct {
	Start    time.Time     `json:"start"`
	End      time.Time     `json:"end"`
	Interval time.Duration `json:"interval"`
}

// Returns current time in UTC format
func GetCurrentTime() time.Time {
	currentTime := time.Now().UTC()
	return currentTime
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

// Default period start is 12 hours before period end.
func DeafultPeriodStart(periodEnd time.Time) time.Time {
	return periodEnd.Add(time.Hour * (-12))
}

// If it's afternoon, then set end time to the noon current day.
// Otherwise set it to midnight of the previous day.
func DefaultEndTime() time.Time {
	currentTime := time.Now().UTC()
	return currentTime.Truncate(time.Hour * 12)
}

// Set up the period configuration based on user-provided inputs.
// If at least 2 of the arguments are not-null, then the third one
// is computed based on the other two. If less than 2 arguments are
// provided, defaults kick in. When all 3 arguments are provided,
// they have to match or the program fails.
func GetPeriodConfig(periodStart *time.Time, periodEnd *time.Time,
                     executionInterval *time.Duration, log logging.EventLogger) PeriodConfig {
	var start time.Time
	var end time.Time
	var interval time.Duration

	switch {
	    case periodStart != nil && periodEnd != nil && executionInterval != nil:
             start = *periodStart
			 end = *periodEnd
             interval = *executionInterval * time.Minute
             if periodEnd.Sub(*periodStart) != interval {
               log.Fatal("Period start and period end do not match execution interval. Please check your configuration.")
             }

        case periodStart != nil && periodEnd != nil:
        	interval = periodEnd.Sub(*periodStart)
		    start = *periodStart
		    end = *periodEnd
        case periodStart != nil && executionInterval != nil:
            interval = *executionInterval
            end = periodStart.Add(interval)
		    start = *periodStart
        case periodEnd != nil && executionInterval != nil:
            interval = *executionInterval
            start = periodEnd.Add(-interval)
		    end = *periodEnd

        case periodStart != nil:
		    start = *periodStart
		    interval = time.Hour * 12
            end = periodStart.Add(interval)
        case periodEnd != nil:
			end = *periodEnd
		    interval = time.Hour * 12
            start = periodEnd.Add(-interval)
        case executionInterval != nil:
            interval = *executionInterval
            end = DefaultEndTime()
            start = end.Add(-interval)

        default:
            end = DefaultEndTime()
            interval = time.Hour * 12
            start = end.Add(-interval)
    }
    return PeriodConfig{
        Start:    start,
        End:      end,
        Interval: interval,
    }
}
