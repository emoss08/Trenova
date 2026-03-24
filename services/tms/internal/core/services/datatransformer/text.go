package datatransformer

import (
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/dataentrycontrol"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var multiSpaceRegex = regexp.MustCompile(`\s+`)

func cleanText(s string) string {
	s = strings.TrimSpace(s)
	s = multiSpaceRegex.ReplaceAllString(s, " ")
	return s
}

func cleanCode(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func toUpper(s string) string {
	return strings.ToUpper(s)
}

func toLower(s string) string {
	return strings.ToLower(s)
}

func toTitleCase(s string) string {
	if s == "" {
		return s
	}

	caser := cases.Title(language.English)
	titled := caser.String(s)

	return titled
}

func applyCase(s string, format dataentrycontrol.CaseFormat) string {
	if s == "" {
		return s
	}

	switch format { //nolint:exhaustive // only 4 cases are valid
	case dataentrycontrol.CaseFormatUpper:
		return toUpper(s)
	case dataentrycontrol.CaseFormatLower:
		return toLower(s)
	case dataentrycontrol.CaseFormatTitleCase:
		return toTitleCase(s)
	default:
		return s
	}
}
