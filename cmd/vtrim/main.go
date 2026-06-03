// main.go

package main

import (
	"fmt"
	"os"

	color "github.com/queone/governa-color"
	"github.com/spf13/cobra"
)

const (
	// Global constants
	programName    = "vtrim"
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
		"Trim a video by driving ffmpeg — https://github.com/queone/utils/blob/main/cmd/vtrim/README.md\n"+
		"%s\n"+
		"  Trims a video to a new file. Timestamps are MM:SS by default (e.g. 8:31); a bare\n"+
		"  integer is whole seconds (e.g. 90); HH:MM:SS is allowed only when the source is\n"+
		"  longer than one hour. The trim cannot exceed the source duration. The output is\n"+
		"  named by inserting '_' before the input's trailing digits (april1.mp4 -> april_1.mp4),\n"+
		"  or appending '_1' when the name has no trailing digit; vtrim refuses to overwrite an\n"+
		"  existing output.\n"+
		"\n"+
		"%s\n"+
		"  vtrim <mode> [-a] <args> <input>\n"+
		"\n"+
		"%s\n"+
		"  l START         Left   — keep from START to the end       (vtrim l 8:31 april1.mp4)\n"+
		"  r END           Right  — keep from the start to END        (vtrim r 8:31 april1.mp4)\n"+
		"  m START END     Middle — keep the segment START..END       (vtrim m 1:00 8:31 april1.mp4)\n"+
		"  x START END     Cut    — remove START..END, join the rest  (vtrim x 1:00 8:31 april1.mp4)\n"+
		"\n"+
		"%s\n"+
		"  -a, --accurate  Frame-accurate trim via re-encode (default is a fast, lossless,\n"+
		"                  keyframe-snapped stream copy)\n"+
		"  -v, --version   Show this help message and exit\n"+
		"  -h, -?, --help  Show this help message and exit\n"+
		"\n"+
		"%s\n"+
		"  Requires ffmpeg and ffprobe on PATH (brew install ffmpeg).\n",
		n, programVersion,
		color.Whi10("Overview"), color.Whi10("Usage"), color.Whi10("Modes"),
		color.Whi10("Options"), color.Whi10("Notes"))
	fmt.Print(usage)
	os.Exit(0)
}

// runCLI wires the cobra subcommands and dispatches a trim.
func runCLI() {
	// Handle -?, -h, --help, -v, --version before cobra — -? isn't a valid pflag
	// character, and these should print the same screen from any position.
	for _, arg := range os.Args[1:] {
		if arg == "-?" || arg == "-h" || arg == "--help" || arg == "-v" || arg == "--version" {
			printUsage()
			return
		}
	}
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	var accurate bool

	root := &cobra.Command{
		Use:           programName,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVarP(&accurate, "accurate", "a", false, "Frame-accurate re-encode")

	mode := func(use string, nargs int, m string) *cobra.Command {
		return &cobra.Command{
			Use:  use,
			Args: cobra.ExactArgs(nargs),
			RunE: func(cmd *cobra.Command, args []string) error {
				return run(m, accurate, args)
			},
		}
	}
	root.AddCommand(
		mode("l", 2, "l"),
		mode("r", 2, "r"),
		mode("m", 3, "m"),
		mode("x", 3, "x"),
	)

	// Disable cobra's default help command/flag; our pre-cobra loop owns help.
	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().BoolP("help", "h", false, "")
	root.PersistentFlags().Lookup("help").Hidden = true

	if err := root.Execute(); err != nil {
		die("%s: %v\n", programName, err)
	}
}

func main() {
	runCLI()
}
