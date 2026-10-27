package csvutils

import (
	"strings"
	"unicode"
)

func isFormulaPrefix(c byte) bool {
	switch c {
	case '=', '+', '-', '@', '\t', '\r':
		return true
	default:
		return false
	}
}

// SafeCell neutralises a cell a spreadsheet would read as a formula by
// prefixing it with a single quote, which every spreadsheet shows as text.
// The CSV writer still quotes and escapes the result as usual.
func SafeCell(value string) string {
	if value == "" || !isFormulaPrefix(value[0]) {
		return value
	}

	return "'" + value
}

// SafeRow applies SafeCell to every cell of a row in place and returns it.
func SafeRow(row []string) []string {
	for i := range row {
		row[i] = SafeCell(row[i])
	}

	return row
}

func SafeNumberCell(value string) string {
	if value == "" || !isFormulaPrefix(value[0]) {
		return value
	}
	if (value[0] == '-' || value[0] == '+') && isPlainNumber(value[1:]) {
		return value
	}

	return "'" + value
}

func isPlainNumber(value string) bool {
	runes := []rune(value)
	digits := 0
	for i, r := range runes {
		switch {
		case unicode.IsDigit(r):
			digits++
		case r == ',' || r == '.' || r == '%' || r == '\'' || unicode.IsSpace(r) ||
			unicode.Is(unicode.Sc, r):
		case strings.ContainsRune("KMBT", r) && i == len(runes)-1:
		default:
			return false
		}
	}

	return digits > 0
}
