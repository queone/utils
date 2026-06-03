// vtrim.go

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	color "github.com/queone/governa-color"
)

// Injectable seams, overridden in tests to avoid invoking real binaries.
var (
	lookPath = exec.LookPath

	runFFmpeg = func(args []string) error {
		cmd := exec.Command("ffmpeg", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	runFFprobe = func(args []string) ([]byte, error) {
		return exec.Command("ffprobe", args...).Output()
	}

	makeTempDir = func() (string, error) { return os.MkdirTemp("", "vtrim-") }
)

// parseOffset parses a timestamp's syntax into whole seconds, independent of any
// source duration. It returns the seconds, the colon-separated component count
// (1, 2, or 3), and an error for malformed input.
func parseOffset(s string) (int, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) < 1 || len(parts) > 3 {
		return 0, 0, fmt.Errorf("invalid timestamp %q: use SS, MM:SS, or HH:MM:SS", s)
	}
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0, 0, fmt.Errorf("invalid timestamp %q: fields must be non-negative integers", s)
		}
		nums[i] = n
	}
	switch len(nums) {
	case 1:
		return nums[0], 1, nil
	case 2:
		if nums[1] > 59 {
			return 0, 0, fmt.Errorf("invalid timestamp %q: seconds must be 00-59", s)
		}
		return nums[0]*60 + nums[1], 2, nil
	default: // 3
		if nums[1] > 59 || nums[2] > 59 {
			return 0, 0, fmt.Errorf("invalid timestamp %q: minutes and seconds must be 00-59", s)
		}
		return nums[0]*3600 + nums[1]*60 + nums[2], 3, nil
	}
}

// validateTimestamp applies the duration-gated rule: a 3-component HH:MM:SS
// timestamp is permitted only when the source is longer than one hour.
func validateTimestamp(components int, durationSeconds float64) error {
	if components == 3 && durationSeconds <= 3600 {
		return errors.New("HH:MM:SS form is only allowed when the source is longer than one hour")
	}
	return nil
}

// validateBounds checks per-mode timestamp bounds against the source duration d
// (seconds). For single-timestamp modes (l, r) the unused timestamp is ignored.
func validateBounds(mode string, start, end int, d float64) error {
	dd := formatDuration(int(d))
	switch mode {
	case "l":
		if start < 0 || float64(start) >= d {
			return fmt.Errorf("start cannot reach or exceed the source duration %s", dd)
		}
	case "r":
		if end <= 0 || float64(end) >= d {
			return fmt.Errorf("end must be greater than 0 and less than the source duration %s", dd)
		}
	case "m", "x":
		if start < 0 || start >= end || float64(end) > d {
			return fmt.Errorf("require 0 <= start < end <= source duration %s", dd)
		}
		if mode == "x" && start == 0 && float64(end) >= d {
			return errors.New("cut would remove the entire video")
		}
	}
	return nil
}

// deriveOutputName builds the output filename from an input basename: it inserts
// '_' before the trailing run of digits in the stem (april1.mp4 -> april_1.mp4),
// or appends '_1' when the stem has no trailing digit (clip.mp4 -> clip_1.mp4).
func deriveOutputName(name string) string {
	ext := filepath.Ext(name)
	stem := name[:len(name)-len(ext)]
	i := len(stem)
	for i > 0 && stem[i-1] >= '0' && stem[i-1] <= '9' {
		i--
	}
	if i == len(stem) {
		return stem + "_1" + ext
	}
	return stem[:i] + "_" + stem[i:] + ext
}

// formatBytes renders a byte count with thousands separators (1234567 -> "1,234,567").
func formatBytes(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	for i, ch := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}
	out := b.String()
	if neg {
		out = "-" + out
	}
	return out
}

// formatDuration renders whole seconds as zero-padded HH:MM:SS (4111 -> "01:08:31").
func formatDuration(totalSeconds int) string {
	if totalSeconds < 0 {
		totalSeconds = 0
	}
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// probeDurationSeconds returns the total stream duration of path via ffprobe.
func probeDurationSeconds(path string) (float64, error) {
	out, err := runFFprobe([]string{
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	})
	if err != nil {
		return 0, fmt.Errorf("probing %q: %w (is it a valid video file?)", path, err)
	}
	s := strings.TrimSpace(string(out))
	d, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("probing %q: could not parse duration %q", path, s)
	}
	return d, nil
}

