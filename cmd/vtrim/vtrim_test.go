// vtrim_test.go

package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// AT1 — parseOffset syntax, duration-independent.
func TestParseOffset(t *testing.T) {
	ok := []struct {
		in    string
		secs  int
		comps int
	}{
		{"8:31", 511, 2},
		{"0", 0, 1},
		{"90", 90, 1},
		{"1:08:31", 4111, 3},
		{"0:00", 0, 2},
	}
	for _, c := range ok {
		secs, comps, err := parseOffset(c.in)
		if err != nil {
			t.Errorf("parseOffset(%q) unexpected error: %v", c.in, err)
			continue
		}
		if secs != c.secs || comps != c.comps {
			t.Errorf("parseOffset(%q) = (%d, %d), want (%d, %d)", c.in, secs, comps, c.secs, c.comps)
		}
	}

	bad := []string{"8:61", "1:2:3:4", "abc", "-5", "1:-1", "", "1:60:00"}
	for _, c := range bad {
		if _, _, err := parseOffset(c); err == nil {
			t.Errorf("parseOffset(%q) expected error, got nil", c)
		}
	}
}

// AT2 — validateTimestamp duration gate for 3-component forms.
func TestValidateTimestampDurationGate(t *testing.T) {
	if err := validateTimestamp(3, 3000); err == nil {
		t.Error("expected rejection of HH:MM:SS when duration <= 3600")
	}
	if err := validateTimestamp(3, 3600); err == nil {
		t.Error("expected rejection of HH:MM:SS when duration == 3600")
	}
	if err := validateTimestamp(3, 4000); err != nil {
		t.Errorf("expected HH:MM:SS accepted when duration > 3600, got %v", err)
	}
	if err := validateTimestamp(2, 100); err != nil {
		t.Errorf("MM:SS should never be duration-gated, got %v", err)
	}
}

// AT3 — per-mode bounds.
func TestValidateBounds(t *testing.T) {
	const d = 600.0
	cases := []struct {
		name       string
		mode       string
		start, end int
		wantErr    bool
	}{
		{"l ok", "l", 300, 0, false},
		{"l start==D", "l", 600, 0, true},
		{"l start>D", "l", 700, 0, true},
		{"r ok", "r", 0, 300, false},
		{"r end==0", "r", 0, 0, true},
		{"r end==D", "r", 0, 600, true},
		{"m ok", "m", 60, 300, false},
		{"m start>=end", "m", 300, 60, true},
		{"m end>D", "m", 60, 700, true},
		{"x ok", "x", 60, 300, false},
		{"x remove all", "x", 0, 600, true},
		{"x start>=end", "x", 300, 300, true},
	}
	for _, c := range cases {
		err := validateBounds(c.mode, c.start, c.end, d)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: validateBounds(%q,%d,%d) err=%v, wantErr=%v", c.name, c.mode, c.start, c.end, err, c.wantErr)
		}
	}
}

// AT4 — output name derivation.
func TestDeriveOutputName(t *testing.T) {
	cases := map[string]string{
		"april1.mp4":  "april_1.mp4",
		"april12.mp4": "april_12.mp4",
		"clip.mp4":    "clip_1.mp4",
		"a.b.mkv":     "a.b_1.mkv",
	}
	for in, want := range cases {
		if got := deriveOutputName(in); got != want {
			t.Errorf("deriveOutputName(%q) = %q, want %q", in, got, want)
		}
	}
}

// AT5 — byte and duration formatting.
func TestFormatBytesAndDuration(t *testing.T) {
	if got := formatBytes(1234567); got != "1,234,567" {
		t.Errorf("formatBytes(1234567) = %q, want %q", got, "1,234,567")
	}
	if got := formatBytes(999); got != "999" {
		t.Errorf("formatBytes(999) = %q, want %q", got, "999")
	}
	if got := formatDuration(4111); got != "01:08:31" {
		t.Errorf("formatDuration(4111) = %q, want %q", got, "01:08:31")
	}
	if got := formatDuration(0); got != "00:00:00" {
		t.Errorf("formatDuration(0) = %q, want %q", got, "00:00:00")
	}
}

