// Package buildtool implements the build/test pipeline for utils.
// It is invoked via `go run ./cmd/build` (or the `./build.sh` wrapper).
package buildtool

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/queone/utils/internal/color"
)

// Config holds parsed build arguments.
type Config struct {
	Verbose bool
	Targets []string
}

// CmdRunner abstracts command execution so the pipeline can be tested with
// fake commands.
type CmdRunner interface {
	// Streaming runs a command with stdout/stderr connected to the given writers.
	Streaming(out, errOut io.Writer, name string, args ...string) error
	// CapturedSoft runs a command and returns its combined output. On failure
	// with empty output, returns the error string.
	CapturedSoft(name string, args ...string) string
	// CapturedCheck runs a command and returns (output, failed).
	CapturedCheck(name string, args ...string) (string, bool)
	// Captured runs a command and returns its combined output or an error.
	Captured(name string, args ...string) (string, error)
}

// Pipeline holds the injectable dependencies for a build run.
type Pipeline struct {
	Cmd CmdRunner
}

// scriptOnlyCommands lists cmd/ entrypoints that are run via `go run` and
// should not be installed as binaries.
var scriptOnlyCommands = map[string]struct{}{
	"build": {},
	"rel":   {},
}

var versionPattern = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

type semver struct {
	major int
	minor int
	patch int
}

// ParseArgs parses build command-line arguments.
func ParseArgs(args []string) (Config, bool, error) {
	if len(args) == 1 && isHelpArg(args[0]) {
		return Config{}, true, nil
	}
	cfg := Config{}
	for _, arg := range args {
		switch arg {
		case "-v", "--verbose":
			cfg.Verbose = true
		case "-h", "-?", "--help":
			return Config{}, false, errors.New("help flags must be used by themselves")
		default:
			if strings.HasPrefix(arg, "-") {
				return Config{}, false, fmt.Errorf("unsupported option %q; use target names plus optional -v, --verbose", arg)
			}
			cfg.Targets = append(cfg.Targets, arg)
		}
	}
	return cfg, false, nil
}

// Usage returns the help text for the build command.
func Usage() string {
	return color.FormatUsage("build [target ...] [-v|--verbose]", []color.UsageLine{
		{Flag: "-v, --verbose", Desc: "run go test in verbose mode"},
		{Flag: "-h, -?, --help", Desc: "show this help"},
	}, "When targets are specified, validation (vet, fmt, test, staticcheck) runs\nonly against those cmd packages. To validate the full repo, run with no targets.")
}

// Run executes the full build pipeline using real command execution.
func Run(cfg Config, out io.Writer, errOut io.Writer) error {
	return (&Pipeline{Cmd: &execRunner{}}).Run(cfg, out, errOut)
}

