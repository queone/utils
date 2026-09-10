package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// renderPath maps a store target to a relative path inside a render tree:
// a leading "~" becomes HOME, a leading "$NAME" becomes NAME, and a leading
// "/" becomes root/.
func renderPath(target string) string {
	switch {
	case target == "~":
		return "HOME"
	case strings.HasPrefix(target, "~/"):
		return filepath.Join("HOME", target[2:])
	case strings.HasPrefix(target, "$"):
		name, rest, _ := strings.Cut(target[1:], "/")
		return filepath.Join(name, rest)
	case strings.HasPrefix(target, "/"):
		return filepath.Join("root", target[1:])
	}
	return target
}

// scopeOf names the top-level render directory for an entry's host binding.
func scopeOf(host string) string {
	if host == "" {
		return "any"
	}
	return host
}

// writeRendered writes one rendered file with the entry's mode inside 0700 directories.
func writeRendered(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

// renderDir resolves the destination: a fresh private temp directory, or -o DIR.
func renderDir(out string, force bool) (string, error) {
	if out == "" {
		return os.MkdirTemp("", "macfit-render-")
	}
	dir, err := absPath(out)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(dir)
	switch {
	case os.IsNotExist(err):
		return dir, os.MkdirAll(dir, 0o700)
	case err != nil:
		return "", err
	case !info.IsDir():
		return "", fmt.Errorf("%s is not a directory", dir)
	}
	names, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	if len(names) > 0 && !force {
		return "", fmt.Errorf("%s is not empty; pass -f to write into it", dir)
	}
	return dir, nil
}

// cmdRender writes the store's latest files into a browsable directory tree.
func (a *app) cmdRender(ref storeRef, args []string) int {
	flags, pos, err := parseArgs(args, []flagSpec{{"-o", "--out", true}, {"-a", "--all", false}, {"-f", "--force", false}})
	if err != nil || len(pos) > 0 {
		a.errorf("render: usage: macfit render [-o DIR] [-a] [-f]")
		return 2
	}
	all, force := flags["--all"] == "true", flags["--force"] == "true"
	st, err := a.openStore(ref.path)
	if err != nil {
		a.errorf("render: %s", err)
		return 1
	}
	defer st.Close()
	entries, err := st.Entries()
	if err != nil {
		a.errorf("render: %s", err)
		return 1
	}
	dir, err := renderDir(flags["--out"], force)
	if err != nil {
		a.errorf("render: %s", err)
		return 1
	}
	var rows []string
	n := 0
	for _, e := range entries {
		latest, ok, err := st.Latest(e.ID)
		if err != nil {
			a.errorf("render: %s: %s", e.Target, err)
			return 1
		}
		if !ok {
			fmt.Fprintln(a.stdout, status("empty", e.Target+" (nothing stored yet)"))
			continue
		}
		rel := filepath.Join(scopeOf(e.Host), renderPath(e.Target))
		if err := writeRendered(filepath.Join(dir, rel), latest.Content, e.Mode); err != nil {
			a.errorf("render: %s: %s", rel, err)
			return 1
		}
		host := e.Host
		if host == "" {
			host = "-"
		}
		rows = append(rows, fmt.Sprintf("%s\t%s\t%s\t%04o\t%d\t%s\t%s",
			rel, e.Target, host, e.Mode, latest.Generation, latest.CapturedAt.UTC().Format(time.RFC3339), latest.SHA256))
		fmt.Fprintln(a.stdout, status("rendered", rel))
		n++
		if !all {
			continue
		}
		versions, err := st.Versions(e.ID)
		if err != nil {
			a.errorf("render: %s: %s", e.Target, err)
			return 1
		}
		for _, v := range versions {
			name := fmt.Sprintf("%d-%s", v.Generation, v.CapturedAt.UTC().Format("20060102T150405Z"))
			p := filepath.Join(dir, "versions", scopeOf(e.Host), renderPath(e.Target), name)
			if err := writeRendered(p, v.Content, e.Mode); err != nil {
				a.errorf("render: %s: %s", p, err)
				return 1
			}
		}
	}
	manifest := fmt.Sprintf("store: %s\ngeneration: %d\nrendered: %s\nnote: plaintext copies of the store; delete this directory when done\n\n",
		ref.path, st.Header.Generation, time.Now().UTC().Format(time.RFC3339))
	if len(rows) > 0 {
		manifest += strings.Join(rows, "\n") + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "MANIFEST.txt"), []byte(manifest), 0o600); err != nil {
		a.errorf("render: manifest: %s", err)
		return 1
	}
	fmt.Fprintf(a.stdout, "rendered %d file(s) into %s\n", n, dir)
	return 0
}

// cmdCat prints one entry's latest stored content to stdout, byte for byte.
func (a *app) cmdCat(ref storeRef, args []string) int {
	flags, pos, err := parseArgs(args, []flagSpec{{"-H", "--host", true}})
	if err != nil || len(pos) != 1 {
		a.errorf("cat: usage: macfit cat TARGET [-H HOST]")
		return 2
	}
	st, err := a.openStore(ref.path)
	if err != nil {
		a.errorf("cat: %s", err)
		return 1
	}
	defer st.Close()
	all, err := st.Entries()
	if err != nil {
		a.errorf("cat: %s", err)
		return 1
	}
	host, explicit := flags["--host"]
	pick := a.pickEntry(all, pos[0], host, explicit)
	if pick == nil {
		a.errorf("cat: %s is not registered%s", pos[0], forHost(host))
		return 1
	}
	latest, ok, err := st.Latest(pick.ID)
	if err != nil {
		a.errorf("cat: %s", err)
		return 1
	}
	if !ok {
		a.errorf("cat: %s has nothing stored yet", pick.Target)
		return 1
	}
	if _, err := a.stdout.Write(latest.Content); err != nil {
		a.errorf("cat: %s", err)
		return 1
	}
	return 0
}
