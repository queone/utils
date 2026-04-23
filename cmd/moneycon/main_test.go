package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommatize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"100", "100"},
		{"999", "999"},
		{"999.99", "999.99"},
		{"1000", "1,000"},
		{"1234", "1,234"},
		{"1234567", "1,234,567"},
		{"1234.56", "1,234.56"},
		{"12345.678", "12,345.678"},
		{"-1500", "-1,500"},
		{"-1500.25", "-1,500.25"},
		{"0", "0"},
		{"abc", "abc"},
		{"", ""},
	}
	for _, tt := range tests {
		got := commatize(tt.in)
		if got != tt.want {
			t.Errorf("commatize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"1234", 1234.0},
		{"1,234", 1234.0},
		{"1,234.56", 1234.56},
		{"abc", 0.0},
		{"", 0.0},
	}
	for _, tt := range tests {
		got := toFloat(tt.in)
		if got != tt.want {
			t.Errorf("toFloat(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestIsMoneyconOutput(t *testing.T) {
	dir := t.TempDir()

	mco := filepath.Join(dir, "output.txt")
	os.WriteFile(mco, []byte("DESCRIPTION  MO/AVG  YR/AVG  NOTES\nRent  1,200.00  14,400.00\n"), 0644)
	got, err := isMoneyconOutput(mco)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("expected moneycon output format to be detected")
	}

	csvf := filepath.Join(dir, "input.csv")
	os.WriteFile(csvf, []byte("TYPE,DESCRIPTION,MO/AVG,YR/AVG,NOTES\nIncome,Salary,5000,60000,\n"), 0644)
	got, err = isMoneyconOutput(csvf)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Error("expected CSV to not be detected as moneycon output")
	}

	aligned := filepath.Join(dir, "aligned.txt")
	os.WriteFile(aligned, []byte("DESCRIPTION  MO/AVG  YR/AVG  NOTES\nIncome - Salary  5000  60000\n"), 0644)
	got, err = isMoneyconOutput(aligned)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("expected 4-column aligned with DESCRIPTION header to be detected as moneycon output")
	}
}

func runInDir(t *testing.T, dir string, args ...string) error {
	t.Helper()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	os.Args = append([]string{"moneycon"}, args...)
	return run()
}

func TestCSVInput(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")

	csv := "TYPE,DESCRIPTION,MO/AVG,YR/AVG,NOTES\n" +
		"Income,Salary,5000,60000,primary\n" +
		"Income,Freelance,1500,18000,\n"
	os.WriteFile(in, []byte(csv), 0644)

	if err := runInDir(t, dir, in); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, outputFile))
	result := string(data)

	if !strings.Contains(result, "Income - Salary") {
		t.Error("expected TYPE merged into DESCRIPTION as 'Income - Salary'")
	}
	if !strings.Contains(result, "Income - Freelance") {
		t.Error("expected TYPE merged into DESCRIPTION as 'Income - Freelance'")
	}
	if !strings.Contains(result, "TOTAL") {
		t.Error("expected TOTAL row")
	}
	if !strings.Contains(result, "6,500.00") {
		t.Errorf("expected MO total 6,500.00 in output:\n%s", result)
	}
	if !strings.Contains(result, "78,000.00") {
		t.Errorf("expected YR total 78,000.00 in output:\n%s", result)
	}
}

func TestAlignedInput(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.txt")

	aligned := "DESCRIPTION  TYPE  MO/AVG  YR/AVG  NOTES\n" +
		"Income - Salary  Inc  5000  60000  primary\n" +
		"Income - Freelance  Inc  1500  18000\n"
	os.WriteFile(in, []byte(aligned), 0644)

	if err := runInDir(t, dir, in); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, outputFile))
	result := string(data)

	if !strings.Contains(result, "Income - Salary") {
		t.Error("expected TYPE extracted and merged back as 'Income - Salary'")
	}
	if !strings.Contains(result, "6,500.00") {
		t.Errorf("expected MO total 6,500.00 in output:\n%s", result)
	}
}

