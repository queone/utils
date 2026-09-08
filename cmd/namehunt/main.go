package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/queone/gkit/internal/color"
)

const (
	programName    = "namehunt"
	programVersion = "1.0.0"
)

const (
	requestTimeout  = 5 * time.Second
	initialBackoff  = 100 * time.Millisecond
	maxAttempts     = 5
	defaultFreeCode = http.StatusNotFound
	sourceBuiltIn   = "built-in"
	sourceFile      = "file"
	sourceArgument  = "argument"
)

// site pairs a name with the profile-URL template used to look usernames up.
type site struct {
	name     string
	template string
	free     []int  // status codes meaning free; nil means the default
	source   string // sourceBuiltIn, sourceFile, or sourceArgument
}

// builtinSites are always known, even without a sites file.
var builtinSites = []site{
	{name: "github", template: "https://github.com/{}", source: sourceBuiltIn},
	{name: "lichess", template: "https://lichess.org/@/{}", source: sourceBuiltIn},
}

// action is what a parsed command line asks the utility to do.
type action int

const (
	actionCheck action = iota
	actionList
	actionHelp
	actionVersion
)

// options holds the parsed command-line flags and positionals.
type options struct {
	distinct  bool
	require   string
	dryRun    bool
	free      []int // nil when -f was not given
	wait      time.Duration
	site      string
	candidate string
}

// env holds the streams, config path, HTTP client, and sleep function run uses.
type env struct {
	stdin     io.Reader
	stdout    io.Writer
	stderr    io.Writer
	sitesPath string
	client    *http.Client
	sleep     func(time.Duration)
}

// usage returns the help screen.
func usage() string {
	lines := []color.UsageLine{
		{Flag: "-d, --distinct", Desc: "Keep only names with no repeated character"},
		{Flag: "-r, --require CHARS", Desc: "Keep only names containing every character in CHARS"},
		{Flag: "-n, --dry-run", Desc: "Print the names and send no request"},
		{Flag: "-f, --free CODES", Desc: "Status codes that mean free, comma-separated (default 404)"},
		{Flag: "-w, --wait MS", Desc: "Pause MS milliseconds between requests (default 0)"},
		{Flag: "-l, --list", Desc: "List known sites and exit"},
		{Flag: "-v, --version", Desc: "Print " + programName + " v" + programVersion + " and exit"},
		{Flag: "-h, --help", Desc: "Show this help"},
	}
	footer := `SITE is a site name or a URL template with {} where the username goes.
Built-in sites: github, lichess. More come from ~/.config/namehunt/sites,
one per line: NAME TEMPLATE [FREE_CODES].
CANDIDATE is one name, a pattern like [qk][aeou][qk][aeou], or - for stdin.

Examples:
  namehunt github kaqe
  namehunt -d lichess '[qk][aeiou][qk][aeiou]'
  namehunt 'https://www.reddit.com/user/{}' kaqe`
	h := color.Whi10
	return fmt.Sprintf("%s v%s\n"+
		"Find free usernames on any site with a predictable profile URL.\n"+
		"\n"+
		"%s\n"+
		"  namehunt sends one HEAD request per name to a site's profile URL and\n"+
		"  reads the status: 404 means free, any 2xx means taken, anything else is\n"+
		"  reported as unknown. Give it one name for a yes/no answer, or a pattern\n"+
		"  like [qk][aeou][qk][aeou] to list every free name. github and lichess\n"+
		"  are built in; name more sites in ~/.config/namehunt/sites.\n"+
		"\n"+
		"%s",
		h(programName), programVersion, h("Overview"),
		color.FormatUsage(programName+" [flags] SITE CANDIDATE", lines, footer))
}

