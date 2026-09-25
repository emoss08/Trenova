package csvutils

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
