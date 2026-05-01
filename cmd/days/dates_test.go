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

	if got := getDaysSinceOrTo(fiveDaysAgo); got != -5 {
		t.Errorf("getDaysSinceOrTo(%q) = %d, want -5", fiveDaysAgo, got)
	}
	if got := getDaysSinceOrTo(fiveDaysFuture); got != 5 {
		t.Errorf("getDaysSinceOrTo(%q) = %d, want 5", fiveDaysFuture, got)
	}
	if got := getDaysSinceOrTo(today); got != 0 {
		t.Errorf("getDaysSinceOrTo(%q) = %d, want 0", today, got)
	}
}

func TestGetDateInDaysOffsets(t *testing.T) {
	now := time.Now()
	wantPlus := now.AddDate(0, 0, 5).Format("2006-01-02")
	wantMinus := now.AddDate(0, 0, -3).Format("2006-01-02")

	if got := getDateInDays("+5").Format("2006-01-02"); got != wantPlus {
		t.Errorf("getDateInDays(\"+5\") = %s, want %s", got, wantPlus)
	}
	if got := getDateInDays("-3").Format("2006-01-02"); got != wantMinus {
		t.Errorf("getDateInDays(\"-3\") = %s, want %s", got, wantMinus)
	}
}

func TestGetDaysBetweenLeapYear(t *testing.T) {
	if got := getDaysBetween("2024-01-01", "2024-12-31"); got != 365 {
		t.Errorf("getDaysBetween(2024-01-01, 2024-12-31) = %d, want 365", got)
	}
	if got := getDaysBetween("2024-12-31", "2024-01-01"); got != 365 {
		t.Errorf("getDaysBetween reversed = %d, want 365 (unsigned)", got)
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
