// main.go

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/queone/gkit/internal/color"
)

const (
	programName    = "tfe"
	programVersion = "2.0.0"
)

// env carries the streams, the environment, the config path, and the client
// constructor so tests can replace every outside contact.
type env struct {
	stdout     io.Writer
	stderr     io.Writer
	getenv     func(string) string
	configPath string
	newClient  func(domain, token string) (api, error)
}

// usage returns the help screen.
func usage() string {
	lines := []color.UsageLine{
		{Flag: "-a, --all", Desc: "mods: list every version instead of the latest"},
		{Flag: "-j, --json", Desc: "mods: print a single matching module as JSON"},
		{Flag: "-v, --version", Desc: "Print " + programName + " v" + programVersion + " and exit"},
		{Flag: "-h, --help", Desc: "Show this help"},
	}
	footer := `Subcommands:
  orgs [FILTER]            List organizations
  mods [-a] [-j] [FILTER]  List registry modules; one match prints its details
  ws [FILTER]              List workspaces
  show NAME                Show one workspace with its variables
  clone SRC DEST           Clone workspace SRC as DEST, variables included

FILTER is a case-insensitive substring of the name.

Authentication, in this order:
  1. TF_ORG, TF_DOMAIN, and TF_TOKEN in the environment, all three set.
  2. The same three keys in $XDG_CONFIG_HOME/tfe/config.yaml, or
     ~/.config/tfe/config.yaml. A missing file is created as a skeleton.

Examples:
  tfe orgs
  tfe mods -a network
  tfe show prod-network
  tfe clone prod-network staging-network`
	h := color.Whi10
	return fmt.Sprintf("%s v%s\n"+
		"Terraform Cloud command-line utility.\n"+
		"\n"+
		"%s\n"+
		"  tfe lists organizations, registry modules, and workspaces of a Terraform\n"+
		"  Cloud or Terraform Enterprise instance, shows one workspace with its\n"+
		"  variables, and clones a workspace, which the web interface cannot do.\n"+
		"\n"+
		"%s",
		h(programName), programVersion, h("Overview"),
		color.FormatUsage(programName+" [flags] SUBCOMMAND [ARGS]", lines, footer))
}

// fail prints err with the program name and returns the failure exit code.
func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "%s: %v\n", programName, err)
	return 1
}

// registryHost returns the host part of the configured domain, used as the
// module address prefix; a bare host name is returned unchanged.
func registryHost(domain string) string {
	if u, err := url.Parse(domain); err == nil && u.Host != "" {
		return u.Host
	}
	return strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://"), "/")
}

// request is one parsed command line.
type request struct {
	sub  string
	args []string
	all  bool
	json bool
}

// parseArgs separates the subcommand, its flags, and its positional arguments.
func parseArgs(args []string) (request, error) {
	var r request
	for _, a := range args {
		switch {
		case a == "-a" || a == "--all":
			r.all = true
		case a == "-j" || a == "--json":
			r.json = true
		case strings.HasPrefix(a, "-") && a != "-":
			return r, fmt.Errorf("unknown flag %q (see %s --help)", a, programName)
		case r.sub == "":
			r.sub = a
		default:
			r.args = append(r.args, a)
		}
	}
	if r.sub == "" {
		return r, errors.New("missing subcommand")
	}
	if (r.all || r.json) && r.sub != "mods" {
		return r, fmt.Errorf("-a/--all and -j/--json apply to mods only (see %s --help)", programName)
	}
	return r, nil
}

// run executes the utility with the given arguments and returns its exit code.
func run(args []string, e env) int {
	if len(args) == 0 {
		fmt.Fprint(e.stdout, usage())
		return 0
	}
	for _, a := range args {
		switch a {
		case "-v", "--version":
			fmt.Fprintf(e.stdout, "%s v%s\n", programName, programVersion)
			return 0
		case "-h", "-?", "--help":
			fmt.Fprint(e.stdout, usage())
			return 0
		}
	}
	r, err := parseArgs(args)
	if err != nil {
		return fail(e.stderr, err)
	}
	var want int
	switch r.sub {
	case "orgs", "mods", "ws":
		if len(r.args) > 1 {
			return fail(e.stderr, fmt.Errorf("%s takes at most one FILTER (see %s --help)", r.sub, programName))
		}
	case "show":
		want = 1
	case "clone":
		want = 2
	default:
		return fail(e.stderr, fmt.Errorf("unknown subcommand %q (see %s --help)", r.sub, programName))
	}
	if want > 0 && len(r.args) != want {
		return fail(e.stderr, fmt.Errorf("%s needs %d argument(s) (see %s --help)", r.sub, want, programName))
	}

	c, err := resolveCreds(e.getenv, e.configPath, e.stdout)
	if err != nil {
		return fail(e.stderr, err)
	}
	a, err := e.newClient(c.domain, c.token)
	if err != nil {
		return fail(e.stderr, err)
	}
	ctx := context.Background()
	filter := ""
	if len(r.args) == 1 {
		filter = r.args[0]
	}
	switch r.sub {
	case "orgs":
		err = listOrganizations(ctx, a, filter, e.stdout)
	case "mods":
		err = listModules(ctx, a, c.org, registryHost(c.domain), filter, r.all, r.json, e.stdout)
	case "ws":
		err = listWorkspaces(ctx, a, c.org, filter, e.stdout)
	case "show":
		err = showWorkspace(ctx, a, c.org, r.args[0], e.stdout)
	case "clone":
		err = cloneWorkspace(ctx, a, c.org, r.args[0], r.args[1], e.stdout)
	}
	if err != nil {
		return fail(e.stderr, err)
	}
	return 0
}

func main() {
	path, err := configPath(os.Getenv)
	if err != nil {
		os.Exit(fail(os.Stderr, err))
	}
	os.Exit(run(os.Args[1:], env{
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		getenv:     os.Getenv,
		configPath: path,
		newClient:  newClient,
	}))
}
