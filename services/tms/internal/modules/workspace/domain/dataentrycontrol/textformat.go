package dataentrycontrol

import (
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type TextFormattable interface {
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
	ApplyTextFormat(format TextFormat)
}

type TextFormat struct {
	CodeCase  CaseFormat
	NameCase  CaseFormat
	EmailCase CaseFormat
	CityCase  CaseFormat
}

func (dec *DataEntryControl) TextFormat() TextFormat {
	return TextFormat{
		CodeCase:  dec.CodeCase,
		NameCase:  dec.NameCase,
		EmailCase: dec.EmailCase,
		CityCase:  dec.CityCase,
	}
}

func (f TextFormat) Code(s string) string {
	return applyCase(strings.Join(strings.Fields(s), ""), f.CodeCase)
}

func (f TextFormat) Name(s string) string {
	return applyCase(CleanText(s), f.NameCase)
}

func (f TextFormat) City(s string) string {
	return applyCase(CleanText(s), f.CityCase)
}

func (f TextFormat) Email(s string) string {
	s = strings.TrimSpace(s)
	switch f.EmailCase { //nolint:exhaustive // title case never applies to an email address
	case CaseFormatUpper:
		return strings.ToUpper(s)
	case CaseFormatLower:
		return strings.ToLower(s)
	default:
		return s
	}
}

func CleanText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func applyCase(s string, format CaseFormat) string {
	if s == "" {
		return s
	}

	switch format {
	case CaseFormatUpper:
		return strings.ToUpper(s)
	case CaseFormatLower:
		return strings.ToLower(s)
	case CaseFormatTitleCase:
		return cases.Title(language.English, cases.NoLower).String(s)
	case CaseFormatAsEntered:
		return s
	default:
		return s
	}
}
