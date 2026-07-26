package reportfmt

import "strings"

// ExcelFormat reports the spreadsheet number-format code for a column and
// whether the value can be written as a native Excel number/date. Text-only
// displays (durations, compact notation, booleans, strings) fall back to the
// rendered string so the sheet always matches the other export formats.
func (r *Resolved) ExcelFormat() (code string, native bool) {
	// A banded cell reads as the range it covers, so its text is the value.
	if !r.Band.IsEmpty() {
		return "", false
	}

	switch r.Style {
	case StyleDate:
		return excelDateCode(r.DateStyle), true
	case StyleNumber, StyleCurrency, StylePercent:
		if r.Notation == NotationCompact {
			return "", false
		}
		return r.excelNumberCode(), true
	case StyleAuto, StyleDuration, StyleBool, StyleText:
		return "", false
	default:
		return "", false
	}
}

func (r *Resolved) excelNumberCode() string {
	var sb strings.Builder

	sb.WriteString(excelLiteral(r.Prefix))
	if r.Style == StyleCurrency {
		sb.WriteString(excelLiteral(currencySymbol(r.Currency)))
	}

	if r.Grouping {
		sb.WriteString("#,##0")
	} else {
		sb.WriteString("0")
	}
	if r.Decimals > 0 {
		sb.WriteByte('.')
		sb.WriteString(strings.Repeat("0", r.Decimals))
	}

	if r.Style == StylePercent {
		sb.WriteByte('%')
	}
	sb.WriteString(excelLiteral(r.Suffix))

	positive := sb.String()
	if r.Negative == NegativeParentheses {
		return positive + ";(" + positive + ")"
	}
	return positive
}

func excelDateCode(style DateStyle) string {
	switch style {
	case DateStyleDate:
		return "yyyy-mm-dd"
	case DateStyleDateLong:
		return `mmm d", "yyyy`
	case DateStyleTime:
		return "hh:mm"
	case DateStyleMonthYear:
		return "mmm yyyy"
	case DateStyleQuarter, DateStyleYear:
		return "yyyy"
	case DateStyleWeekday:
		return "ddd yyyy-mm-dd"
	case DateStyleISO:
		return "yyyy-mm-dd\\Thh:mm:ss"
	case DateStyleAuto, DateStyleDateTime:
		return "yyyy-mm-dd hh:mm"
	default:
		return "yyyy-mm-dd hh:mm"
	}
}

func excelLiteral(text string) string {
	if text == "" {
		return ""
	}
	return `"` + strings.ReplaceAll(text, `"`, "") + `"`
}
