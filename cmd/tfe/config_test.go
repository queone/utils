package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runCreds runs the credential path with the given variables and config file,
// recording what the client constructor received.
func runCreds(t *testing.T, vars map[string]string, configFile string, args ...string) (result, *[2]string) {
	t.Helper()
	t.Setenv("TFE_TOKEN", "")
	t.Setenv("TFE_ADDRESS", "")
	var out, errBuf bytes.Buffer
	var got [2]string
	code := run(args, env{
		stdout:     &out,
		stderr:     &errBuf,
		getenv:     func(k string) string { return vars[k] },
		configPath: configFile,
		newClient: func(domain, token string) (api, error) {
			got = [2]string{domain, token}
			return newFake(), nil
		},
	})
	return result{code: code, stdout: out.String(), stderr: errBuf.String()}, &got
}

func TestCredsComeFromEnvironmentFirst(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	r, got := runCreds(t, map[string]string{"TF_ORG": "acme", "TF_DOMAIN": "https://tfe.example.com", "TF_TOKEN": "tok-ENV"}, cfg, "orgs")
	if r.code != 0 || got[0] != "https://tfe.example.com" || got[1] != "tok-ENV" {
		t.Errorf("exit %d client got %v stderr %q", r.code, *got, r.stderr)
	}
	if _, err := os.Stat(cfg); err == nil {
		t.Errorf("config file was created although the environment was complete")
	}
}

func TestCredsNameTheEmptyVariable(t *testing.T) {
	for _, missing := range []string{"TF_ORG", "TF_DOMAIN", "TF_TOKEN"} {
		vars := map[string]string{"TF_ORG": "acme", "TF_DOMAIN": "https://tfe.example.com", "TF_TOKEN": "tok-ENV"}
		vars[missing] = ""
		r, got := runCreds(t, vars, filepath.Join(t.TempDir(), "config.yaml"), "orgs")
		if r.code != 1 || !strings.Contains(r.stderr, missing+" is empty") || got[1] != "" {
			t.Errorf("%s empty: exit %d stderr %q client got %v", missing, r.code, r.stderr, *got)
		}
	}
}

func TestCredsWriteSkeletonWhenConfigMissingOrEmpty(t *testing.T) {
	for _, preexisting := range []bool{false, true} {
		dir := t.TempDir()
		cfg := filepath.Join(dir, "sub", "tfe", "config.yaml")
		if preexisting {
			if err := os.MkdirAll(filepath.Dir(cfg), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(cfg, nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		r, got := runCreds(t, map[string]string{}, cfg, "orgs")
		if r.code != 1 || !strings.Contains(r.stderr, "wrote an empty config file at "+cfg) || !strings.Contains(r.stdout, cfg) || got[1] != "" {
			t.Errorf("preexisting=%t: exit %d stdout %q stderr %q", preexisting, r.code, r.stdout, r.stderr)
		}
		b, err := os.ReadFile(cfg)
		if err != nil || string(b) != configSkeleton {
			t.Errorf("preexisting=%t: skeleton not written: %v %q", preexisting, err, b)
		}
		if fi, _ := os.Stat(cfg); fi != nil && fi.Mode().Perm() != 0o600 {
			t.Errorf("preexisting=%t: mode %o, want 600", preexisting, fi.Mode().Perm())
		}
	}
}

func TestCredsLoadFromConfigFileWithoutPrintingTheToken(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(cfg, []byte("TF_ORG: acme  # org\nTF_DOMAIN: https://tfe.example.com\nTF_TOKEN: tok-FILE-SECRET\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, got := runCreds(t, map[string]string{}, cfg, "orgs")
	if r.code != 0 || got[0] != "https://tfe.example.com" || got[1] != "tok-FILE-SECRET" {
		t.Errorf("exit %d client got %v stderr %q", r.code, *got, r.stderr)
	}
	if strings.Contains(r.stdout, "tok-FILE-SECRET") || strings.Contains(r.stderr, "tok-FILE-SECRET") {
		t.Errorf("token leaked into output")
	}
	if err := os.WriteFile(cfg, []byte(configSkeleton), 0o600); err != nil {
		t.Fatal(err)
	}
	r, _ = runCreds(t, map[string]string{}, cfg, "orgs")
	if r.code != 1 || !strings.Contains(r.stderr, "TF_ORG is empty") {
		t.Errorf("unfilled skeleton: exit %d stderr %q", r.code, r.stderr)
	}
}

func TestConfigPathFollowsXDGThenHome(t *testing.T) {
	get := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if p, err := configPath(get(map[string]string{"XDG_CONFIG_HOME": "/x/cfg", "HOME": "/x/u"})); err != nil || p != "/x/cfg/tfe/config.yaml" {
		t.Errorf("xdg: %q %v", p, err)
	}
	if p, err := configPath(get(map[string]string{"HOME": "/x/u"})); err != nil || p != "/x/u/.config/tfe/config.yaml" {
		t.Errorf("home: %q %v", p, err)
	}
	if _, err := configPath(get(map[string]string{})); err == nil {
		t.Errorf("no HOME: expected an error")
	}
}