// parseArgs parses flags, which come before positionals, and the SITE and CANDIDATE positionals.
// No arguments at all asks for the help screen.
func parseArgs(args []string) (options, action, error) {
	var o options
	act := actionCheck
	if len(args) == 0 {
		return o, actionHelp, nil
	}
	for len(args) > 0 && strings.HasPrefix(args[0], "-") && args[0] != "-" {
		switch args[0] {
		case "-h", "-?", "--help":
			return o, actionHelp, nil
		case "-v", "--version":
			return o, actionVersion, nil
		case "-l", "--list":
			act = actionList
			args = args[1:]
		case "-d", "--distinct":
			o.distinct = true
			args = args[1:]
		case "-n", "--dry-run":
			o.dryRun = true
			args = args[1:]
		case "-r", "--require", "-f", "--free", "-w", "--wait":
			flag := args[0]
			if len(args) < 2 {
				return o, act, fmt.Errorf("%s requires a value", flag)
			}
			value := args[1]
			switch flag {
			case "-r", "--require":
				if value == "" {
					return o, act, fmt.Errorf("%s requires at least one character", flag)
				}
				o.require = value
			case "-f", "--free":
				codes, err := parseCodes(value)
				if err != nil {
					return o, act, fmt.Errorf("%s: %v", flag, err)
				}
				o.free = codes
			default:
				ms, err := strconv.Atoi(value)
				if err != nil || ms < 0 {
					return o, act, fmt.Errorf("%s requires a non-negative integer, got %q", flag, value)
				}
				o.wait = time.Duration(ms) * time.Millisecond
			}
			args = args[2:]
		default:
			return o, act, fmt.Errorf("unknown option %s", args[0])
		}
	}
	if act == actionList {
		if len(args) != 0 {
			return o, act, fmt.Errorf("-l takes no arguments")
		}
		return o, act, nil
	}
	if len(args) != 2 {
		return o, act, fmt.Errorf("expected SITE and CANDIDATE arguments")
	}
	o.site, o.candidate = args[0], args[1]
	return o, act, nil
}

// parseCodes parses a comma-separated list of HTTP status codes.
func parseCodes(s string) ([]int, error) {
	var codes []int
	for part := range strings.SplitSeq(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || n < 100 || n > 599 {
			return nil, fmt.Errorf("invalid status code %q", part)
		}
		codes = append(codes, n)
	}
	return codes, nil
}

// codesString renders the effective free codes for a site.
func codesString(codes []int) string {
	if len(codes) == 0 {
		return strconv.Itoa(defaultFreeCode)
	}
	parts := make([]string, len(codes))
	for i, c := range codes {
		parts[i] = strconv.Itoa(c)
	}
	return strings.Join(parts, ",")
}

// validateTemplate checks that a template is an absolute http or https URL containing {}.
func validateTemplate(t string) error {
	if !strings.Contains(t, "{}") {
		return fmt.Errorf("template %q must contain {} where the username goes", t)
	}
	u, err := url.Parse(strings.ReplaceAll(t, "{}", "x"))
	if err != nil {
		return fmt.Errorf("template %q is not a valid URL: %v", t, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("template %q must be an absolute http or https URL", t)
	}
	return nil
}

// configDir returns the user's XDG config directory, honoring XDG_CONFIG_HOME
// when set to an absolute path and falling back to $HOME/.config otherwise.
func configDir() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" && filepath.IsAbs(v) {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

// sitesPath returns the path of the sites file.
func sitesPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, programName, "sites"), nil
}

// parseSites reads NAME TEMPLATE [CODES] lines from r, naming path in errors.
func parseSites(r io.Reader, path string) ([]site, error) {
	var sites []site
	scanner := bufio.NewScanner(r)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) < 2 || len(fields) > 3 {
			return nil, fmt.Errorf("%s line %d: expected NAME TEMPLATE [CODES], got %d field(s)", path, line, len(fields))
		}
		s := site{name: fields[0], template: fields[1], source: sourceFile}
		if err := validateTemplate(s.template); err != nil {
			return nil, fmt.Errorf("%s line %d: %v", path, line, err)
		}
		if len(fields) == 3 {
			codes, err := parseCodes(fields[2])
			if err != nil {
				return nil, fmt.Errorf("%s line %d: %v", path, line, err)
			}
			s.free = codes
		}
		sites = append(sites, s)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	return sites, nil
}

// loadSites merges the sites file over the built-ins, sorted by name. A missing file yields the built-ins.
func loadSites(path string) ([]site, error) {
	byName := map[string]site{}
	for _, s := range builtinSites {
		byName[s.name] = s
	}
	f, err := os.Open(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		defer f.Close()
		fromFile, perr := parseSites(f, path)
		if perr != nil {
			return nil, perr
		}
		for _, s := range fromFile {
			byName[s.name] = s
		}
	}
	sites := make([]site, 0, len(byName))
	for _, s := range byName {
		sites = append(sites, s)
	}
	sort.Slice(sites, func(i, j int) bool { return sites[i].name < sites[j].name })
	return sites, nil
}

