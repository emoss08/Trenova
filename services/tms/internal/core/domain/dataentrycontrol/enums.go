package dataentrycontrol

import "errors"

var ErrInvalidCaseFormat = errors.New("invalid case format")

type CaseFormat string

const (
	CaseFormatAsEntered = CaseFormat("AsEntered")
	CaseFormatUpper     = CaseFormat("Upper")
	CaseFormatLower     = CaseFormat("Lower")
	CaseFormatTitleCase = CaseFormat("TitleCase")
)

func (c CaseFormat) String() string {
	return string(c)
}

func (c CaseFormat) IsValid() bool {
	switch c {
	case CaseFormatAsEntered, CaseFormatUpper, CaseFormatLower, CaseFormatTitleCase:
		return true
	default:
		return false
	}
}

func CaseFormatFromString(s string) (CaseFormat, error) {
	switch s {
	case "AsEntered":
		return CaseFormatAsEntered, nil
	case "Upper":
		return CaseFormatUpper, nil
	case "Lower":
		return CaseFormatLower, nil
	case "TitleCase":
		return CaseFormatTitleCase, nil
	default:
		return "", ErrInvalidCaseFormat
	}
}
