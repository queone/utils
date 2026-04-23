package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	programName    = "moneycon"
	programVersion = "1.0.0"
	outputFile     = "moneycon-output.txt"
)

var outHeader = [4]string{"DESCRIPTION", "MO/AVG", "YR/AVG", "NOTES"}

type row struct {
	typ  string
	desc string
	mo   string
	yr   string
	note string
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: %s FILE\n", programName)
	fmt.Fprintf(os.Stderr, "       %s -v | --version\n", programName)
	os.Exit(1)
}

func stripBOM(s string) string {
	return strings.TrimPrefix(s, "\xef\xbb\xbf")
}

func commatize(s string) string {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	if math.Abs(n) < 1000 {
		return s
	}
	if strings.Contains(s, ".") {
		parts := strings.SplitN(s, ".", 2)
		decimals := len(parts[1])
		formatted := strconv.FormatFloat(n, 'f', decimals, 64)
		return addCommas(formatted)
	}
	return addCommas(strconv.FormatInt(int64(n), 10))
}

func addCommas(s string) string {
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	intPart := s
	decPart := ""
	if idx := strings.Index(s, "."); idx >= 0 {
		intPart = s[:idx]
		decPart = s[idx:]
	}

	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}

	result := b.String() + decPart
	if negative {
		result = "-" + result
	}
	return result
}

func normalize2(s string) string {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return fmt.Sprintf("%.2f", n)
}

func toFloat(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.0
	}
	return n
}

var reQuoted = regexp.MustCompile(`"[^"]*"`)
var reTwoSpaces = regexp.MustCompile(` {2,}`)

