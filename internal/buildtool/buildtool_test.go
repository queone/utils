package buildtool

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

// fakeCmdRunner records calls and returns preconfigured results.
type fakeCmdRunner struct {
	fmtOutput       string
	vetFailed       bool
	vetOutput       string
	scFailed        bool
	scOutput        string
	capturedResults map[string]capturedResult
	streamingErr    map[string]error
}

type capturedResult struct {
	output string
	err    error
}

func newFakeRunner() *fakeCmdRunner {
	return &fakeCmdRunner{
		capturedResults: map[string]capturedResult{
			"go list -m -f {{.Path}}": {output: "github.com/queone/utils\n"},
			"go env GOPATH":           {output: "/tmp/fakego\n"},
			"git tag --list":          {output: "v0.5.0\n"},
			"go tool cover -func=":    {output: "total:\t(statements)\t50.0%\n"},
		},
		streamingErr: map[string]error{},
	}
}

func (f *fakeCmdRunner) Streaming(out, errOut io.Writer, name string, args ...string) error {
	key := name + " " + strings.Join(args, " ")
	for prefix, err := range f.streamingErr {
		if strings.HasPrefix(key, prefix) {
			return err
		}
	}
	return nil
}

func (f *fakeCmdRunner) CapturedSoft(name string, args ...string) string {
	key := name + " " + strings.Join(args, " ")
	if strings.HasPrefix(key, "go fmt") {
		return f.fmtOutput
	}
	if strings.HasPrefix(key, "go fix") {
		return ""
	}
	return ""
}

func (f *fakeCmdRunner) CapturedCheck(name string, args ...string) (string, bool) {
	key := name + " " + strings.Join(args, " ")
	if strings.HasPrefix(key, "go vet") {
		return f.vetOutput, f.vetFailed
	}
	if strings.Contains(name, "staticcheck") {
		return f.scOutput, f.scFailed
	}
	return "", false
}

func (f *fakeCmdRunner) Captured(name string, args ...string) (string, error) {
	key := name + " " + strings.Join(args, " ")
	for prefix, result := range f.capturedResults {
		if strings.HasPrefix(key, prefix) {
			return result.output, result.err
		}
	}
	return "", nil
}

func TestPipelineRun_fmtFailure(t *testing.T) {
	fake := newFakeRunner()
	fake.fmtOutput = "internal/color/color.go\n"

	p := &Pipeline{Cmd: fake}
	var out, errOut bytes.Buffer
	err := p.Run(Config{}, &out, &errOut)

	if err == nil {
		t.Fatal("expected error from go fmt failure")
	}
	if !strings.Contains(err.Error(), "go fmt found files that need formatting") {
		t.Errorf("error = %q, want message about go fmt", err.Error())
	}
}

func TestPipelineRun_fmtClean(t *testing.T) {
	fake := newFakeRunner()
	fake.fmtOutput = ""

	p := &Pipeline{Cmd: fake}
	var out, errOut bytes.Buffer
	err := p.Run(Config{}, &out, &errOut)

	if err != nil && strings.Contains(err.Error(), "go fmt") {
		t.Errorf("pipeline failed at go fmt with clean code: %v", err)
	}
}

func TestPipelineRun_vetFailure(t *testing.T) {
	fake := newFakeRunner()
	fake.vetFailed = true
	fake.vetOutput = "cmd/fr/main.go:5: unreachable code"

	p := &Pipeline{Cmd: fake}
	var out, errOut bytes.Buffer
	err := p.Run(Config{}, &out, &errOut)

	if err == nil {
		t.Fatal("expected error from go vet failure")
	}
	if !strings.Contains(err.Error(), "go vet found issues") {
		t.Errorf("error = %q, want message about go vet", err.Error())
	}
}

func TestPipelineRun_staticcheckFailure(t *testing.T) {
	fake := newFakeRunner()
	fake.scFailed = true
	fake.scOutput = "internal/color/color.go:10: SA1000 - some issue"

	p := &Pipeline{Cmd: fake}
	var out, errOut bytes.Buffer
	err := p.Run(Config{}, &out, &errOut)

	if err == nil {
		t.Fatal("expected error from staticcheck failure")
	}
	if !strings.Contains(err.Error(), "staticcheck found issues") {
		t.Errorf("error = %q, want message about staticcheck", err.Error())
	}
}

func TestFilterInstallTargets(t *testing.T) {
	got := FilterInstallTargets([]string{"fr", "build", "rel"})
	if len(got) != 1 || got[0] != "fr" {
		t.Errorf("FilterInstallTargets = %v, want [fr]", got)
	}
}

