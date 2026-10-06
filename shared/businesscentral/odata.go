package businesscentral

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

const (
	filterParam        = "$filter"
	topParam           = "$top"
	skipParam          = "$skip"
	maxFilterValueSize = 250
)

type number decimal.Decimal

func (n number) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(n).String()), nil
}

func numberOf(value decimal.Decimal) *number {
	out := number(value)
	return &out
}

func odataString(value string) (string, error) {
	if value == "" || !textWithin(value, maxFilterValueSize) || hasControl(value) {
		return "", ErrInvalidFilter
	}
	return "'" + strings.ReplaceAll(value, "'", "''") + "'", nil
}

func filterEquals(field, literal string) string {
	return field + " eq " + literal
}

func filterIn(field string, literals []string) string {
	return field + " in (" + strings.Join(literals, ",") + ")"
}

func filterModifiedSince(since *time.Time) string {
	if since == nil || since.IsZero() {
		return ""
	}
	return lastModifiedDateTimeField + " gt " + odataDateTime(*since)
}

func filterJoin(clauses ...string) string {
	kept := make([]string, 0, len(clauses))
	for _, clause := range clauses {
		if clause != "" {
			kept = append(kept, clause)
		}
	}
	return strings.Join(kept, " and ")
}

func textWithin(value string, limit int) bool {
	return utf8.RuneCountInString(value) <= limit
}

func hasControl(value string) bool {
	for _, r := range value {
		if r < ' ' || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return true
		}
	}
	return false
}

func cleanText(raw string, limit int) (string, error) {
	value := strings.TrimSpace(raw)
	if !textWithin(value, limit) {
		return "", ErrFieldTooLong
	}
	if hasControl(value) {
		return "", ErrInvalidText
	}
	return value, nil
}

func optionalGUID(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return guid(raw)
}