func isMoneyconOutput(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	content := stripBOM(string(data))
	for _, ln := range strings.SplitN(content, "\n", 2) {
		line := strings.TrimSpace(ln)
		if line == "" {
			continue
		}
		cols := reTwoSpaces.Split(line, -1)
		if len(cols) >= 3 && cols[0] == "DESCRIPTION" && cols[1] == "MO/AVG" && cols[2] == "YR/AVG" {
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

func detectInputFormat(firstLine string) string {
	stripped := reQuoted.ReplaceAllString(firstLine, "")
	if strings.Contains(stripped, ",") {
		return "csv"
	}
	if reTwoSpaces.MatchString(firstLine) {
		return "aligned"
	}
	return "csv"
}

func readCSV(path string) ([]row, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	content := stripBOM(string(data))

	r := csv.NewReader(strings.NewReader(content))
	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV headers from %s: %w", path, err)
	}
	for i := range headers {
		headers[i] = strings.ToUpper(strings.TrimSpace(headers[i]))
	}

	colIdx := map[string]int{}
	for i, h := range headers {
		colIdx[h] = i
	}

	var rows []row
	for {
		record, err := r.Read()
		if err != nil {
			break
		}
		get := func(name string) string {
			if idx, ok := colIdx[name]; ok && idx < len(record) {
				return strings.TrimSpace(record[idx])
			}
			return ""
		}
		rows = append(rows, row{
			typ:  get("TYPE"),
			desc: get("DESCRIPTION"),
			mo:   get("MO/AVG"),
			yr:   get("YR/AVG"),
			note: get("NOTES"),
		})
	}
	return rows, nil
}

func splitAligned(line string, ncols int) []string {
	parts := reTwoSpaces.Split(strings.TrimSpace(line), -1)
	for len(parts) < ncols {
		parts = append(parts, "")
	}
	if len(parts) > ncols {
		tail := strings.Join(parts[ncols-1:], "  ")
		parts = append(parts[:ncols-1], tail)
	}
	return parts
}

func readAligned5(path string) ([]row, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	content := stripBOM(string(data))

	var lines []string
	for ln := range strings.SplitSeq(content, "\n") {
		if strings.TrimSpace(ln) != "" {
			lines = append(lines, ln)
		}
	}
	if len(lines) == 0 {
		return nil, nil
	}

	headers := reTwoSpaces.Split(strings.TrimSpace(lines[0]), -1)
	ncols := len(headers)
	colIdx := map[string]int{}
	for i, h := range headers {
		colIdx[h] = i
	}

	var rows []row
	for _, line := range lines[1:] {
		values := splitAligned(line, ncols)
		get := func(name string) string {
			if idx, ok := colIdx[name]; ok && idx < len(values) {
				return values[idx]
			}
			return ""
		}

		desc := get("DESCRIPTION")
		var typ string
		if idx := strings.Index(desc, " - "); idx >= 0 {
			typ = desc[:idx]
			desc = desc[idx+3:]
		}

		rows = append(rows, row{
			typ:  typ,
			desc: desc,
			mo:   get("MO/AVG"),
			yr:   get("YR/AVG"),
			note: get("NOTES"),
		})
	}
	return rows, nil
}

func readMoneyconOutput(path string) ([]row, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	content := stripBOM(string(data))

	var lines []string
	for ln := range strings.SplitSeq(content, "\n") {
		if strings.TrimSpace(ln) != "" {
			lines = append(lines, ln)
		}
	}
	if len(lines) < 2 {
		return nil, nil
	}

	headers := reTwoSpaces.Split(strings.TrimSpace(lines[0]), -1)
	ncols := len(headers)
	colIdx := map[string]int{}
	for i, h := range headers {
		colIdx[h] = i
	}

	var rows []row
	for _, line := range lines[1:] {
		values := splitAligned(line, ncols)
		get := func(name string) string {
			if idx, ok := colIdx[name]; ok && idx < len(values) {
				return values[idx]
			}
			return ""
		}

		desc := get("DESCRIPTION")
		if strings.ToLower(desc) == "total" || strings.Contains(strings.ToLower(desc), "total") {
			continue
		}

		rows = append(rows, row{
			desc: desc,
			mo:   get("MO/AVG"),
			yr:   get("YR/AVG"),
			note: get("NOTES"),
		})
	}
	return rows, nil
}

type outputRow [4]string

func process(input []row) []outputRow {
	var out []outputRow
	var moTotal, yrTotal float64

	for _, r := range input {
		if strings.Contains(strings.ToLower(r.typ), "total") ||
			strings.Contains(strings.ToLower(r.desc), "total") {
			continue
		}
		if strings.ToLower(r.typ) == "type" && strings.ToLower(r.desc) == "description" {
			continue
		}

		mo := strings.ReplaceAll(r.mo, ",", "")
		yr := strings.ReplaceAll(r.yr, ",", "")
		moTotal += toFloat(mo)
		yrTotal += toFloat(yr)

		desc := r.desc
		if r.typ != "" {
			desc = r.typ + " - " + r.desc
		}
		out = append(out, outputRow{desc, commatize(normalize2(mo)), commatize(normalize2(yr)), r.note})
	}

	out = append(out, outputRow{
		"TOTAL",
		commatize(fmt.Sprintf("%.2f", moTotal)),
		commatize(fmt.Sprintf("%.2f", yrTotal)),
		"",
	})
	return out
}

func processRetally(input []row) []outputRow {
	var out []outputRow
	var moTotal, yrTotal float64

	for _, r := range input {
		mo := strings.ReplaceAll(r.mo, ",", "")
		yr := strings.ReplaceAll(r.yr, ",", "")
		moTotal += toFloat(mo)
		yrTotal += toFloat(yr)

		out = append(out, outputRow{r.desc, commatize(normalize2(mo)), commatize(normalize2(yr)), r.note})
	}

	out = append(out, outputRow{
		"TOTAL",
		commatize(fmt.Sprintf("%.2f", moTotal)),
		commatize(fmt.Sprintf("%.2f", yrTotal)),
		"",
	})
	return out
}

func formatOutput(rows []outputRow) string {
	widths := [4]int{}
	for i, h := range outHeader {
		if len(h) > widths[i] {
			widths[i] = len(h)
		}
	}
	for _, r := range rows {
		for i, v := range r {
			if len(v) > widths[i] {
				widths[i] = len(v)
			}
		}
	}

	rightAlign := [4]bool{false, true, true, false}

	pad := func(s string, w int, right bool) string {
		if right {
			return fmt.Sprintf("%*s", w, s)
		}
		return fmt.Sprintf("%-*s", w, s)
	}

	emit := func(vals [4]string) string {
		parts := make([]string, 4)
		for i, v := range vals {
			parts[i] = pad(v, widths[i], rightAlign[i])
		}
		return strings.TrimRight(strings.Join(parts, "  "), " ")
	}

	var b strings.Builder
	b.WriteString(emit(outHeader))
	b.WriteByte('\n')
	for _, r := range rows {
		b.WriteString(emit(r))
		b.WriteByte('\n')
	}
	return b.String()
}

func run() error {
	args := os.Args[1:]

	if len(args) == 1 && (args[0] == "-v" || args[0] == "--version") {
		fmt.Printf("%s v%s\n", programName, programVersion)
		return nil
	}

	if len(args) != 1 {
		printUsage()
	}

	inPath := args[0]

	retally, err := isMoneyconOutput(inPath)
	if err != nil {
		return err
	}

	if retally {
		input, err := readMoneyconOutput(inPath)
		if err != nil {
			return err
		}
		out := processRetally(input)
		result := formatOutput(out)
		if err := os.WriteFile(inPath, []byte(result), 0644); err != nil {
			return fmt.Errorf("write %s: %w", inPath, err)
		}
		return nil
	}

	if _, err := os.Stat(outputFile); err == nil {
		return fmt.Errorf("%s already exists; remove or rename it first", outputFile)
	}

	data, err := os.ReadFile(inPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inPath, err)
	}
	firstLine := ""
	for _, ln := range strings.SplitN(stripBOM(string(data)), "\n", 2) {
		if strings.TrimSpace(ln) != "" {
			firstLine = stripBOM(ln)
			break
		}
	}

	format := detectInputFormat(firstLine)

	var input []row
	if format == "csv" {
		input, err = readCSV(inPath)
	} else {
		input, err = readAligned5(inPath)
	}
	if err != nil {
		return err
	}

	out := process(input)
	result := formatOutput(out)

	if err := os.WriteFile(outputFile, []byte(result), 0644); err != nil {
		return fmt.Errorf("write %s: %w", outputFile, err)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", programName, err)
		os.Exit(1)
	}
}