// resolveSite returns the site for a name or a validated URL template argument.
func resolveSite(arg string, sites []site) (site, error) {
	if strings.Contains(arg, "{}") {
		if err := validateTemplate(arg); err != nil {
			return site{}, err
		}
		return site{name: arg, template: arg, source: sourceArgument}, nil
	}
	names := make([]string, 0, len(sites))
	for _, s := range sites {
		if s.name == arg {
			return s, nil
		}
		names = append(names, s.name)
	}
	hint := ""
	if strings.Contains(arg, "/") {
		hint = "; a template needs {} where the username goes"
	}
	return site{}, fmt.Errorf("unknown site %q%s; known sites: %s", arg, hint, strings.Join(names, ", "))
}

// uniqueRunes drops repeated runes, keeping first-seen order.
func uniqueRunes(rs []rune) []rune {
	seen := map[rune]bool{}
	out := make([]rune, 0, len(rs))
	for _, r := range rs {
		if !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	return out
}

// expandPattern expands every [chars] class in p into all candidate names.
func expandPattern(p string) ([]string, error) {
	var classes [][]rune
	runes := []rune(p)
	for i := 0; i < len(runes); {
		switch runes[i] {
		case '[':
			j := i + 1
			for j < len(runes) && runes[j] != ']' {
				j++
			}
			if j == len(runes) {
				return nil, fmt.Errorf("pattern %q has an unclosed [ class", p)
			}
			if j == i+1 {
				return nil, fmt.Errorf("pattern %q has an empty [] class", p)
			}
			classes = append(classes, uniqueRunes(runes[i+1:j]))
			i = j + 1
		case ']':
			return nil, fmt.Errorf("pattern %q has a ] without a matching [", p)
		default:
			classes = append(classes, []rune{runes[i]})
			i++
		}
	}
	names := []string{""}
	for _, class := range classes {
		next := make([]string, 0, len(names)*len(class))
		for _, prefix := range names {
			for _, r := range class {
				next = append(next, prefix+string(r))
			}
		}
		names = next
	}
	return names, nil
}

// candidates returns the names for the CANDIDATE argument and whether it was one literal name.
func candidates(arg string, stdin io.Reader) ([]string, bool, error) {
	switch {
	case arg == "-":
		var names []string
		scanner := bufio.NewScanner(stdin)
		for scanner.Scan() {
			if name := strings.TrimSpace(scanner.Text()); name != "" {
				names = append(names, name)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, false, fmt.Errorf("reading stdin: %v", err)
		}
		return names, false, nil
	case strings.Contains(arg, "["):
		names, err := expandPattern(arg)
		return names, false, err
	default:
		return []string{arg}, true, nil
	}
}

// keep reports whether name passes the -d and -r filters.
func keep(name string, distinct bool, require string) bool {
	if distinct {
		seen := map[rune]bool{}
		for _, r := range name {
			if seen[r] {
				return false
			}
			seen[r] = true
		}
	}
	for _, r := range require {
		if !strings.ContainsRune(name, r) {
			return false
		}
	}
	return true
}

// filter applies the -d and -r filters to names.
func filter(names []string, distinct bool, require string) []string {
	if !distinct && require == "" {
		return names
	}
	out := make([]string, 0, len(names))
	for _, n := range names {
		if keep(n, distinct, require) {
			out = append(out, n)
		}
	}
	return out
}

// outcome is the answer for one username.
type outcome int

const (
	outcomeFree outcome = iota
	outcomeTaken
	outcomeUnknown
)

// result is the outcome of one username check plus the reason when unknown.
type result struct {
	outcome outcome
	reason  string
}

// describe renders a result the way the output lines show it.
func (r result) describe() string {
	switch r.outcome {
	case outcomeFree:
		return "free"
	case outcomeTaken:
		return "taken"
	default:
		return "unknown (" + r.reason + ")"
	}
}

// checker looks usernames up against one profile-URL template.
type checker struct {
	client   *http.Client
	template string
	free     map[int]bool
	sleep    func(time.Duration)
}

// newChecker builds a checker for template with the given free status codes.
func newChecker(client *http.Client, template string, free []int, sleep func(time.Duration)) *checker {
	c := &checker{client: client, template: template, free: map[int]bool{}, sleep: sleep}
	for _, code := range free {
		c.free[code] = true
	}
	return c
}

// status returns the final status code for target, sending HEAD and falling back to GET once on 405.
func (c *checker) status(target string) (int, error) {
	resp, err := c.client.Head(target)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		return resp.StatusCode, nil
	}
	resp, err = c.client.Get(target)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode, nil
}

// check looks one username up, retrying rate limits and transport errors with doubling backoff.
func (c *checker) check(name string) result {
	target := strings.ReplaceAll(c.template, "{}", url.PathEscape(name))
	backoff := initialBackoff
	var last string
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			c.sleep(backoff)
			backoff *= 2
		}
		status, err := c.status(target)
		if err != nil {
			last = err.Error()
			continue
		}
		switch {
		case c.free[status]:
			return result{outcome: outcomeFree}
		case status == http.StatusTooManyRequests:
			last = "HTTP 429"
			continue
		case status >= 200 && status < 300:
			return result{outcome: outcomeTaken}
		default:
			return result{outcome: outcomeUnknown, reason: fmt.Sprintf("HTTP %d", status)}
		}
	}
	return result{outcome: outcomeUnknown, reason: fmt.Sprintf("%s after %d attempts", last, maxAttempts)}
}

