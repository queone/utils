package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}

func TestValidDateAcceptsAndRejects(t *testing.T) {
	cases := []struct {
		in     string
		layout string
		want   bool
	}{
		{"2026-04-30", "2006-01-02", true},
		{"2026-Apr-30", "2006-01-02", true},
		{"2026-APR-30", "2006-01-02", true},
		{"2026-apr-30", "2006-01-02", true},
		{"not-a-date", "2006-01-02", false},
		{"2026-13-01", "2006-01-02", false},
	}
	for _, c := range cases {
		if got := validDate(c.in, c.layout); got != c.want {
			t.Errorf("validDate(%q, %q) = %v, want %v", c.in, c.layout, got, c.want)
		}
	}
}

func TestGetDaysSinceOrTo(t *testing.T) {
	now := time.Now().UTC()
	fiveDaysAgo := now.AddDate(0, 0, -5).Format("2006-01-02")
	fiveDaysFuture := now.AddDate(0, 0, 5).Format("2006-01-02")
	today := now.Format("2006-01-02")

	if got, err := getDaysSinceOrTo(fiveDaysAgo); err != nil || got != -5 {
		t.Errorf("getDaysSinceOrTo(%q) = (%d, %v), want (-5, nil)", fiveDaysAgo, got, err)
	}
	if got, err := getDaysSinceOrTo(fiveDaysFuture); err != nil || got != 5 {
		t.Errorf("getDaysSinceOrTo(%q) = (%d, %v), want (5, nil)", fiveDaysFuture, got, err)
	}
	if got, err := getDaysSinceOrTo(today); err != nil || got != 0 {
		t.Errorf("getDaysSinceOrTo(%q) = (%d, %v), want (0, nil)", today, got, err)
	}
}

func TestGetDateInDaysOffsets(t *testing.T) {
	now := time.Now()
	wantPlus := now.AddDate(0, 0, 5).Format("2006-01-02")
	wantMinus := now.AddDate(0, 0, -3).Format("2006-01-02")

	gotPlus, err := getDateInDays("+5")
	if err != nil {
		t.Fatalf("getDateInDays(\"+5\"): %v", err)
	}
	if formatted := gotPlus.Format("2006-01-02"); formatted != wantPlus {
		t.Errorf("getDateInDays(\"+5\") = %s, want %s", formatted, wantPlus)
	}

	gotMinus, err := getDateInDays("-3")
	if err != nil {
		t.Fatalf("getDateInDays(\"-3\"): %v", err)
	}
	if formatted := gotMinus.Format("2006-01-02"); formatted != wantMinus {
		t.Errorf("getDateInDays(\"-3\") = %s, want %s", formatted, wantMinus)
	}
}

func TestGetDaysBetweenLeapYear(t *testing.T) {
	if got, err := getDaysBetween("2024-01-01", "2024-12-31"); err != nil || got != 365 {
		t.Errorf("getDaysBetween(2024-01-01, 2024-12-31) = (%d, %v), want (365, nil)", got, err)
	}
	if got, err := getDaysBetween("2024-12-31", "2024-01-01"); err != nil || got != 365 {
		t.Errorf("getDaysBetween reversed = (%d, %v), want (365, nil)", got, err)
	}
}

func TestGetDateInDaysBadInput(t *testing.T) {
	if _, err := getDateInDays("not-a-number"); err == nil {
		t.Errorf("getDateInDays(\"not-a-number\") returned nil err")
	}
}

func TestGetDaysSinceOrToBadInput(t *testing.T) {
	if _, err := getDaysSinceOrTo("not-a-date"); err == nil {
		t.Errorf("getDaysSinceOrTo(\"not-a-date\") returned nil err")
	}
}

func TestGetDaysBetweenBadInput(t *testing.T) {
	if _, err := getDaysBetween("bad", "2026-01-01"); err == nil {
		t.Errorf("getDaysBetween(\"bad\", \"2026-01-01\") returned nil err")
	}
	if _, err := getDaysBetween("2026-01-01", "bad"); err == nil {
		t.Errorf("getDaysBetween(\"2026-01-01\", \"bad\") returned nil err")
	}
}

func TestNoPanicsInDates(t *testing.T) {
	body, err := os.ReadFile("dates.go")
	if err != nil {
		t.Fatalf("read dates.go: %v", err)
	}
	if strings.Contains(string(body), "panic(") {
		t.Errorf("dates.go still contains panic()")
	}
}

func TestPrintDaysFormatting(t *testing.T) {
	out := captureStdout(t, func() { printDays(400) })
	want := "400 (1 years + 34 days)\n"
	if out != want {
		t.Errorf("printDays(400) = %q, want %q", out, want)
	}

	out = captureStdout(t, func() { printDays(30) })
	if out != "30\n" {
		t.Errorf("printDays(30) = %q, want %q", out, "30\n")
	}
}

func TestPrintDaysNegative(t *testing.T) {
	out := captureStdout(t, func() { printDays(-400) })
	want := "-400 (1 years + 34 days)\n"
	if out != want {
		t.Errorf("printDays(-400) = %q, want %q", out, want)
	}
}

func TestNoQueoneUtlImport(t *testing.T) {
	// Production-code contract: main.go and dates.go must not import queone/utl.
	for _, f := range []string{"main.go", "dates.go"} {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		// Match only the actual import path (quoted), not stray string mentions.
		if strings.Contains(string(body), `"github.com/queone/utl"`) {
			t.Errorf("%s still imports github.com/queone/utl", f)
		}
	}
}
