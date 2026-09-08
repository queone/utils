// Package numfmt inserts thousands separators into numbers for display. It is
// the one implementation shared by every utility that prints large counts.
package numfmt

import (
	"strconv"
	"strings"
)

// Commas returns s with a comma between every three integer digits. s is a
// decimal number string such as "-1234567.89": the sign and the fraction are
// kept as they are, and a value with fewer than four integer digits comes back
// unchanged.
func Commas(s string) string {
	negative := strings.HasPrefix(s, "-")
	if negative {
		s = s[1:]
	}
	intPart, decPart := s, ""
	if idx := strings.Index(s, "."); idx >= 0 {
		intPart, decPart = s[:idx], s[idx:]
	}
	var b strings.Builder
	for i, ch := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}
	out := b.String() + decPart
	if negative {
		out = "-" + out
	}
	return out
}

// Int formats n with a comma between every three digits (1234567 -> "1,234,567").
func Int(n int64) string {
	return Commas(strconv.FormatInt(n, 10))
}
