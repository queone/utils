package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/color"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}

func TestNoQueoneUtlImport(t *testing.T) {
	for _, f := range []string{"main.go", "json.go", "yaml.go"} {
		body, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if strings.Contains(string(body), `"github.com/queone/utl"`) {
			t.Errorf("%s still imports github.com/queone/utl", f)
		}
	}
}

func TestJsonGoDeclaresPortedFunctions(t *testing.T) {
	body, err := os.ReadFile("json.go")
	if err != nil {
		t.Fatalf("read json.go: %v", err)
	}
	want := []string{
		"func printJson(",
		"func jsonBytesReindent(",
		"func jsonBytesToJsonObj(",
		"func prettify(",
		"func printJsonBytesColor(",
	}
	for _, w := range want {
		if !strings.Contains(string(body), w) {
			t.Errorf("json.go missing %q", w)
		}
	}
}

func TestYamlGoDeclaresPortedFunctions(t *testing.T) {
	body, err := os.ReadFile("yaml.go")
	if err != nil {
		t.Fatalf("read yaml.go: %v", err)
	}
	want := []string{
		"func yamlToBytesIndent(",
		"func yamlToBytes(",
		"func printYaml(",
		"func printYamlColor(",
		"func printYamlBytesColor(",
		"func colorizeString(",
		"blu = color.FgLightBlue.Render",
		"gre = color.FgGreen.Render",
		"yel = color.FgYellow.Render",
		"whi = color.FgWhite.Render",
		"mag = color.FgLightMagenta.Render",
	}
	for _, w := range want {
		if !strings.Contains(string(body), w) {
			t.Errorf("yaml.go missing %q", w)
		}
	}
}

func TestMainGoDeclaresHelpers(t *testing.T) {
	body, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	want := []string{
		"func die(",
		"func fileUsable(",
		"func loadFileText(",
		"func loadFileYamlBytes(",
	}
	for _, w := range want {
		if !strings.Contains(string(body), w) {
			t.Errorf("main.go missing %q", w)
		}
	}
	if strings.Contains(string(body), "utl.") {
		t.Errorf("main.go still references utl.")
	}
}

func TestJsonBytesReindent(t *testing.T) {
	got, err := jsonBytesReindent([]byte(`{"a":1,"b":2}`), 2)
	if err != nil {
		t.Fatalf("jsonBytesReindent: %v", err)
	}
	want := "{\n  \"a\": 1,\n  \"b\": 2\n}"
	if string(got) != want {
		t.Errorf("jsonBytesReindent = %q, want %q", got, want)
	}
}

func TestJsonBytesToJsonObj(t *testing.T) {
	got, err := jsonBytesToJsonObj([]byte(`{"k":"v"}`))
	if err != nil {
		t.Fatalf("jsonBytesToJsonObj: %v", err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("got type %T, want map[string]interface{}", got)
	}
	if m["k"] != "v" {
		t.Errorf("got[%q] = %v, want %q", "k", m["k"], "v")
	}
}

func TestPrintJson(t *testing.T) {
	out := captureStdout(t, func() { printJson(map[string]int{"x": 1}) })
	want := "{\n  \"x\": 1\n}\n"
	if out != want {
		t.Errorf("printJson = %q, want %q", out, want)
	}
}

func TestPrintYamlBytesColorAnsi(t *testing.T) {
	color.ForceColor() // ensure escapes emit even when stdout isn't a tty
	defer color.ResetOptions()
	out := captureStdout(t, func() { printYamlBytesColor([]byte("key: value\n")) })
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("printYamlBytesColor produced no ANSI escapes: %q", out)
	}
}

func TestPrintYamlNoAnsi(t *testing.T) {
	out := captureStdout(t, func() { printYaml(map[string]int{"x": 1}) })
	if strings.Contains(out, "\x1b[") {
		t.Errorf("printYaml emitted ANSI escapes: %q", out)
	}
	if !strings.Contains(out, "x: 1") {
		t.Errorf("printYaml output missing %q: %q", "x: 1", out)
	}
}

func TestFileUsable(t *testing.T) {
	dir := t.TempDir()

	if fileUsable(filepath.Join(dir, "nonexistent")) {
		t.Errorf("fileUsable(nonexistent) = true, want false")
	}

	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(empty, nil, 0644); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	if fileUsable(empty) {
		t.Errorf("fileUsable(empty) = true, want false")
	}

	full := filepath.Join(dir, "full")
	if err := os.WriteFile(full, []byte("content"), 0644); err != nil {
		t.Fatalf("write full: %v", err)
	}
	if !fileUsable(full) {
		t.Errorf("fileUsable(full) = false, want true")
	}
}

func TestLoadFileText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "text")
	body := []byte("hello world\n")
	if err := os.WriteFile(path, body, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := loadFileText(path)
	if err != nil {
		t.Fatalf("loadFileText: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("loadFileText = %q, want %q", got, body)
	}
}

func TestLoadFileYamlBytesMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	// Lone "&" — anchor marker with no anchor name. Errors cleanly in goccy v1.11.0
	// (most other malformed YAML is accepted leniently or panics).
	if err := os.WriteFile(path, []byte("&"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := loadFileYamlBytes(path); err == nil {
		t.Errorf("loadFileYamlBytes accepted malformed YAML")
	}
}

func TestLoadFileYamlBytesValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "good.yaml")
	body := []byte("key: value\nlist:\n  - a\n  - b\n")
	if err := os.WriteFile(path, body, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := loadFileYamlBytes(path)
	if err != nil {
		t.Fatalf("loadFileYamlBytes: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("loadFileYamlBytes = %q, want %q", got, body)
	}
}
