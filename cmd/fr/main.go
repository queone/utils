package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitfield/script"
	"github.com/queone/utl"
)

const (
	program_name    = "fr"
	program_version = "1.0.2"
)

func init() {
	_ = program_name
	_ = program_version
}

// ---------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------

func isTextFile(path string) bool {
	out, err := script.Exec(fmt.Sprintf(`file -b --mime-type "%s"`, path)).String()
	if err != nil {
		return false
	}
	mime := strings.TrimSpace(out)
	if mime == "application/xml" || mime == "application/json" {
		return true
	}
	return strings.HasPrefix(mime, "text/")
}

func countMatches(data []byte, pattern string) int {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0 // invalid regex
	}
	return len(re.FindAllIndex(data, -1))
}

func replaceAll(data []byte, pattern, to string) []byte {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return data // invalid regex
	}
	return re.ReplaceAll(data, []byte(to))
}

func highlightLine(line, pattern string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return line // invalid regex
	}
	return re.ReplaceAllStringFunc(line, func(m string) string {
		return utl.Red(m)
	})
}

// ---------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------

func main() {
	var from, to string
	var replaceMode, singleMode bool

	switch len(os.Args) {
	case 2:
		// single-argument search
		from = os.Args[1]
		singleMode = true
	case 3:
		// show-only mode
		from = os.Args[1]
		to = os.Args[2]
	case 4:
		// replace-and-write mode
		from = os.Args[1]
		to = os.Args[2]
		if os.Args[3] != "-f" {
			fmt.Fprintf(os.Stderr, "Unrecognised flag %q. Only -f is supported.\n", os.Args[3])
			os.Exit(1)
		}
		replaceMode = true
	default:
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s <REGEX>                -> search-only mode\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s <FROM> <TO>            -> show-only mode\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "  %s <FROM> <TO> -f         -> replace-and-write mode\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	// -------------------- walk the tree --------------------
	err := filepath.Walk(".", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && path != "." {
			return filepath.SkipDir
		}
		if info.IsDir() || !info.Mode().IsRegular() || !isTextFile(path) {
			return nil
		}

		orig, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		occ := countMatches(orig, from)
		if occ == 0 {
			return nil
		}

		if replaceMode {
			// ---------------- replacement mode ----------------
			newContent := replaceAll(orig, from, to)
			tmp := path + ".tmp"
			if err := os.WriteFile(tmp, newContent, info.Mode()); err != nil {
				return err
			}
			if err := os.Rename(tmp, path); err != nil {
				return err
			}
			fmt.Printf("%s: %d occurrence(s) replaced\n", utl.Yel(path), occ)
		} else if singleMode || (!replaceMode && !singleMode) {
			// ---------------- show-only or single-arg search ----------------
			scanner := bufio.NewScanner(strings.NewReader(string(orig)))
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				line := scanner.Text()
				if countMatches([]byte(line), from) > 0 {
					hl := highlightLine(line, from)
					fmt.Printf("%s:%d: %s\n", utl.Yel(path), lineNum, hl)
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Fatalf("walk error: %v", err)
	}
}