func TestFilterInstallTargets_empty(t *testing.T) {
	got := FilterInstallTargets([]string{"build", "rel"})
	if len(got) != 0 {
		t.Errorf("FilterInstallTargets = %v, want []", got)
	}
}

func TestFilterInstallTargets_allProduct(t *testing.T) {
	got := FilterInstallTargets([]string{"tree", "fr", "bak"})
	want := []string{"bak", "fr", "tree"}
	if len(got) != len(want) {
		t.Fatalf("FilterInstallTargets = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("FilterInstallTargets[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDomainCoverageFromBytes(t *testing.T) {
	profile := []byte(`mode: set
github.com/queone/utils/internal/color/color.go:10.1,12.1 2 1
github.com/queone/utils/internal/color/color.go:14.1,16.1 2 0
github.com/queone/utils/cmd/fr/main.go:5.1,7.1 2 0
`)
	pct, err := DomainCoverageFromBytes(profile, "github.com/queone/utils/internal/")
	if err != nil {
		t.Fatalf("DomainCoverageFromBytes: %v", err)
	}
	if pct != 50.0 {
		t.Errorf("domain coverage = %.1f%%, want 50.0%%", pct)
	}
}

func TestDomainCoverageFromBytes_noDomainLines(t *testing.T) {
	profile := []byte(`mode: set
github.com/queone/utils/cmd/fr/main.go:5.1,7.1 2 1
`)
	pct, err := DomainCoverageFromBytes(profile, "github.com/queone/utils/internal/")
	if err != nil {
		t.Fatalf("DomainCoverageFromBytes: %v", err)
	}
	if pct != 0 {
		t.Errorf("domain coverage = %.1f%%, want 0%%", pct)
	}
}

func TestCoverageColor_thresholds(t *testing.T) {
	cases := []struct {
		pct  float64
		text string
	}{
		{80.0, "domain coverage: 80.0%"},
		{50.0, "domain coverage: 50.0%"},
		{30.0, "domain coverage: 30.0%"},
	}
	for _, tc := range cases {
		got := CoverageColor(tc.pct, tc.text)
		if !strings.Contains(got, tc.text) {
			t.Errorf("CoverageColor(%.1f, %q) = %q, does not contain input", tc.pct, tc.text, got)
		}
	}
}

func TestNextPatchTagFromOutput(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{"normal", "v0.4.0\nv0.5.0\nv0.5.6\n", "v0.5.7", true},
		{"unordered", "v0.5.6\nv0.4.0\nv0.5.0\n", "v0.5.7", true},
		{"single", "v1.0.0\n", "v1.0.1", true},
		{"no tags", "", "", false},
		{"non-semver ignored", "latest\nv0.5.0\nfoo\n", "v0.5.1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok, err := NextPatchTagFromOutput(tc.input)
			if err != nil {
				t.Fatalf("NextPatchTagFromOutput: %v", err)
			}
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if got != tc.want {
				t.Errorf("tag = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantCfg Config
		wantH   bool
		wantErr bool
	}{
		{"no args", nil, Config{}, false, false},
		{"help", []string{"-h"}, Config{}, true, false},
		{"verbose", []string{"-v"}, Config{Verbose: true}, false, false},
		{"targets", []string{"fr"}, Config{Targets: []string{"fr"}}, false, false},
		{"verbose+target", []string{"-v", "fr"}, Config{Verbose: true, Targets: []string{"fr"}}, false, false},
		{"bad flag", []string{"--unknown"}, Config{}, false, true},
		{"help not alone", []string{"-h", "fr"}, Config{}, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, help, err := ParseArgs(tc.args)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if help != tc.wantH {
				t.Errorf("help = %v, want %v", help, tc.wantH)
			}
			if !tc.wantErr {
				if cfg.Verbose != tc.wantCfg.Verbose {
					t.Errorf("Verbose = %v, want %v", cfg.Verbose, tc.wantCfg.Verbose)
				}
			}
		})
	}
}

func TestExtractProgramVersion(t *testing.T) {
	tmp, err := os.CreateTemp("", "main-*.go")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	fmt.Fprint(tmp, `package main

const programVersion = "2.0.0"

func main() {}
`)
	tmp.Close()

	got := extractProgramVersion(tmp.Name())
	if got != "2.0.0" {
		t.Errorf("extractProgramVersion = %q, want %q", got, "2.0.0")
	}
}

func TestExtractProgramVersion_missing(t *testing.T) {
	got := extractProgramVersion("/nonexistent/path/main.go")
	if got != "unknown" {
		t.Errorf("extractProgramVersion = %q, want %q", got, "unknown")
	}
}