// Run executes the full build pipeline using the pipeline's command runner.
func (p *Pipeline) Run(cfg Config, out io.Writer, errOut io.Writer) error {
	modulePath, err := p.Cmd.Captured("go", "list", "-m", "-f", "{{.Path}}")
	if err != nil {
		return err
	}
	modulePath = strings.TrimSpace(modulePath)

	gopathOut, err := p.Cmd.Captured("go", "env", "GOPATH")
	if err != nil {
		return err
	}
	gopath := strings.TrimSpace(gopathOut)
	if gopath == "" {
		return errors.New("go env GOPATH returned an empty value")
	}
	binDir := filepath.Join(gopath, "bin")
	ext := binaryExt()
	scopes := packageScopes(cfg.Targets)

	// go mod tidy
	fmt.Fprintln(out, color.Yel("==> Update go.mod to reflect actual dependencies"))
	if err := p.Cmd.Streaming(out, errOut, "go", "mod", "tidy"); err != nil {
		return err
	}

	// go fmt — fail-hard if files were reformatted
	fmt.Fprintln(out, "\n"+color.Yel("==> Format Go code according to standard rules"))
	if fmtOutput := p.Cmd.CapturedSoft("go", append([]string{"fmt"}, scopes...)...); strings.TrimSpace(fmtOutput) == "" {
		fmt.Fprintln(out, "    No formatting changes needed.")
	} else {
		writeIndented(out, fmtOutput)
		return fmt.Errorf("go fmt found files that need formatting")
	}

	// go fix — advisory only
	fmt.Fprintln(out, "\n"+color.Yel("==> Automatically fix code for API/language changes"))
	if fixOutput := p.Cmd.CapturedSoft("go", append([]string{"fix"}, scopes...)...); strings.TrimSpace(fixOutput) == "" {
		fmt.Fprintln(out, "    No fixes applied.")
	} else {
		writeIndented(out, fixOutput)
	}

	// go vet — fail-hard
	fmt.Fprintln(out, "\n"+color.Yel("==> Check code for potential issues"))
	if vetOutput, vetFailed := p.Cmd.CapturedCheck("go", append([]string{"vet"}, scopes...)...); vetFailed {
		writeIndented(out, vetOutput)
		return fmt.Errorf("go vet found issues")
	} else if trimmed := strings.TrimSpace(vetOutput); trimmed != "" {
		writeIndented(out, vetOutput)
	} else {
		fmt.Fprintln(out, "    No issues found by go vet.")
	}

	// go test with coverage
	coverFile, err := os.CreateTemp("", "utils-cover-*.out")
	if err != nil {
		return fmt.Errorf("create coverage file: %w", err)
	}
	coverPath := coverFile.Name()
	coverFile.Close()
	defer os.Remove(coverPath)

	fmt.Fprintln(out, "\n"+color.Yel("==> Run tests for all packages in the repository"))
	testArgs := []string{"test"}
	if cfg.Verbose {
		testArgs = append(testArgs, "-v")
	}
	testArgs = append(testArgs, "-coverprofile="+coverPath)
	testArgs = append(testArgs, scopes...)
	if err := p.Cmd.Streaming(out, errOut, "go", testArgs...); err != nil {
		return err
	}
	if err := p.printCoverageSummary(out, coverPath, modulePath); err != nil {
		return err
	}

	// staticcheck — always install @latest, fail-hard
	fmt.Fprintln(out, "\n"+color.Yel("==> Install static analysis tool for Go"))
	if err := p.Cmd.Streaming(out, errOut, "go", "install", "honnef.co/go/tools/cmd/staticcheck@latest"); err != nil {
		return err
	}

	fmt.Fprintln(out, "\n"+color.Yel("==> Analyze Go code for potential issues"))
	staticcheckPath, err := exec.LookPath("staticcheck")
	if err != nil {
		staticcheckPath = filepath.Join(binDir, "staticcheck"+binaryExt())
	}
	if scOutput, scFailed := p.Cmd.CapturedCheck(staticcheckPath, scopes...); scFailed {
		writeIndented(out, scOutput)
		return fmt.Errorf("staticcheck found issues")
	} else if trimmed := strings.TrimSpace(scOutput); trimmed != "" {
		writeIndented(out, scOutput)
	} else {
		fmt.Fprintln(out, "    No issues found by staticcheck.")
	}

	// Build binaries
	targets, err := buildTargets(cfg.Targets)
	if err != nil {
		return err
	}
	if len(cfg.Targets) == 0 {
		fmt.Fprintln(out, "\n"+color.Yel("==> Building all utilities"))
	} else {
		fmt.Fprintf(out, "\n%s %s\n", color.Yel("==> Building specific utilities:"), color.Grn(strings.Join(cfg.Targets, " ")))
	}
	builtCount := 0
	for _, target := range targets {
		version := extractProgramVersion(filepath.Join("cmd", target, "main.go"))
		outputPath := filepath.Join(binDir, target+ext)
		fmt.Fprintf(out, "\n%s %s\n", color.Yel("==> Building and installing"), color.Grn(target+" v"+version))
		if err := p.Cmd.Streaming(out, errOut, "go", "build", "-o", outputPath, "-ldflags", "-s -w", "./cmd/"+target); err != nil {
			return err
		}
		fmt.Fprintf(out, "    installed: %s\n", color.Cya(outputPath))
		builtCount++
	}
	if builtCount == 0 {
		fmt.Fprintln(out, "\n"+color.Yel("Warning: No utilities were built."))
		if len(cfg.Targets) > 0 {
			fmt.Fprintln(out, "    Check that the specified utilities exist in ./cmd/")
		}
	}

	// Version hint
	tagOutput, err := p.Cmd.Captured("git", "tag", "--list")
	if err != nil {
		return err
	}
	if nextTag, ok, err := NextPatchTagFromOutput(tagOutput); err != nil {
		return err
	} else if ok {
		fmt.Fprintf(out, "\n%s\n\n    ./build.sh %s %s\n\n", color.Yel("==> To release, run:"), color.Grn(nextTag), color.Gra("\"<release message>\""))
	}
	return nil
}

func (p *Pipeline) printCoverageSummary(out io.Writer, coverPath, modulePath string) error {
	output, err := p.Cmd.Captured("go", "tool", "cover", "-func="+coverPath)
	if err != nil {
		return err
	}
	var total string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "total:") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				total = fields[len(fields)-1]
			}
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan coverage output: %w", err)
	}
	if total == "" {
		return nil
	}

	domainPct, err := DomainCoverage(coverPath, modulePath+"/internal/")
	if err != nil {
		return err
	}
	coverageText := fmt.Sprintf("domain coverage: %.1f%%", domainPct)
	styledCoverage := CoverageColor(domainPct, coverageText)
	fmt.Fprintf(out, "    %s  %s\n", styledCoverage, color.Gra("(total: "+total+")"))
	return nil
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "-?" || arg == "--help"
}

