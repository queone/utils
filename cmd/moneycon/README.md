## moneycon
Financial data consolidation utility.

### Why?
Consolidating financial data from spreadsheets or text exports into a clean, aligned summary is a common task. This utility reads CSV or space-aligned tabular input and produces a consistent aligned text output with computed totals. It also re-tallies existing output files in-place after manual edits.

### Usage

```bash
moneycon FILE
```

The tool auto-detects the file format and operates in one of two modes:

**Consolidation mode** — FILE is a CSV input file with financial data. Output is written to `moneycon-output.txt` in the current directory (fails if it already exists).

CSV input columns: TYPE, DESCRIPTION, MO/AVG, YR/AVG, NOTES

```csv
TYPE,DESCRIPTION,MO/AVG,YR/AVG,NOTES
Income,Salary,5000,60000,primary
Income,Freelance,1500,18000,
,Rent,1200,14400,monthly
```

Output (`moneycon-output.txt`):

```
DESCRIPTION            MO/AVG      YR/AVG  NOTES
Income - Salary      5,000.00   60,000.00  primary
Income - Freelance   1,500.00   18,000.00
Rent                 1,200.00   14,400.00  monthly
TOTAL                7,700.00   92,400.00
```

**Re-tally mode** — FILE is already in moneycon output format (4-column aligned: DESCRIPTION, MO/AVG, YR/AVG, NOTES header). Normalizes sloppy entries, recomputes TOTAL, writes back in-place.

```bash
moneycon budget.txt
```

Rows containing "total" in TYPE or DESCRIPTION are skipped from input. All numeric values are normalized to 2 decimal places with thousand separators for values >= 1,000.

### Getting Started
This utility is part of a collection of Go utilities. To compile and install follow the **Getting Started** instructions at the [utils repo](https://github.com/queone/utils).
