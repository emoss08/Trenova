package board

import (
	"regexp"
	"slices"
	"strings"
)

const maxFiguresNamed = 4

var (
	fixedMoment = regexp.MustCompile(`(?i)` +
		`\b\d{1,2}:\d{2}(?:\s?[ap]\.?m\.?)?|` +
		`\b\d{1,2}\s?[ap]\.?m\.?\b|` +
		`\b\d{1,4}[/-]\d{1,2}(?:[/-]\d{1,4})?\b|` +
		`\b(?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\.?\s+\d{1,2}(?:st|nd|rd|th)?\b|` +
		`\b\d{1,2}(?:st|nd|rd|th)?\s+(?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\b|` +
		`\b(?:19|20)\d{2}\b|` +
		// A span of time named from now ("the next 24 hours", "the past 7
		// days") says which time the board covers, like a date; it is not a
		// reading that goes stale. Refused as a figure, an agent had no block
		// to move "the board covers the next 24 hours" into.
		`\b(?:next|past|last|coming|previous|prior|following)\s+\d{1,3}\s+` +
		`(?:hours?|days?|weeks?|months?)\b`)
	figureNumber = regexp.MustCompile(`\d[\d,]*(?:\.\d+)?`)
	figureUnit   = regexp.MustCompile(`^(?i)(?:%|(?:h|hr|hrs|m|min|mins|d|s|mi|km|k|lb|lbs)\b)`)
	figureWord   = regexp.MustCompile(`^\s+[\p{L}%]+`)
)

func Figures(text string) []string {
	masked := fixedMoment.ReplaceAllStringFunc(text, func(moment string) string {
		return strings.Repeat(" ", len(moment))
	})
	found := make([]string, 0, 2)
	for _, at := range figureNumber.FindAllStringIndex(masked, -1) {
		start, end := at[0], at[1]
		if start > 0 && identifierRune(masked[start-1]) {
			continue
		}
		rest := masked[end:]
		figure := masked[start:end]
		switch {
		case figureUnit.MatchString(rest):
			figure += figureUnit.FindString(rest)
		case rest != "" && identifierRune(rest[0]):
			continue
		case figureWord.MatchString(rest):
			figure += figureWord.FindString(rest)
		}
		if start > 0 && masked[start-1] == '$' {
			figure = "$" + figure
		}
		if !slices.Contains(found, figure) {
			found = append(found, figure)
		}
	}

	return found
}

func identifierRune(b byte) bool {
	return b == '-' || b == '_' || b == '#' || b == '/' || b == '.' ||
		(b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}
