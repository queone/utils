package reltool

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantCfg Config
		wantH   bool
		wantErr string
	}{
		{"no args shows help", nil, Config{}, true, ""},
		{"help flag", []string{"-h"}, Config{}, true, ""},
		{"help ?", []string{"-?"}, Config{}, true, ""},
		{"help long", []string{"--help"}, Config{}, true, ""},
		{"valid", []string{"v1.0.0", "initial release"}, Config{Tag: "v1.0.0", Message: "initial release"}, false, ""},
		{"empty message", []string{"v1.0.0", ""}, Config{}, false, "non-empty"},
		{"whitespace message", []string{"v1.0.0", "  "}, Config{}, false, "non-empty"},
		{"bad semver", []string{"notasemver", "msg"}, Config{}, false, "vMAJOR.MINOR.PATCH"},
		{"missing v prefix", []string{"1.0.0", "msg"}, Config{}, false, "vMAJOR.MINOR.PATCH"},
		{"too many args", []string{"v1.0.0", "msg", "extra"}, Config{}, false, "usage:"},
		{"one non-help arg", []string{"v1.0.0"}, Config{}, false, "usage:"},
		{"unsupported flag", []string{"--force", "v1.0.0"}, Config{}, false, "unsupported"},
		{"help not alone", []string{"-h", "v1.0.0"}, Config{}, false, "help flags must be used by themselves"},
		{"long message", []string{"v1.0.0", strings.Repeat("x", 81)}, Config{}, false, "80 characters"},
		{"exact 80", []string{"v1.0.0", strings.Repeat("x", 80)}, Config{Tag: "v1.0.0", Message: strings.Repeat("x", 80)}, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, help, err := ParseArgs(tc.args)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if help != tc.wantH {
				t.Errorf("help = %v, want %v", help, tc.wantH)
			}
			if cfg.Tag != tc.wantCfg.Tag {
				t.Errorf("Tag = %q, want %q", cfg.Tag, tc.wantCfg.Tag)
			}
			if cfg.Message != tc.wantCfg.Message {
				t.Errorf("Message = %q, want %q", cfg.Message, tc.wantCfg.Message)
			}
		})
	}
}

// fakeGitRunner records calls and can fail at a specific step.
type fakeGitRunner struct {
	calls   []string
	failAt  string
	failErr error
}

func (f *fakeGitRunner) RunGit(out, errOut io.Writer, name string, args ...string) error {
	f.calls = append(f.calls, name)
	fmt.Fprintf(out, "running: git %s\n", strings.Join(args, " "))
	if name == f.failAt {
		return f.failErr
	}
	return nil
}

func alwaysConfirm(in io.Reader, out io.Writer, prompt string) (bool, error) {
	return true, nil
}

func neverConfirm(in io.Reader, out io.Writer, prompt string) (bool, error) {
	return false, nil
}

func noopGitRepo() error { return nil }

func TestReleaseRun_tagCreatedPushFails(t *testing.T) {
	fake := &fakeGitRunner{
		failAt:  "git push tag",
		failErr: errors.New("network timeout"),
	}
	r := &Release{
		Git:          fake,
		Confirm:      alwaysConfirm,
		CheckGitRepo: noopGitRepo,
	}
	var out strings.Builder
	err := r.Run(Config{Tag: "v0.6.0", Message: "test"}, strings.NewReader(""), &out, io.Discard)
	if err == nil {
		t.Fatal("expected error from push tag failure")
	}
	msg := err.Error()

	if !strings.Contains(msg, "git push tag failed") {
		t.Error("missing failed step name")
	}
	if !strings.Contains(msg, "completed before failure: git add, git commit, git tag") {
		t.Errorf("wrong completed steps in: %s", msg)
	}
	if !strings.Contains(msg, "v0.6.0 exists locally but was not pushed") {
		t.Error("missing local tag recovery message")
	}
	if !strings.Contains(msg, "git push origin v0.6.0") {
		t.Error("missing retry push instruction")
	}
	if !strings.Contains(msg, "git tag -d v0.6.0") {
		t.Error("missing tag removal instruction")
	}
}