// toolsAvailable verifies ffmpeg and ffprobe are on PATH, with an install hint.
func toolsAvailable() error {
	for _, t := range []string{"ffmpeg", "ffprobe"} {
		if _, err := lookPath(t); err != nil {
			return fmt.Errorf("%s not found on PATH — install it with: brew install ffmpeg", t)
		}
	}
	return nil
}

// Plan is the ordered set of ffmpeg commands for one trim, plus an optional
// concat-list file to materialize before the final command (used by 'x' default).
type Plan struct {
	Cmds       [][]string // each entry is the argv passed to ffmpeg
	ConcatList string     // path of the concat-list file to write, or "" if none
	ConcatBody string     // contents of that file
}

// buildPlan returns the command plan for a mode. tmpDir is used only by the
// default 'x' path for its intermediate segments and concat list.
func buildPlan(mode string, start, end int, accurate bool, in, out, tmpDir string) Plan {
	ss := strconv.Itoa(start)
	switch mode {
	case "l":
		if accurate {
			return Plan{Cmds: [][]string{{"-i", in, "-ss", ss, "-c:v", "libx264", "-c:a", "aac", out}}}
		}
		return Plan{Cmds: [][]string{{"-ss", ss, "-i", in, "-c", "copy", out}}}
	case "r":
		to := strconv.Itoa(end)
		if accurate {
			return Plan{Cmds: [][]string{{"-i", in, "-to", to, "-c:v", "libx264", "-c:a", "aac", out}}}
		}
		return Plan{Cmds: [][]string{{"-i", in, "-to", to, "-c", "copy", out}}}
	case "m":
		dur := strconv.Itoa(end - start)
		if accurate {
			return Plan{Cmds: [][]string{{"-i", in, "-ss", ss, "-t", dur, "-c:v", "libx264", "-c:a", "aac", out}}}
		}
		return Plan{Cmds: [][]string{{"-ss", ss, "-i", in, "-t", dur, "-c", "copy", out}}}
	case "x":
		if accurate {
			return Plan{Cmds: [][]string{{"-i", in, "-filter_complex", cutFilter(start, end), "-map", "[v]", "-map", "[a]", out}}}
		}
		ext := filepath.Ext(in)
		seg1 := filepath.Join(tmpDir, "seg0"+ext)
		seg2 := filepath.Join(tmpDir, "seg1"+ext)
		list := filepath.Join(tmpDir, "concat.txt")
		return Plan{
			Cmds: [][]string{
				{"-i", in, "-to", ss, "-c", "copy", seg1},
				{"-ss", strconv.Itoa(end), "-i", in, "-c", "copy", seg2},
				{"-f", "concat", "-safe", "0", "-i", list, "-c", "copy", out},
			},
			ConcatList: list,
			ConcatBody: fmt.Sprintf("file '%s'\nfile '%s'\n", seg1, seg2),
		}
	}
	return Plan{}
}

// cutFilter builds the -filter_complex graph that drops [start,end] and
// concatenates the surrounding parts, re-encoding for a frame-accurate cut.
func cutFilter(start, end int) string {
	return fmt.Sprintf(
		"[0:v]trim=0:%d,setpts=PTS-STARTPTS[v0];"+
			"[0:a]atrim=0:%d,asetpts=PTS-STARTPTS[a0];"+
			"[0:v]trim=start=%d,setpts=PTS-STARTPTS[v1];"+
			"[0:a]atrim=start=%d,asetpts=PTS-STARTPTS[a1];"+
			"[v0][a0][v1][a1]concat=n=2:v=1:a=1[v][a]",
		start, start, end, end)
}