// fail prints err to stderr with the program name and returns exit code 2.
func fail(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "%s: %v\n", programName, err)
	return 2
}

// run executes the utility with the given arguments and returns its exit code.
func run(args []string, e env) int {
	o, act, err := parseArgs(args)
	if err != nil {
		return fail(e.stderr, err)
	}
	switch act {
	case actionHelp:
		fmt.Fprint(e.stdout, usage())
		return 0
	case actionVersion:
		fmt.Fprintf(e.stdout, "%s v%s\n", programName, programVersion)
		return 0
	}
	sites, err := loadSites(e.sitesPath)
	if err != nil {
		return fail(e.stderr, err)
	}
	if act == actionList {
		for _, s := range sites {
			fmt.Fprintf(e.stdout, "%-12s %-44s %-10s %s\n", s.name, s.template, codesString(s.free), s.source)
		}
		return 0
	}
	target, err := resolveSite(o.site, sites)
	if err != nil {
		return fail(e.stderr, err)
	}
	names, single, err := candidates(o.candidate, e.stdin)
	if err != nil {
		return fail(e.stderr, err)
	}
	names = filter(names, o.distinct, o.require)
	if single && len(names) == 0 {
		return fail(e.stderr, fmt.Errorf("%s is excluded by -d or -r", o.candidate))
	}
	if o.dryRun {
		for _, n := range names {
			fmt.Fprintln(e.stdout, n)
		}
		return 0
	}
	free := o.free
	if free == nil {
		free = target.free
	}
	if free == nil {
		free = []int{defaultFreeCode}
	}
	c := newChecker(e.client, target.template, free, e.sleep)
	if single {
		r := c.check(names[0])
		fmt.Fprintf(e.stdout, "%s: %s\n", names[0], r.describe())
		switch r.outcome {
		case outcomeFree:
			return 0
		case outcomeTaken:
			return 1
		default:
			return 2
		}
	}
	var nFree, nTaken, nUnknown int
	for i, n := range names {
		if i > 0 && o.wait > 0 {
			e.sleep(o.wait)
		}
		r := c.check(n)
		switch r.outcome {
		case outcomeFree:
			nFree++
			fmt.Fprintln(e.stdout, n)
		case outcomeTaken:
			nTaken++
		default:
			nUnknown++
			fmt.Fprintf(e.stderr, "%s: %s\n", n, r.describe())
		}
	}
	fmt.Fprintf(e.stderr, "Checked %d: %d free, %d taken, %d unknown.\n", len(names), nFree, nTaken, nUnknown)
	if nUnknown > 0 {
		return 2
	}
	return 0
}

func main() {
	path, err := sitesPath()
	if err != nil {
		os.Exit(fail(os.Stderr, err))
	}
	os.Exit(run(os.Args[1:], env{
		stdin:     os.Stdin,
		stdout:    os.Stdout,
		stderr:    os.Stderr,
		sitesPath: path,
		client:    &http.Client{Timeout: requestTimeout},
		sleep:     time.Sleep,
	}))
}
