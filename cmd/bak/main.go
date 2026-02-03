package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	program_name    = "bak"
	program_version = "2.0.0"
)

func init() {
	_ = program_name
	_ = program_version
}

func usage() {
	fmt.Printf("Usage: %s <file|directory>\n", filepath.Base(os.Args[0]))
	os.Exit(1)
}

func nextSuffix(s string) string {
	if s == "" {
		return "a"
	}

	r := []rune(s)
	for i := len(r) - 1; i >= 0; i-- {
		if r[i] != 'z' {
			r[i]++
			return string(r[:i+1])
		}
	}
	return "a" + string(r)
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		info, err := d.Info()
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return copyFile(path, target, info.Mode())
	})
}

func main() {
	if len(os.Args) != 2 {
		usage()
	}

	src := os.Args[1]
	info, err := os.Stat(src)
	if err != nil {
		usage()
	}

	date := time.Now().Format("20060102")
	base := fmt.Sprintf("%s.%s", src, date)

	suffix := ""
	target := base

	for {
		if _, err := os.Stat(target); os.IsNotExist(err) {
			break
		}
		suffix = nextSuffix(suffix)
		target = base + suffix
	}

	if info.IsDir() {
		err = copyDir(src, target)
	} else {
		err = copyFile(src, target, info.Mode())
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "backup failed: %v\n", err)
		os.Exit(1)
	}
}
