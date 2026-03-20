package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	blue           = "\033[34m"
	reset          = "\033[0m"
	programName    = "dos2unix"
	programVersion = "2.0.0"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s FILE [-f]\n", programName)
	fmt.Fprintf(os.Stderr, "       %s -v | --version\n", programName)
	os.Exit(1)
}

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Printf("%s v%s\n", programName, programVersion)
		return
	}

	if len(os.Args) < 2 || len(os.Args) > 3 {
		usage()
	}

	path := os.Args[1]
	force := false

	if len(os.Args) == 3 {
		if os.Args[2] != "-f" {
			usage()
		}
		force = true
	}

	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open error: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	if force {
		// Read entire file, convert, write back
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			os.Exit(1)
		}

		converted := strings.ReplaceAll(string(data), "\r\n", "\n")

		if err := os.WriteFile(path, []byte(converted), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Preview mode: cat file and highlight CRLF
	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if before, ok := strings.CutSuffix(line, "\r\n"); ok {
				// Show the CRLF explicitly in blue
				fmt.Print(before)
				fmt.Print(blue + "\\r\\n" + reset)
				fmt.Print("\n")
			} else {
				fmt.Print(line)
			}
		}

		if err != nil {
			break
		}
	}
}