// AT6 — buildPlan command shape per mode x copy/accurate.
func TestBuildPlan(t *testing.T) {
	const in, out, tmp = "in.mp4", "in_1.mp4", "/tmp/vt"

	t.Run("l copy", func(t *testing.T) {
		p := buildPlan("l", 511, 0, false, in, out, "")
		want := [][]string{{"-ss", "511", "-i", in, "-c", "copy", out}}
		assertCmds(t, p.Cmds, want)
	})
	t.Run("l accurate", func(t *testing.T) {
		p := buildPlan("l", 511, 0, true, in, out, "")
		want := [][]string{{"-i", in, "-ss", "511", "-c:v", "libx264", "-c:a", "aac", out}}
		assertCmds(t, p.Cmds, want)
	})
	t.Run("r copy", func(t *testing.T) {
		p := buildPlan("r", 0, 300, false, in, out, "")
		want := [][]string{{"-i", in, "-to", "300", "-c", "copy", out}}
		assertCmds(t, p.Cmds, want)
	})
	t.Run("m copy", func(t *testing.T) {
		p := buildPlan("m", 60, 511, false, in, out, "")
		want := [][]string{{"-ss", "60", "-i", in, "-t", "451", "-c", "copy", out}}
		assertCmds(t, p.Cmds, want)
	})
	t.Run("m accurate", func(t *testing.T) {
		p := buildPlan("m", 60, 511, true, in, out, "")
		want := [][]string{{"-i", in, "-ss", "60", "-t", "451", "-c:v", "libx264", "-c:a", "aac", out}}
		assertCmds(t, p.Cmds, want)
	})
	t.Run("x copy", func(t *testing.T) {
		p := buildPlan("x", 60, 300, false, in, out, tmp)
		seg1 := filepath.Join(tmp, "seg0.mp4")
		seg2 := filepath.Join(tmp, "seg1.mp4")
		list := filepath.Join(tmp, "concat.txt")
		want := [][]string{
			{"-i", in, "-to", "60", "-c", "copy", seg1},
			{"-ss", "300", "-i", in, "-c", "copy", seg2},
			{"-f", "concat", "-safe", "0", "-i", list, "-c", "copy", out},
		}
		assertCmds(t, p.Cmds, want)
		if p.ConcatList != list {
			t.Errorf("ConcatList = %q, want %q", p.ConcatList, list)
		}
		wantBody := "file '" + seg1 + "'\nfile '" + seg2 + "'\n"
		if p.ConcatBody != wantBody {
			t.Errorf("ConcatBody = %q, want %q", p.ConcatBody, wantBody)
		}
	})
	t.Run("x accurate", func(t *testing.T) {
		p := buildPlan("x", 60, 300, true, in, out, tmp)
		if len(p.Cmds) != 1 {
			t.Fatalf("x accurate: want single command, got %d", len(p.Cmds))
		}
		got := p.Cmds[0]
		if got[0] != "-i" || got[1] != in || got[2] != "-filter_complex" {
			t.Errorf("x accurate argv prefix = %v", got[:3])
		}
		if got[len(got)-1] != out {
			t.Errorf("x accurate output = %q, want %q", got[len(got)-1], out)
		}
		if p.ConcatList != "" {
			t.Errorf("x accurate should not use a concat list, got %q", p.ConcatList)
		}
	})
}

// AT7 — tools missing prints the brew hint and fails before any work.
func TestToolMissing(t *testing.T) {
	restore := lookPath
	defer func() { lookPath = restore }()
	lookPath = func(string) (string, error) { return "", errors.New("not found") }

	err := run("l", false, []string{"8:31", "whatever.mp4"})
	if err == nil {
		t.Fatal("expected error when ffmpeg is missing")
	}
	if want := "brew install ffmpeg"; !contains(err.Error(), want) {
		t.Errorf("error %q does not mention %q", err.Error(), want)
	}
}

