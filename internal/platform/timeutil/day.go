package timeutil

import (
	"strings"
	"time"
)

const DefaultTimezone = "Asia/Ho_Chi_Minh"

func ResolveLocation(name string) (*time.Location, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = DefaultTimezone
	}

	return time.LoadLocation(name)
}

func StartOfDay(day time.Time) time.Time {
	loc := day.Location()
	local := day.In(loc)

	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}

func CanonicalUTCDate(day time.Time) time.Time {
	local := day.In(day.Location())

	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC)
}

func DayBoundsUTC(day time.Time) (time.Time, time.Time) {
	start := StartOfDay(day)
	return start.UTC(), start.Add(24 * time.Hour).UTC()
}
