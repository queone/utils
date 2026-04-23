import csv
import re
import sys
from itertools import zip_longest


OUT_HEADER = ['DESCRIPTION', 'MO/AVG', 'YR/AVG', 'NOTES']


def commatize(s):
    try:
        n = float(s)
    except ValueError:
        return s
    if abs(n) < 1000:
        return s
    # Preserve decimal places from input
    if '.' in s:
        decimals = len(s.split('.')[1])
        return f"{n:,.{decimals}f}"
    return f"{int(n):,}"


def to_float(s):
    try:
        return float(s.replace(',', ''))
    except (ValueError, AttributeError):
        return 0.0


def detect_format(path):
    with open(path, encoding='utf-8-sig') as f:
        first = f.readline()
    if ',' in re.sub(r'"[^"]*"', '', first):
        return 'csv'
    if re.search(r' {2,}', first):
        return 'aligned'
    return 'csv'


def read_csv(path):
    with open(path, newline='', encoding='utf-8-sig') as f:
        r = csv.DictReader(f)
        r.fieldnames = [h.strip() for h in r.fieldnames]
        for row in r:
            row = {k: (v or '').strip() for k, v in row.items()}
            yield {
                'TYPE': row.get('TYPE', ''),
                'DESCRIPTION': row.get('DESCRIPTION', ''),
                'MO/AVG': row.get('MO/AVG', ''),
                'YR/AVG': row.get('YR/AVG', ''),
                'NOTES': row.get('NOTES', ''),
            }


def split_aligned(line, expected_cols):
    # Split on 2+ spaces. If fewer parts than expected (e.g. NOTES missing), pad.
    parts = re.split(r' {2,}', line.strip())
    while len(parts) < expected_cols:
        parts.append('')
    # If too many parts (shouldn't happen with 2+ space rule), merge trailing into last
    if len(parts) > expected_cols:
        parts = parts[:expected_cols - 1] + ['  '.join(parts[expected_cols - 1:])]
    return parts


def read_aligned(path):
    with open(path, encoding='utf-8-sig') as f:
        lines = [ln.rstrip('\n') for ln in f if ln.strip()]
    if not lines:
        return
    headers = re.split(r' {2,}', lines[0].strip())
    ncols = len(headers)

    for line in lines[1:]:
        values = split_aligned(line, ncols)
        row = dict(zip(headers, values))
        desc = row.get('DESCRIPTION', '')
        if ' - ' in desc:
            t, d = desc.split(' - ', 1)
        else:
            t, d = '', desc
        yield {
            'TYPE': t,
            'DESCRIPTION': d,
            'MO/AVG': row.get('MO/AVG', ''),
            'YR/AVG': row.get('YR/AVG', ''),
            'NOTES': row.get('NOTES', ''),
        }


def main():
    if len(sys.argv) != 3:
        print("Usage: python3 con.py <input> <output.txt>", file=sys.stderr)
        sys.exit(1)

    in_path, out_path = sys.argv[1], sys.argv[2]
    fmt_in = detect_format(in_path)
    reader = read_csv(in_path) if fmt_in == 'csv' else read_aligned(in_path)

    rows = []
    mo_total = 0.0
    yr_total = 0.0

    for row in reader:
        if 'total' in row['TYPE'].lower() or 'total' in row['DESCRIPTION'].lower():
            continue
        if row['TYPE'].lower() == 'type' and row['DESCRIPTION'].lower() == 'description':
            continue
        mo = row['MO/AVG'].replace(',', '')
        yr = row['YR/AVG'].replace(',', '')
        mo_total += to_float(mo)
        yr_total += to_float(yr)
        desc = f"{row['TYPE']} - {row['DESCRIPTION']}" if row['TYPE'] else row['DESCRIPTION']
        rows.append([desc, commatize(mo), commatize(yr), row['NOTES']])

    rows.append(['TOTAL', commatize(f"{mo_total:.2f}"), commatize(f"{yr_total:.2f}"), ''])

    widths = [max(len(x) for x in col) for col in zip_longest(OUT_HEADER, *rows, fillvalue='')]
    aligns = ['ljust', 'rjust', 'rjust', 'ljust']

    def emit(values):
        parts = [getattr(v, a)(w) for v, w, a in zip(values, widths, aligns)]
        return '  '.join(parts).rstrip()

    with open(out_path, 'w', encoding='utf-8') as f:
        f.write(emit(OUT_HEADER) + '\n')
        for row in rows:
            f.write(emit(row) + '\n')


if __name__ == '__main__':
    main()
