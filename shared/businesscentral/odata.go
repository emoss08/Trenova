package businesscentral

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/shopspring/decimal"
)

const (
	filterParam        = "$filter"
	topParam           = "$top"
	skipParam          = "$skip"
	maxFilterValueSize = 250
)

type Number decimal.Decimal

func (n Number) MarshalJSON() ([]byte, error) {
	return []byte(decimal.Decimal(n).String()), nil
}

func NumberOf(value decimal.Decimal) *Number {
	out := Number(value)
	return &out
}

func ODataString(value string) (string, error) {
	if value == "" || !TextWithin(value, maxFilterValueSize) || HasControl(value) {
		return "", businesscentralerrors.ErrInvalidFilter
	}
	return "'" + strings.ReplaceAll(value, "'", "''") + "'", nil
}

func FilterEquals(field, literal string) string {
	return field + " eq " + literal
}

func FilterIn(field string, literals []string) string {
	return field + " in (" + strings.Join(literals, ",") + ")"
}

func FilterModifiedSince(since *time.Time) string {
	if since == nil || since.IsZero() {
		return ""
	}
	return LastModifiedDateTimeField + " gt " + odataDateTime(*since)
}

func FilterJoin(clauses ...string) string {
	kept := make([]string, 0, len(clauses))
	for _, clause := range clauses {
		if clause != "" {
			kept = append(kept, clause)
		}
	}
	return strings.Join(kept, " and ")
}

func TextWithin(value string, limit int) bool {
	return utf8.RuneCountInString(value) <= limit
}

func HasControl(value string) bool {
	for _, r := range value {
		if r < ' ' || r == 0x7f || (r >= 0x80 && r < 0xa0) {
			return true
		}
	}
	return false
}

func CleanText(raw string, limit int) (string, error) {
	value := strings.TrimSpace(raw)
	if !TextWithin(value, limit) {
		return "", businesscentralerrors.ErrFieldTooLong
	}
	if HasControl(value) {
		return "", businesscentralerrors.ErrInvalidText
	}
	return value, nil
}

func OptionalGUID(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return GUID(raw)
}

const (
	LastModifiedDateTimeField = "lastModifiedDateTime"
)

const (
	MaxDisplayNameLength  = 100
	maxCurrencyCodeLength = 10
	blankEnumValue        = "_x0020_"
)

func EnumValue(raw string) string {
	value := strings.TrimSpace(raw)
	if value == blankEnumValue {
		return ""
	}
	return value
}

func EmptyGUID(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "00000000-0000-0000-0000-000000000000" {
		return ""
	}
	return value
}

func CurrencyCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if !TextWithin(code, maxCurrencyCodeLength) || HasControl(code) {
		return "", businesscentralerrors.ErrCurrencyCodeInvalid
	}
	return code, nil
}
