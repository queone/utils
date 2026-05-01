// dates.go — date-math helpers used by main.go. Pure stdlib.

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// stringToInt64 converts a numeric string to int64.
func stringToInt64(s string) (int64, error) {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return i, nil
}

// int64Abs returns the absolute value of an int64.
func int64Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// normalizeMonthAbbrev normalizes a 3-letter month string to "Jan"-style
// (uppercase first letter, lowercase next two). Returns input unchanged if too short.
func normalizeMonthAbbrev(m string) string {
	if len(m) < 3 {
		return m
	}
	return strings.ToUpper(m[:1]) + strings.ToLower(m[1:3])
}

// parseFlexibleDate tries parsing with both YYYY-MM-DD and YYYY-MMM-DD layouts.
// Accepts case-insensitive three-letter months (e.g., Jan, JAN, jan).
func parseFlexibleDate(dateStr string) (time.Time, error) {
	layouts := []string{"2006-01-02", "2006-Jan-02"}
	var lastErr error

	for _, layout := range layouts {
		try := dateStr
		if layout == "2006-Jan-02" && len(try) >= 8 {
			year := try[0:4]
			month := normalizeMonthAbbrev(try[5:8])
			rest := try[8:]
			try = year + "-" + month + rest
		}
		t, err := time.Parse(layout, try)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}

// validDateFlex reports whether dateString parses against any of the supplied layouts.
// Accepts case-insensitive three-letter months for "2006-Jan-02".
func validDateFlex(dateString string, layouts ...string) bool {
	for _, layout := range layouts {
		try := dateString
		if layout == "2006-Jan-02" && len(try) >= 8 {
			year := try[0:4]
			month := normalizeMonthAbbrev(try[5:8])
			rest := try[8:]
			try = year + "-" + month + rest
		}
		if _, err := time.Parse(layout, try); err == nil {
			return true
		}
	}
	return false
}

// validDate reports whether dateString parses as expectedFormat.
// When expectedFormat is "2006-01-02", also accepts "2006-Jan-02" in any casing.
func validDate(dateString, expectedFormat string) bool {
	if _, err := time.Parse(expectedFormat, dateString); err == nil {
		return true
	}
	if expectedFormat == "2006-01-02" {
		return validDateFlex(dateString, "2006-Jan-02")
	}
	return false
}

// epocInt64ToTime converts an epoch timestamp (seconds) to a time.Time.
func epocInt64ToTime(epocInt int64) time.Time {
	return time.Unix(epocInt, 0)
}

// dateStringToEpocInt64 converts dateString in dateFormat to Unix epoch seconds.
// When dateFormat is "2006-01-02", parsing is flexible (also accepts "2006-Jan-02" in any casing).
func dateStringToEpocInt64(dateString, dateFormat string) (int64, error) {
	var t time.Time
	var err error
	if dateFormat == "2006-01-02" {
		t, err = parseFlexibleDate(dateString)
	} else {
		t, err = time.Parse(dateFormat, dateString)
	}
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// getDateInDays returns the time.Time obtained by adding N days (parsed from "±N") to now.
func getDateInDays(days string) time.Time {
	now := time.Now().Unix()
	daysInt64, err := stringToInt64(days)
	if err != nil {
		panic(err.Error())
	}
	now += daysInt64 * 86400 // 86400 seconds in a day
	return epocInt64ToTime(now)
}

// isLeapYear reports whether year is a leap year by the Gregorian rule.
func isLeapYear(year int64) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// getDaysSinceOrTo returns the signed day offset from today (UTC) to date1.
// Negative if date1 is in the past, positive if in the future. Leap-year aware.
func getDaysSinceOrTo(date1 string) int64 {
	start, err := parseFlexibleDate(date1)
	if err != nil {
		panic(err.Error())
	}

	end := time.Now().UTC()

	var days int64 = 0
	var sign int64 = -1

	if start.After(end) {
		start, end = end, start
		sign = 1
	}

	for start.Year() < end.Year() || (start.Year() == end.Year() && start.YearDay() < end.YearDay()) {
		days++
		start = start.AddDate(0, 0, 1)

		// Adjust for leap years
		if start.Month() == time.February && start.Day() == 28 && isLeapYear(int64(start.Year())) {
			days++
			start = start.AddDate(0, 0, 1)
		}
	}

	return sign * days
}

// printDays prints the integer day count, also broken out as "years + days" when the magnitude
// fills at least one 365-or-366-day chunk. The year-counter starts at 0 and isLeapYear(0) is true,
// so the first chunk consumes 366 days — quirk preserved verbatim from the upstream utl.PrintDays.
func printDays(days int64) {
	daysAbs := int64Abs(days)
	var years int64 = 0

	for daysAbs >= 365 {
		leap := int64(0)
		if isLeapYear(years) {
			leap = 1
		}
		if daysAbs >= (365 + leap) {
			daysAbs -= (365 + leap)
			years++
		} else {
			break
		}
	}

	if years > 0 {
		fmt.Printf("%d (%d years + %d days)\n", days, years, daysAbs)
	} else {
		fmt.Println(days)
	}
}

// getDaysBetween returns the unsigned number of calendar days between two dates.
func getDaysBetween(date1, date2 string) int64 {
	epoc1, err := dateStringToEpocInt64(date1, "2006-01-02")
	if err != nil {
		panic(err.Error())
	}
	epoc2, err := dateStringToEpocInt64(date2, "2006-01-02")
	if err != nil {
		panic(err.Error())
	}

	return int64Abs(epoc1-epoc2) / 86400
}
