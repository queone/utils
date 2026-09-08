package numfmt

import "testing"

func TestCommasGroupsIntegerDigits(t *testing.T) {
	cases := map[string]string{
		"1000":       "1,000",
		"1234567":    "1,234,567",
		"123456789":  "123,456,789",
		"1000000000": "1,000,000,000",
	}
	for in, want := range cases {
		if got := Commas(in); got != want {
			t.Errorf("Commas(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCommasKeepsSignAndFraction(t *testing.T) {
	cases := map[string]string{
		"-1234567.89": "-1,234,567.89",
		"1234.5":      "1,234.5",
		"-1000":       "-1,000",
		"12345.":      "12,345.",
	}
	for in, want := range cases {
		if got := Commas(in); got != want {
			t.Errorf("Commas(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCommasLeavesShortValuesUnchanged(t *testing.T) {
	for _, in := range []string{"999", "0", "-5", "12.34", "", "-"} {
		if got := Commas(in); got != in {
			t.Errorf("Commas(%q) = %q, want it unchanged", in, got)
		}
	}
}

func TestIntGroupsDigits(t *testing.T) {
	cases := map[int64]string{
		0:          "0",
		999:        "999",
		1000:       "1,000",
		1234567:    "1,234,567",
		-1234567:   "-1,234,567",
		9007199254: "9,007,199,254",
	}
	for in, want := range cases {
		if got := Int(in); got != want {
			t.Errorf("Int(%d) = %q, want %q", in, got, want)
		}
	}
}
