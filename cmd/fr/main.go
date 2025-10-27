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
	program_version = "1.0.0"
)

func init() {
	_ = program_name
	_ = program_version
}

// ---------------------------------------------------------------------
// Helpers that were already in the working version
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

func escapeForRegexp(s string) string {
	s = strings.ReplaceAll(s, "/", `\/`)
	s = strings.ReplaceAll(s, ".", `\.`)
	return s
}

func countMatches(data []byte, from string) int {
	esc := escapeForRegexp(from)
	reSlash := regexp.MustCompile(fmt.Sprintf(`/%s/`, esc))
	reDot := regexp.MustCompile(fmt.Sprintf(`\.%s\.`, esc))
	return len(reSlash.FindAllIndex(data, -1)) + len(reDot.FindAllIndex(data, -1))
}

func replaceAll(data []byte, from, to string) []byte {
	esc := escapeForRegexp(from)

	// Two regexps, each captures the surrounding character.
	reSlash := regexp.MustCompile(fmt.Sprintf(`(/)%s(/)`, esc))
	reDot := regexp.MustCompile(fmt.Sprintf(`(\.)%s(\.)`, esc))

	// Replace the “/FROM/” form.
	data = reSlash.ReplaceAllFunc(data, func(m []byte) []byte {
		sub := reSlash.FindSubmatch(m) // sub[0]=whole, sub[1]="/", sub[2]="/"
		return []byte(fmt.Sprintf("%s%s%s", sub[1], to, sub[2]))
	})

	// Replace the “.FROM.” form.
	data = reDot.ReplaceAllFunc(data, func(m []byte) []byte {
		sub := reDot.FindSubmatch(m) // sub[0]=whole, sub[1]=".", sub[2]="."
		return []byte(fmt.Sprintf("%s%s%s", sub[1], to, sub[2]))
	})

	return data
}

// ---------------------------------------------------------------------
// New helper for the “show‑only” mode
// ---------------------------------------------------------------------

// highlightLine returns the line with every occurrence of the pattern
// wrapped in utl.Red() (so it appears red on the terminal).
func highlightLine(line, from string) string {
	esc := escapeForRegexp(from)
	// One regexp that matches either /FROM/ or .FROM.
	pat := regexp.MustCompile(fmt.Sprintf(`(/%s/|\.%s\.)`, esc, esc))

	return pat.ReplaceAllStringFunc(line, func(m string) string {
		return utl.Red(m)
	})
}

// ---------------------------------------------------------------------
// main()
// ---------------------------------------------------------------------

func main() {
	// -------------------- argument handling --------------------
	//
	//   prog FROM TO          -> show‑only mode
	//   prog FROM TO -f       -> replace‑and‑write mode
	//
	if len(os.Args) != 3 && len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <FROM> <TO> [-f]\n", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	from := os.Args[1]
	to := os.Args[2]
	replaceMode := false
	if len(os.Args) == 4 {
		if os.Args[3] != "-f" {
			fmt.Fprintf(os.Stderr, "Unrecognised flag %q. Only -f is supported.\n", os.Args[3])
			os.Exit(1)
		}
		replaceMode = true
	}

	// -------------------- walk the tree --------------------
	err := filepath.Walk(".", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip hidden directories (e.g. .git, .svn, .idea)
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && path != "." {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if !isTextFile(path) {
			return nil
		}

		// Read the whole file once (fast enough for typical source/docs)
		orig, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		occ := countMatches(orig, from)
		if occ == 0 {
			return nil // nothing to act on
		}

		if replaceMode {
			// ------------ replacement mode ------------
			newContent := replaceAll(orig, from, to)

			// Write back atomically.
			tmp := path + ".tmp"
			if err := os.WriteFile(tmp, newContent, info.Mode()); err != nil {
				return err
			}
			if err := os.Rename(tmp, path); err != nil {
				return err
			}

			fmt.Printf("%s: %d occurrence(s) replaced\n", utl.Yel(path), occ)
		} else {
			// ------------ show‑only mode ------------
			yellowPath := utl.Yel(path)
			scanner := bufio.NewScanner(strings.NewReader(string(orig)))
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				line := scanner.Text()
				if countMatches([]byte(line), from) > 0 {
					hl := highlightLine(line, from)
					fmt.Printf("%s:%d: %s\n", yellowPath, lineNum, hl)
				}
			}
			// ignore scanner.Err() for simplicity – an error would be rare on a local file
		}
		return nil
	})

	if err != nil {
		log.Fatalf("walk error: %v", err)
	}
}
