package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPat.ReplaceAllString(s, "")
}

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

// synthesizeDraws builds n distinct Cash5-shaped draws, one per day going
// backward from a fixed date, with deterministic non-colliding primary
// numbers per draw.
func synthesizeDraws(n int) []Draw {
	base := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	draws := make([]Draw, n)
	for i := range n {
		nums := [5]int{
			1 + (i*1)%45,
			1 + (i*2+7)%45,
			1 + (i*3+13)%45,
			1 + (i*5+19)%45,
			1 + (i*7+29)%45,
		}
		seen := map[int]bool{}
		for j := range 5 {
			for seen[nums[j]] {
				nums[j] = nums[j]%45 + 1
			}
			seen[nums[j]] = true
		}
		primary := make([]string, 5)
		for j := range 5 {
			primary[j] = fmt.Sprintf("%d", nums[j])
		}
		draws[i] = Draw{
			ID:       fmt.Sprintf("syn-%04d", i),
			GameName: "Cash 5",
			DrawTime: base.AddDate(0, 0, -i).UnixMilli(),
			Results:  []Result{{Primary: primary}},
		}
	}
	return draws
}

// Per-draw header lines have the shape:
//
//	<date>  <numStr>  5/5 <payout>
//
// where <date> is narrativeDate (YYYY-mmm-DD) and <numStr> is
// DD-DD-DD-DD-DD. Inner match lines have leading whitespace.
var perDrawHeaderPat = regexp.MustCompile(`(?m)^\d{4}-[a-z]{3}-\d{2}\s+\d{2}-\d{2}-\d{2}-\d{2}-\d{2}\s+5/5`)

func countPerDrawBlocks(out string) int {
	return len(perDrawHeaderPat.FindAllString(stripANSI(out), -1))
}

func firstLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// neutralizeTerm makes isITerm2() deterministic so the image-escape branch
// never fires during tests, regardless of the developer's terminal.
func neutralizeTerm(t *testing.T) {
	t.Helper()
	t.Setenv("TERM_PROGRAM", "")
}

func TestMatchAnalysisWindowsDisplayLoop(t *testing.T) {
	neutralizeTerm(t)
	draws := synthesizeDraws(40)

	cases := []struct {
		name string
		n    int
		want int // expected per-draw block count
	}{
		{"n=5", 5, 5},
		{"n=30", 30, 30},
		{"n=100 caps at len-1", 100, 39}, // entry 0 always skipped → 39 of 40
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				if err := displayMatchAnalysis(draws, tc.n); err != nil {
					t.Fatalf("displayMatchAnalysis: %v", err)
				}
			})
			if got := countPerDrawBlocks(out); got != tc.want {
				t.Errorf("blocks = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestMatchAnalysisPatternSectionsAlwaysPresent(t *testing.T) {
	neutralizeTerm(t)
	draws := synthesizeDraws(40)

	out := captureStdout(t, func() {
		if err := displayMatchAnalysis(draws, 5); err != nil {
			t.Fatalf("displayMatchAnalysis: %v", err)
		}
	})
	stripped := stripANSI(out)
	for _, s := range []string{
		"PATTERN ANALYSIS",
		"Number Frequency in Top Matches",
		"Recency Weighting",
		"Match Distribution Shift Over Time",
		"Top Pairs in Closest Matches",
	} {
		if !strings.Contains(stripped, s) {
			t.Errorf("output missing section %q", s)
		}
	}
}

func TestMatchAnalysisHeaderPhrasing(t *testing.T) {
	neutralizeTerm(t)
	draws := synthesizeDraws(40)

	// Windowed: n < len(parsed) → header includes "(of 40 total)".
	out := captureStdout(t, func() {
		if err := displayMatchAnalysis(draws, 5); err != nil {
			t.Fatalf("displayMatchAnalysis: %v", err)
		}
	})
	stripped := stripANSI(out)
	if !strings.Contains(stripped, "Analyzing 5 drawings (of 40 total) for closest") {
		t.Errorf("windowed header missing or malformed; first lines:\n%s", firstLines(stripped, 6))
	}

	// Unwindowed: n >= len(parsed) → plain form, no "(of".
	out = captureStdout(t, func() {
		if err := displayMatchAnalysis(draws, 40); err != nil {
			t.Fatalf("displayMatchAnalysis: %v", err)
		}
	})
	stripped = stripANSI(out)
	if !strings.Contains(stripped, "Analyzing 40 drawings for closest") {
		t.Errorf("unwindowed header missing or malformed; first lines:\n%s", firstLines(stripped, 6))
	}
	if strings.Contains(stripped, "(of") {
		t.Errorf("unwindowed header should not contain '(of'; first lines:\n%s", firstLines(stripped, 6))
	}
}
