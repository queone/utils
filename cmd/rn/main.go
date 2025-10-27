package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/queone/utl"
)

const (
	program_name    = "rn"
	program_version = "1.5.0"
)

func init() {
	_ = program_name
	_ = program_version
}

func printUsage() {
	n := utl.Whi2(program_name)
	v := program_version
	usage := fmt.Sprintf("%s v%s\n"+
		"Bulk file re-namer — https://github.com/queone/utils/blob/main/cmd/rn/README.md\n"+
		"\n"+
		"%s\n"+
		"  %s \"OldString\" \"NewString\" [-f]\n"+
		"\n"+
		"  Renames all files in the current directory by replacing occurrences of OldString\n"+
		"  in filenames with NewString. If NewString is empty (\"\"), the OldString is removed.\n"+
		"\n"+
		"%s\n"+
		"  -f                     Perform actual renaming (required to make changes).\n"+
		"  -?, --help, -h         Show this help message and exit.\n"+
		"\n"+
		"%s\n"+
		"  %s \"_draft\" \"\"           Show files that would be renamed (dry run).\n"+
		"  %s \"_draft\" \"\" -f       Actually rename files.\n"+
		"  %s \"temp\" \"final\" -f     Replace one substring with another.\n"+
		"  %s -h                   Display this help message.\n",
		n, v,
		utl.Whi2("Usage"), n,
		utl.Whi2("Options"),
		utl.Whi2("Examples"),
		n, n, n, n)
	fmt.Print(usage)
	os.Exit(0)
}

func main() {
	args := os.Args[1:]
	if len(args) < 1 || len(args) > 3 {
		printUsage()
	}

	oldStr := args[0]
	newStr := ""
	option := ""

	if len(args) >= 2 {
		newStr = args[1]
	}
	if len(args) == 3 {
		option = args[2]
	}

	doRename := option == "-f"
	if !doRename {
		color.Yellow("DRY RUN: Re-run with '-f' option to execute.\n")
	}

	files, err := os.ReadDir(".")
	if err != nil {
		color.Red("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	found := false
	for _, entry := range files {
		if entry.IsDir() {
			continue // skip directories
		}

		oldName := entry.Name()
		if !strings.Contains(oldName, oldStr) {
			continue
		}

		found = true
		newName := strings.ReplaceAll(oldName, oldStr, newStr)

		if doRename {
			err := os.Rename(oldName, newName)
			if err != nil {
				color.Red("Failed to rename %s -> %s: %v\n", oldName, newName, err)
				continue
			}
			color.Green("\"%s\" -> \"%s\"\n", oldName, newName)
		} else {
			fmt.Printf("%-60s  =>  %s\n", fmt.Sprintf("\"%s\"", oldName), fmt.Sprintf("\"%s\"", newName))
		}
	}

	if !found {
		color.Red("No filename has string '%s'.\n", oldStr)
		os.Exit(1)
	}

	os.Exit(0)
}
