//go:build fuzz

package csvutils

import (
	"strings"
	"testing"
)

const formulaStarts = "=+-@\t\r"

func FuzzSafeCellNeverLeavesAFormula(f *testing.F) {
	for _, seed := range []string{
		"", "Acme Freight", "=HYPERLINK(\"http://x\")", "+1", "-1", "@SUM(A1)", "\t=1", "\r=1", "'=1",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		got := SafeCell(value)

		if got != "" && strings.ContainsRune(formulaStarts, rune(got[0])) {
			t.Fatalf("%q still starts like a formula: %q", value, got)
		}
		if got != value && got != "'"+value {
			t.Fatalf("%q changed beyond its quote: %q", value, got)
		}
	})
}

func FuzzSafeNumberCellKeepsSignedNumbersAndNothingElse(f *testing.F) {
	for _, seed := range []string{"-12.50", "+3", "-", "-1e5", "-=1", "=1", "-1,234.00", "12"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		got := SafeNumberCell(value)

		if got == "'"+value {
			return
		}
		if got != value {
			t.Fatalf("%q changed beyond its quote: %q", value, got)
		}
		if value == "" {
			return
		}
		signedNumber := (value[0] == '-' || value[0] == '+') && isPlainNumber(value[1:])
		if value != "" && strings.ContainsRune(formulaStarts, rune(value[0])) && !signedNumber {
			t.Fatalf("%q was left starting like a formula", value)
		}
	})
}