func TestSkipTotalAndHeaderRows(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")

	csv := "TYPE,DESCRIPTION,MO/AVG,YR/AVG,NOTES\n" +
		"TYPE,DESCRIPTION,MO/AVG,YR/AVG,NOTES\n" +
		"Income,Salary,5000,60000,\n" +
		"Total,All Income,5000,60000,\n" +
		",Grand total,5000,60000,\n"
	os.WriteFile(in, []byte(csv), 0644)

	if err := runInDir(t, dir, in); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, outputFile))
	result := string(data)

	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (header + 1 data + TOTAL), got %d:\n%s", len(lines), result)
	}
	if !strings.Contains(result, "5,000.00") {
		t.Errorf("expected MO total 5,000.00 (only Salary counted):\n%s", result)
	}
}

func TestEmptyFields(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")

	csv := "TYPE,DESCRIPTION,MO/AVG,YR/AVG,NOTES\n" +
		",Rent,1200,14400,\n" +
		",Groceries,,3600,\n"
	os.WriteFile(in, []byte(csv), 0644)

	if err := runInDir(t, dir, in); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, outputFile))
	result := string(data)

	if !strings.Contains(result, "Rent") {
		t.Error("expected Rent row without TYPE prefix")
	}
	if !strings.Contains(result, "1,200.00") {
		t.Errorf("expected MO total 1,200.00:\n%s", result)
	}
	if !strings.Contains(result, "18,000.00") {
		t.Errorf("expected YR total 18,000.00:\n%s", result)
	}
}

func TestPrecommatizedInput(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")

	csv := "TYPE,DESCRIPTION,MO/AVG,YR/AVG,NOTES\n" +
		"Income,Salary,\"5,000\",\"60,000\",\n"
	os.WriteFile(in, []byte(csv), 0644)

	if err := runInDir(t, dir, in); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, outputFile))
	result := string(data)

	if !strings.Contains(result, "5,000.00") {
		t.Errorf("expected data row MO 5,000.00:\n%s", result)
	}
	if !strings.Contains(result, "60,000.00") {
		t.Errorf("expected data row YR 60,000.00:\n%s", result)
	}
}

func TestOutputAlignment(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")

	csv := "TYPE,DESCRIPTION,MO/AVG,YR/AVG,NOTES\n" +
		"Income,Salary,5000,60000,primary\n" +
		",Rent,1200,14400,monthly\n"
	os.WriteFile(in, []byte(csv), 0644)

	if err := runInDir(t, dir, in); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, outputFile))
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}

	for i, line := range lines {
		if line != strings.TrimRight(line, " ") {
			t.Errorf("line %d has trailing spaces: %q", i, line)
		}
	}
}

func TestAllNumericTwoDecimalPlaces(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")

	csv := "TYPE,DESCRIPTION,MO/AVG,YR/AVG,NOTES\n" +
		",Item,100,1200,\n"
	os.WriteFile(in, []byte(csv), 0644)

	if err := runInDir(t, dir, in); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(dir, outputFile))
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")

	dataLine := lines[1]
	if !strings.Contains(dataLine, "100.00") {
		t.Errorf("expected data row MO with 2 decimal places (100.00), got: %s", dataLine)
	}
	if !strings.Contains(dataLine, "1,200.00") {
		t.Errorf("expected data row YR with 2 decimal places (1,200.00), got: %s", dataLine)
	}

	totalLine := lines[len(lines)-1]
	if !strings.Contains(totalLine, "100.00") {
		t.Errorf("expected TOTAL MO with 2 decimal places (100.00), got: %s", totalLine)
	}
	if !strings.Contains(totalLine, "1,200.00") {
		t.Errorf("expected TOTAL YR with 2 decimal places (1,200.00), got: %s", totalLine)
	}
}

func TestOutputFileAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.csv")
	existing := filepath.Join(dir, outputFile)

	csv := "TYPE,DESCRIPTION,MO/AVG,YR/AVG,NOTES\n,Item,100,1200,\n"
	os.WriteFile(in, []byte(csv), 0644)
	os.WriteFile(existing, []byte("existing"), 0644)

	err := runInDir(t, dir, in)
	if err == nil {
		t.Fatal("expected error when moneycon-output.txt already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestRetallyMode(t *testing.T) {
	dir := t.TempDir()
	budget := filepath.Join(dir, "budget.txt")

	content := "DESCRIPTION  MO/AVG  YR/AVG  NOTES\n" +
		"Income - Salary  5000  60000  primary\n" +
		"Rent  1200  14400  monthly\n" +
		"Groceries  600  7200\n" +
		"TOTAL  6800  81600\n"
	os.WriteFile(budget, []byte(content), 0644)

	if err := runInDir(t, dir, budget); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(budget)
	result := string(data)

	if !strings.Contains(result, "5,000.00") {
		t.Errorf("expected normalized Salary MO 5,000.00:\n%s", result)
	}
	if !strings.Contains(result, "60,000.00") {
		t.Errorf("expected normalized Salary YR 60,000.00:\n%s", result)
	}
	if !strings.Contains(result, "1,200.00") {
		t.Errorf("expected normalized Rent MO 1,200.00:\n%s", result)
	}
	if !strings.Contains(result, "600.00") {
		t.Errorf("expected normalized Groceries MO 600.00:\n%s", result)
	}

	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	totalLine := lines[len(lines)-1]
	if !strings.Contains(totalLine, "TOTAL") {
		t.Errorf("expected TOTAL row at end:\n%s", result)
	}
	if !strings.Contains(totalLine, "6,800.00") {
		t.Errorf("expected recomputed MO total 6,800.00:\n%s", result)
	}
	if !strings.Contains(totalLine, "81,600.00") {
		t.Errorf("expected recomputed YR total 81,600.00:\n%s", result)
	}
}

func TestRetallyDropsOldTotal(t *testing.T) {
	dir := t.TempDir()
	budget := filepath.Join(dir, "budget.txt")

	content := "DESCRIPTION  MO/AVG  YR/AVG  NOTES\n" +
		"Rent  1200  14400  monthly\n" +
		"TOTAL  9999  99999\n"
	os.WriteFile(budget, []byte(content), 0644)

	if err := runInDir(t, dir, budget); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(budget)
	result := string(data)
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")

	totalCount := 0
	for _, line := range lines {
		if strings.Contains(line, "TOTAL") {
			totalCount++
		}
	}
	if totalCount != 1 {
		t.Errorf("expected exactly 1 TOTAL row, got %d:\n%s", totalCount, result)
	}

	totalLine := lines[len(lines)-1]
	if !strings.Contains(totalLine, "1,200.00") {
		t.Errorf("expected recomputed MO total 1,200.00 (not old 9999):\n%s", result)
	}
	if !strings.Contains(totalLine, "14,400.00") {
		t.Errorf("expected recomputed YR total 14,400.00 (not old 99999):\n%s", result)
	}
}

func TestRetallySloppyEntries(t *testing.T) {
	dir := t.TempDir()
	budget := filepath.Join(dir, "budget.txt")

	content := "DESCRIPTION  MO/AVG  YR/AVG  NOTES\n" +
		"Rent  1200  14400  monthly\n" +
		"Groceries  600.5  7206\n" +
		"Internet  89  1068\n"
	os.WriteFile(budget, []byte(content), 0644)

	if err := runInDir(t, dir, budget); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(budget)
	result := string(data)

	if !strings.Contains(result, "1,200.00") {
		t.Errorf("expected Rent MO normalized to 1,200.00:\n%s", result)
	}
	if !strings.Contains(result, "600.50") {
		t.Errorf("expected Groceries MO normalized to 600.50:\n%s", result)
	}
	if !strings.Contains(result, "89.00") {
		t.Errorf("expected Internet MO normalized to 89.00:\n%s", result)
	}
	if !strings.Contains(result, "1,889.50") {
		t.Errorf("expected recomputed MO total 1,889.50:\n%s", result)
	}
	if !strings.Contains(result, "22,674.00") {
		t.Errorf("expected recomputed YR total 22,674.00:\n%s", result)
	}
}