func packageScopes(targets []string) []string {
	if len(targets) == 0 {
		return []string{"./..."}
	}
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		out = append(out, "./cmd/"+target)
	}
	return out
}

func buildTargets(targets []string) ([]string, error) {
	if len(targets) > 0 {
		return FilterInstallTargets(targets), nil
	}
	entries, err := os.ReadDir("cmd")
	if err != nil {
		return nil, fmt.Errorf("read ./cmd: %w", err)
	}
	var out []string
	for _, entry := range entries {
		if entry.IsDir() {
			out = append(out, entry.Name())
		}
	}
	return FilterInstallTargets(out), nil
}

// FilterInstallTargets removes script-only commands (build, rel) from a list
// of build targets. Only product binaries are installed.
func FilterInstallTargets(targets []string) []string {
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		if _, skip := scriptOnlyCommands[target]; skip {
			continue
		}
		out = append(out, target)
	}
	slices.Sort(out)
	return out
}

func extractProgramVersion(mainPath string) string {
	data, err := os.ReadFile(mainPath)
	if err != nil {
		return "unknown"
	}
	re := regexp.MustCompile(`programVersion\s*=\s*"([^"]*)"`)
	match := re.FindSubmatch(data)
	if len(match) < 2 {
		return "unknown"
	}
	return string(match[1])
}

func binaryExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// DomainCoverage computes coverage percentage for lines matching the given prefix.
func DomainCoverage(coverPath, prefix string) (float64, error) {
	content, err := os.ReadFile(coverPath)
	if err != nil {
		return 0, fmt.Errorf("read coverage profile: %w", err)
	}
	return DomainCoverageFromBytes(content, prefix)
}

// DomainCoverageFromBytes computes coverage from raw profile bytes.
func DomainCoverageFromBytes(content []byte, prefix string) (float64, error) {
	var totalStatements int
	var coveredStatements int
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "mode:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		if !strings.HasPrefix(fields[0], prefix) {
			continue
		}
		statements, err := strconv.Atoi(fields[1])
		if err != nil {
			return 0, fmt.Errorf("parse coverage statements from %q: %w", line, err)
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			return 0, fmt.Errorf("parse coverage count from %q: %w", line, err)
		}
		totalStatements += statements
		if count > 0 {
			coveredStatements += statements
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("scan coverage profile: %w", err)
	}
	if totalStatements == 0 {
		return 0, nil
	}
	return float64(coveredStatements) / float64(totalStatements) * 100, nil
}

// CoverageColor applies threshold-based coloring to coverage text.
func CoverageColor(pct float64, text string) string {
	switch {
	case pct >= 75:
		return color.Grn(text)
	case pct >= 50:
		return color.Yel(text)
	default:
		return color.Red(text)
	}
}

// NextPatchTagFromOutput computes the next patch tag from raw `git tag` output.
func NextPatchTagFromOutput(output string) (string, bool, error) {
	var versions []semver
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		match := versionPattern.FindStringSubmatch(line)
		if len(match) != 4 {
			continue
		}
		major, _ := strconv.Atoi(match[1])
		minor, _ := strconv.Atoi(match[2])
		patch, _ := strconv.Atoi(match[3])
		versions = append(versions, semver{major: major, minor: minor, patch: patch})
	}
	if err := scanner.Err(); err != nil {
		return "", false, fmt.Errorf("scan git tags: %w", err)
	}
	if len(versions) == 0 {
		return "", false, nil
	}
	slices.SortFunc(versions, func(a, b semver) int {
		if a.major != b.major {
			return a.major - b.major
		}
		if a.minor != b.minor {
			return a.minor - b.minor
		}
		return a.patch - b.patch
	})
	last := versions[len(versions)-1]
	return fmt.Sprintf("v%d.%d.%d", last.major, last.minor, last.patch+1), true, nil
}

// execRunner is the production CmdRunner using os/exec.
type execRunner struct{}

func (r *execRunner) Streaming(out, errOut io.Writer, name string, args ...string) error {
	command := strings.TrimSpace(name + " " + strings.Join(args, " "))
	fmt.Fprintf(out, "    %s\n", color.Grn(command))
	cmd := exec.Command(name, args...)
	cmd.Stdout = out
	cmd.Stderr = errOut
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s failed: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func (r *execRunner) CapturedSoft(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil && strings.TrimSpace(string(output)) == "" {
		return err.Error()
	}
	return string(output)
}

func (r *execRunner) CapturedCheck(name string, args ...string) (string, bool) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err != nil
}

func (r *execRunner) Captured(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s %s failed: %w\n%s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func writeIndented(out io.Writer, text string) {
	scanner := bufio.NewScanner(strings.NewReader(strings.TrimSpace(text)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "FAIL") {
			line = color.Red(line)
		}
		fmt.Fprintf(out, "    %s\n", line)
	}
}
