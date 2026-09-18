package ingest

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// dailySpecRE matches the supported cron format: "0 H * * *" where H is 0-23.
var dailySpecRE = regexp.MustCompile(`^0 (\d{1,2}) \* \* \*$`)

// parseHour extracts the hour from a "0 H * * *" spec string.
func parseHour(spec string) (int, error) {
	m := dailySpecRE.FindStringSubmatch(spec)
	if m == nil {
		return 0, fmt.Errorf("unsupported schedule spec %q: only daily format \"0 H * * *\" (H=0-23) is supported", spec)
	}
	h, err := strconv.Atoi(m[1])
	if err != nil || h < 0 || h > 23 {
		return 0, fmt.Errorf("invalid hour in spec %q: %s (expected 0-23)", spec, m[1])
	}
	return h, nil
}

// NextRunTime returns the next time at H:00:00 after `now`, computed from `spec`
// which must be in the form "0 H * * *" (H=0-23).
func NextRunTime(spec string, now time.Time) (time.Time, error) {
	hour, err := parseHour(spec)
	if err != nil {
		return time.Time{}, err
	}

	// Start of today at the specified hour.
	nowLocal := now.In(time.Local)
	midnight := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, time.Local)
	runToday := midnight.Add(time.Duration(hour) * time.Hour)

	if nowLocal.Before(runToday) {
		return runToday, nil
	}
	// Already past today's  H:00 → tomorrow.
	return runToday.Add(24 * time.Hour), nil
}