// run validates inputs, performs the trim for mode, and prints the summary table.
// args is the subcommand's positional list: one or two timestamps followed by the
// input path.
func run(mode string, accurate bool, args []string) error {
	if err := toolsAvailable(); err != nil {
		return err
	}

	input := args[len(args)-1]
	stamps := args[:len(args)-1]
	if fi, err := os.Stat(input); err != nil || fi.IsDir() {
		return fmt.Errorf("input %q: not a readable file", input)
	}

	duration, err := probeDurationSeconds(input)
	if err != nil {
		return err
	}

	var start, end int
	switch mode {
	case "l":
		if start, err = parseAndGate(stamps[0], duration); err != nil {
			return err
		}
	case "r":
		if end, err = parseAndGate(stamps[0], duration); err != nil {
			return err
		}
	case "m", "x":
		if start, err = parseAndGate(stamps[0], duration); err != nil {
			return err
		}
		if end, err = parseAndGate(stamps[1], duration); err != nil {
			return err
		}
	}
	if err := validateBounds(mode, start, end, duration); err != nil {
		return err
	}

	output := filepath.Join(filepath.Dir(input), deriveOutputName(filepath.Base(input)))
	if _, err := os.Stat(output); err == nil {
		return fmt.Errorf("output %q already exists; refusing to overwrite", output)
	}

	// Zero-offset left trim is a straight copy; -a is moot (nothing to re-encode).
	if mode == "l" && start == 0 {
		if err := copyFile(input, output); err != nil {
			return fmt.Errorf("copying %q to %q: %w", input, output, err)
		}
		return printSummary(input, output)
	}

	tmpDir := ""
	if mode == "x" && !accurate {
		if tmpDir, err = makeTempDir(); err != nil {
			return fmt.Errorf("creating temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)
	}

	plan := buildPlan(mode, start, end, accurate, input, output, tmpDir)
	if plan.ConcatList != "" {
		if err := os.WriteFile(plan.ConcatList, []byte(plan.ConcatBody), 0o644); err != nil {
			return fmt.Errorf("writing concat list: %w", err)
		}
	}
	for _, c := range plan.Cmds {
		if err := runFFmpeg(c); err != nil {
			return fmt.Errorf("ffmpeg failed: %w", err)
		}
	}
	return printSummary(input, output)
}

// parseAndGate parses a timestamp and applies the duration gate, returning seconds.
func parseAndGate(ts string, duration float64) (int, error) {
	secs, comps, err := parseOffset(ts)
	if err != nil {
		return 0, err
	}
	if err := validateTimestamp(comps, duration); err != nil {
		return 0, fmt.Errorf("%q: %w", ts, err)
	}
	return secs, nil
}

// copyFile copies src to dst byte-for-byte.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// fileStat holds the summary-table fields for one file.
type fileStat struct {
	label   string
	name    string
	size    int64
	seconds int
}

// gatherStat collects the size and duration of path for the summary table.
func gatherStat(label, path string) (fileStat, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return fileStat{}, err
	}
	d, err := probeDurationSeconds(path)
	if err != nil {
		return fileStat{}, err
	}
	return fileStat{label: label, name: filepath.Base(path), size: fi.Size(), seconds: int(d + 0.5)}, nil
}

// printSummary prints the input-vs-output summary table.
func printSummary(input, output string) error {
	in, err := gatherStat("input", input)
	if err != nil {
		return err
	}
	out, err := gatherStat("output", output)
	if err != nil {
		return err
	}
	fmt.Print(renderSummary([]fileStat{in, out}))
	return nil
}

// renderSummary lays out the FILE/NAME/SIZE/DURATION table for the given rows.
func renderSummary(rows []fileStat) string {
	heads := []string{"FILE", "NAME", "SIZE", "DURATION"}
	right := []bool{false, false, true, true}
	cells := make([][]string, len(rows))
	for i, r := range rows {
		cells[i] = []string{r.label, r.name, formatBytes(r.size), formatDuration(r.seconds)}
	}

	widths := make([]int, len(heads))
	for c, h := range heads {
		widths[c] = len(h)
		for _, row := range cells {
			if len(row[c]) > widths[c] {
				widths[c] = len(row[c])
			}
		}
	}

	var b strings.Builder
	for c, h := range heads {
		if c > 0 {
			b.WriteString("  ")
		}
		b.WriteString(color.Whi10(pad(h, widths[c], right[c])))
	}
	b.WriteByte('\n')
	for _, row := range cells {
		for c, v := range row {
			if c > 0 {
				b.WriteString("  ")
			}
			b.WriteString(pad(v, widths[c], right[c]))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// pad left- or right-justifies s to width w.
func pad(s string, w int, right bool) string {
	if len(s) >= w {
		return s
	}
	sp := strings.Repeat(" ", w-len(s))
	if right {
		return sp + s
	}
	return s + sp
}
