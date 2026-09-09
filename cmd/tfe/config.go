// config.go

package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// creds holds the three values needed to reach one Terraform Cloud organization.
type creds struct {
	org    string
	domain string
	token  string
}

// configSkeleton is written to a missing or empty config file so the user can fill it in.
const configSkeleton = "TF_ORG:     # TFE Organization name (MYORG, etc)\n" +
	"TF_DOMAIN:  # TFE domain name (https://app.terraform.io, etc)\n" +
	"TF_TOKEN:   # Security token to access the respective TFE instance\n"

// configPath returns $XDG_CONFIG_HOME/tfe/config.yaml, or ~/.config/tfe/config.yaml
// when XDG_CONFIG_HOME is unset.
func configPath(getenv func(string) string) (string, error) {
	base := getenv("XDG_CONFIG_HOME")
	if base == "" {
		home := getenv("HOME")
		if home == "" {
			return "", errors.New("neither XDG_CONFIG_HOME nor HOME is set; cannot locate the config file")
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, programName, "config.yaml"), nil
}

// validate reports the first empty credential by its variable name.
func (c creds) validate() error {
	for _, kv := range []struct{ name, value string }{
		{"TF_ORG", c.org}, {"TF_DOMAIN", c.domain}, {"TF_TOKEN", c.token},
	} {
		if strings.TrimSpace(kv.value) == "" {
			return fmt.Errorf("%s is empty; set TF_ORG, TF_DOMAIN, and TF_TOKEN in the environment or in the config file", kv.name)
		}
	}
	return nil
}

// resolveCreds reads the three variables from the environment when any of them
// is set, otherwise from the config file at path. A missing or empty config
// file is replaced by the skeleton, and the caller is told where to fill it in.
func resolveCreds(getenv func(string) string, path string, stdout io.Writer) (creds, error) {
	c := creds{org: getenv("TF_ORG"), domain: getenv("TF_DOMAIN"), token: getenv("TF_TOKEN")}
	if c.org != "" || c.domain != "" || c.token != "" {
		return c, c.validate()
	}
	fmt.Fprintf(stdout, "TF_ORG, TF_DOMAIN, and TF_TOKEN are not set; reading %s\n", path)
	fi, err := os.Stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist) || (err == nil && fi.Size() == 0):
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return creds{}, fmt.Errorf("creating the config directory for %s: %w", path, err)
		}
		if err := os.WriteFile(path, []byte(configSkeleton), 0o600); err != nil {
			return creds{}, fmt.Errorf("writing the config file skeleton at %s: %w", path, err)
		}
		return creds{}, fmt.Errorf("wrote an empty config file at %s; fill in the three values and run again", path)
	case err != nil:
		return creds{}, fmt.Errorf("reading the config file %s: %w", path, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return creds{}, fmt.Errorf("reading the config file %s: %w", path, err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return creds{}, fmt.Errorf("parsing the config file %s: %w", path, err)
	}
	c = creds{org: str(m["TF_ORG"]), domain: str(m["TF_DOMAIN"]), token: str(m["TF_TOKEN"])}
	if err := c.validate(); err != nil {
		return creds{}, fmt.Errorf("config file %s: %w", path, err)
	}
	return c, nil
}

// str renders a YAML scalar as a string; a missing or null value is empty.
func str(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}