// AT8 — overwrite protection errors without invoking ffmpeg.
func TestOverwriteProtection(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "clip0.mp4")
	writeFile(t, input, "data")
	writeFile(t, filepath.Join(dir, "clip_0.mp4"), "existing") // derived output already present

	var calls int
	withFakes(t, 600, func([]string) error { calls++; return nil })

	err := run("l", false, []string{"8:31", input})
	if err == nil || !contains(err.Error(), "already exists") {
		t.Fatalf("expected overwrite error, got %v", err)
	}
	if calls != 0 {
		t.Errorf("ffmpeg invoked %d times, want 0", calls)
	}
}

// AT9 — accurate vs copy routing reaches ffmpeg.
func TestAccurateVsCopyRouting(t *testing.T) {
	check := func(accurate bool, wantFlag, notFlag string) {
		dir := t.TempDir()
		input := filepath.Join(dir, "clip3.mp4")
		writeFile(t, input, "data")

		var got []string
		withFakes(t, 600, func(args []string) error {
			got = args
			writeFile(t, args[len(args)-1], "out") // create the output ffmpeg "produced"
			return nil
		})
		if err := run("l", accurate, []string{"8:31", input}); err != nil {
			t.Fatalf("run failed: %v", err)
		}
		if !sliceHas(got, wantFlag) {
			t.Errorf("accurate=%v argv %v missing %q", accurate, got, wantFlag)
		}
		if sliceHas(got, notFlag) {
			t.Errorf("accurate=%v argv %v should not contain %q", accurate, got, notFlag)
		}
	}
	check(false, "copy", "libx264")
	check(true, "libx264", "copy")
}

// AT10 — x temp dir cleaned up on success and on failure.
func TestCutTempCleanup(t *testing.T) {
	run10 := func(fail bool) string {
		dir := t.TempDir()
		input := filepath.Join(dir, "clip5.mp4")
		writeFile(t, input, "data")
		tmp := filepath.Join(dir, "scratch")

		restoreMk := makeTempDir
		makeTempDir = func() (string, error) { return tmp, os.Mkdir(tmp, 0o755) }
		defer func() { makeTempDir = restoreMk }()

		withFakes(t, 600, func(args []string) error {
			if fail {
				return errors.New("boom")
			}
			writeFile(t, args[len(args)-1], "out")
			return nil
		})

		err := run("x", false, []string{"1:00", "5:00", input})
		if fail && err == nil {
			t.Error("expected failure to propagate")
		}
		if !fail && err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		return tmp
	}

	for _, fail := range []bool{false, true} {
		tmp := run10(fail)
		if _, err := os.Stat(tmp); !os.IsNotExist(err) {
			t.Errorf("fail=%v: temp dir %q not cleaned up (stat err=%v)", fail, tmp, err)
		}
	}
}

// --- helpers ---

// withFakes installs lookPath, runFFprobe (fixed duration), and runFFmpeg fakes,
// restoring them when the test ends.
func withFakes(t *testing.T, durationSecs int, ffmpeg func([]string) error) {
	t.Helper()
	rl, rp, rf := lookPath, runFFprobe, runFFmpeg
	t.Cleanup(func() { lookPath, runFFprobe, runFFmpeg = rl, rp, rf })
	lookPath = func(string) (string, error) { return "/fake/bin", nil }
	runFFprobe = func([]string) ([]byte, error) {
		return []byte(strconv.Itoa(durationSecs) + ".000000\n"), nil
	}
	runFFmpeg = ffmpeg
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %q: %v", path, err)
	}
}

func assertCmds(t *testing.T, got, want [][]string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("argv mismatch:\n got  %v\n want %v", got, want)
	}
}

func sliceHas(s []string, v string) bool {
	return slices.Contains(s, v)
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