func TestReleaseRun_branchPushFails(t *testing.T) {
	fake := &fakeGitRunner{
		failAt:  "git push branch",
		failErr: errors.New("network error"),
	}
	r := &Release{
		Git:          fake,
		Confirm:      alwaysConfirm,
		CheckGitRepo: noopGitRepo,
	}
	var out strings.Builder
	err := r.Run(Config{Tag: "v0.6.0", Message: "test"}, strings.NewReader(""), &out, io.Discard)
	if err == nil {
		t.Fatal("expected error from branch push failure")
	}
	msg := err.Error()

	if !strings.Contains(msg, "git push branch failed") {
		t.Error("missing failed step name")
	}
	if !strings.Contains(msg, "v0.6.0 was pushed but the branch push failed") {
		t.Error("missing branch push recovery message")
	}
	if !strings.Contains(msg, "to retry: git push origin") {
		t.Error("missing branch push retry instruction")
	}
	if strings.Contains(msg, "git tag -d") {
		t.Error("should not suggest tag removal when tag was already pushed")
	}
}

func TestReleaseRun_commitFails(t *testing.T) {
	fake := &fakeGitRunner{
		failAt:  "git commit",
		failErr: errors.New("nothing to commit"),
	}
	r := &Release{
		Git:          fake,
		Confirm:      alwaysConfirm,
		CheckGitRepo: noopGitRepo,
	}
	var out strings.Builder
	err := r.Run(Config{Tag: "v0.6.0", Message: "test"}, strings.NewReader(""), &out, io.Discard)
	if err == nil {
		t.Fatal("expected error from commit failure")
	}
	msg := err.Error()

	if !strings.Contains(msg, "git commit failed") {
		t.Error("missing failed step name")
	}
	if strings.Contains(msg, "git tag -d") {
		t.Error("should not include tag removal when tag was never created")
	}
}

func TestReleaseRun_abortOnDecline(t *testing.T) {
	fake := &fakeGitRunner{}
	r := &Release{
		Git:          fake,
		Confirm:      neverConfirm,
		CheckGitRepo: noopGitRepo,
	}
	var out strings.Builder
	err := r.Run(Config{Tag: "v0.6.0", Message: "test"}, strings.NewReader(""), &out, io.Discard)
	if err == nil {
		t.Fatal("expected error from abort")
	}
	if !strings.Contains(err.Error(), "release aborted") {
		t.Errorf("error = %q, want 'release aborted'", err.Error())
	}
	for _, call := range fake.calls {
		if call == "git add" || call == "git commit" || call == "git tag" {
			t.Errorf("git mutation step %q was called after abort", call)
		}
	}
}

func TestReleaseRun_success(t *testing.T) {
	fake := &fakeGitRunner{}
	r := &Release{
		Git:          fake,
		Confirm:      alwaysConfirm,
		CheckGitRepo: noopGitRepo,
	}
	var out strings.Builder
	err := r.Run(Config{Tag: "v0.6.0", Message: "test release"}, strings.NewReader(""), &out, io.Discard)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantSteps := []string{"git status preview", "git add", "git commit", "git tag", "git push tag", "git push branch"}
	if len(fake.calls) != len(wantSteps) {
		t.Fatalf("calls = %v, want %v", fake.calls, wantSteps)
	}
	for i, want := range wantSteps {
		if fake.calls[i] != want {
			t.Errorf("step %d = %q, want %q", i, fake.calls[i], want)
		}
	}
}

func TestUsage(t *testing.T) {
	out := Usage()
	if !strings.Contains(out, "rel") {
		t.Error("usage should mention 'rel'")
	}
	if !strings.Contains(out, "80 characters") {
		t.Error("usage should mention 80 character limit")
	}
}
