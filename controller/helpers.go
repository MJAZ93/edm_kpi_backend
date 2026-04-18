package controller

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var monthlyPeriodRe = regexp.MustCompile(`^\d{4}-\d{2}$`)
var weeklyPeriodRe = regexp.MustCompile(`^(\d{4})-W(\d{2})$`)
var quarterlyPeriodRe = regexp.MustCompile(`^(\d{4})-Q([1-4])$`)
var biannualPeriodRe = regexp.MustCompile(`^(\d{4})-S([12])$`)
var annualPeriodRe = regexp.MustCompile(`^(\d{4})$`)

// isMonthlyPeriod reports whether s looks like a "YYYY-MM" period key.
func isMonthlyPeriod(s string) bool {
	return monthlyPeriodRe.MatchString(s)
}

// periodRefToMonth converts any period_reference value to a "YYYY-MM" string
// so progress events can be routed to the correct monthly target row.
//
// Conversion rules (in priority order):
//  - "YYYY-MM"  → used directly
//  - "YYYY-Www" → month containing that ISO week's Monday
//  - "YYYY-Qn"  → first month of the quarter (01, 04, 07, 10)
//  - "YYYY-Sn"  → first month of the semester (01, 07)
//  - "YYYY"     → January of that year
//  - anything else → fallback.UTC().Format("2006-01") (typically created_at)
func periodRefToMonth(s string, fallback time.Time) string {
	if isMonthlyPeriod(s) {
		return s
	}

	// Weekly: "2026-W07" → February 2026
	if m := weeklyPeriodRe.FindStringSubmatch(s); m != nil {
		year, _ := strconv.Atoi(m[1])
		week, _ := strconv.Atoi(m[2])
		// ISO week 1 always contains January 4.
		jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
		// Go's Weekday: Sun=0…Sat=6. Convert to ISO Mon=1…Sun=7.
		wd := int(jan4.Weekday())
		if wd == 0 {
			wd = 7
		}
		week1Mon := jan4.AddDate(0, 0, 1-wd)
		targetMon := week1Mon.AddDate(0, 0, (week-1)*7)
		return targetMon.Format("2006-01")
	}

	// Quarterly: "2026-Q2" → "2026-04"
	if m := quarterlyPeriodRe.FindStringSubmatch(s); m != nil {
		q, _ := strconv.Atoi(m[2])
		month := (q-1)*3 + 1
		return fmt.Sprintf("%s-%02d", m[1], month)
	}

	// Biannual: "2026-S1" → "2026-01", "2026-S2" → "2026-07"
	if m := biannualPeriodRe.FindStringSubmatch(s); m != nil {
		s2, _ := strconv.Atoi(m[2])
		month := (s2-1)*6 + 1
		return fmt.Sprintf("%s-%02d", m[1], month)
	}

	// Annual: "2026" → "2026-01"
	if m := annualPeriodRe.FindStringSubmatch(s); m != nil {
		return m[1] + "-01"
	}

	return fallback.UTC().Format("2006-01")
}

func parseDate(s string) (*time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// parseMonthlyPeriod parses a "YYYY-MM" string into a time.Time at the first
// day of that month (UTC). Used by the monthly targets controller to bridge
// with APIs that expect a full date.
func parseMonthlyPeriod(s string) (time.Time, error) {
	return time.Parse("2006-01", s)
}
