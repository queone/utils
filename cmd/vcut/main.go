// main.go

package main

import (
	"fmt"
	"os"
	"strings"

	color "github.com/queone/governa-color"
	"github.com/queone/utils/internal/vedit"
)

const (
	// Global constants
	programName    = "vcut"
	programVersion = "0.1.0"
)

// die prints an error message to stderr and exits with status 1.
func die(format string, args ...any) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, format)
	} else {
		fmt.Fprintf(os.Stderr, format, args...)
	}
	os.Exit(1)
}

// printUsage prints the help/version screen and exits successfully.
func printUsage() {
	n := color.Whi10(programName)
	usage := fmt.Sprintf("%s v%s\n"+
		"Remove a section of a video by driving ffmpeg — https://github.com/queone/utils/blob/main/cmd/vcut/README.md\n"+
		"%s\n"+
		"  vcut removes the part you don't want: it deletes START..END from a video and\n"+
		"  joins the remainder into a new file. (Its counterpart vclip keeps a part instead.)\n"+
		"  Timestamps are MM:SS by default (e.g. 8:31); a bare integer is whole seconds\n"+
		"  (e.g. 90); HH:MM:SS is allowed only when the source is longer than one hour. END\n"+
		"  is optional — omit it (or pass the literal 'end') to remove through to the source\n"+
		"  end. The range cannot exceed the source duration, and a cut cannot remove the\n"+
		"  entire video. The output is named by inserting '_' before the input's trailing\n"+
		"  digits (SOURCE1.mp4 -> SOURCE_1.mp4), or appending '_1' when the name has no\n"+
		"  trailing digit; vcut refuses to overwrite an existing output.\n"+
		"\n"+
		"%s\n"+
		"  vcut START [END] [-a] <input>\n"+
		"\n"+
		"%s\n"+
		"  vcut 4:13 SOURCE1.mp4       Remove from 4:13 to the end (keep the start)\n"+
		"  vcut 0 8:31 SOURCE1.mp4     Remove from the start to 8:31 (keep the rest)\n"+
		"  vcut 1:00 8:31 SOURCE1.mp4  Remove the segment 1:00..8:31, join the rest\n"+
		"\n"+
		"%s\n"+
		"  -a, --accurate  Frame-accurate cut via re-encode (default is a fast, lossless,\n"+
		"                  keyframe-snapped stream copy)\n"+
		"  -v, --version   Show this help message and exit\n"+
		"  -h, -?, --help  Show this help message and exit\n"+
		"\n"+
		"%s\n"+
		"  Requires ffmpeg and ffprobe on PATH (brew install ffmpeg).\n"+
		"  To keep a section instead of removing it, use vclip.\n",
		n, programVersion,
		color.Whi10("Overview"), color.Whi10("Usage"), color.Whi10("Examples"),
		color.Whi10("Options"), color.Whi10("Notes"))
	fmt.Print(usage)
	os.Exit(0)
}

// runCLI parses os.Args manually and dispatches a cut.
func runCLI() {
	args := os.Args[1:]
	for _, a := range args {
		if a == "-?" || a == "-h" || a == "--help" || a == "-v" || a == "--version" {
			printUsage()
			return
		}
	}
	if len(args) == 0 {
		printUsage()
		return
	}

	accurate := false
	var pos []string
	for _, a := range args {
		switch a {
		case "-a", "--accurate":
			accurate = true
		default:
			if strings.HasPrefix(a, "-") {
				die("%s: unknown flag %q (see %s --help)\n", programName, a, programName)
			}
			pos = append(pos, a)
		}
	}

	var startTok, endTok, input string
	switch len(pos) {
	case 2:
		startTok, endTok, input = pos[0], "end", pos[1]
	case 3:
		startTok, endTok, input = pos[0], pos[1], pos[2]
	default:
		die("%s: expected START [END] FILE (see %s --help)\n", programName, programName)
	}

	if err := vedit.Cut(accurate, startTok, endTok, input); err != nil {
		die("%s: %v\n", programName, err)
	}
}

func main() {
	runCLI()
}
