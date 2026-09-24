package piiscrub

import (
	"regexp"
	"strings"
)

const (
	MaskSSN     = "[SSN]"
	MaskCard    = "[CARD]"
	MaskRouting = "[ROUTING]"
	MaskAccount = "[ACCOUNT]"
)

const (
	minCardDigits       = 13
	maxCardDigits       = 19
	cardLeadGroupDigits = 4
	maxCardGroupDigits  = 6
	minScanDigits       = 4
	ibanMinLength       = 15
	ibanMaxLength       = 34
	routingDigits       = 9
	ssnDigits           = 9
	ibanCheckModulus    = 97
)

var (
	cardPattern = regexp.MustCompile(`\b\d(?:[ -]?\d){12,24}\b`)

	formattedSSNPattern = regexp.MustCompile(`\b\d{3}[- ]\d{2}[- ]\d{4}\b`)

	labelledSSNPattern = regexp.MustCompile(
		`(?i)\b(?:ssn|ss\s?#|social\s+security(?:\s+(?:number|no\.?|num|#))?)` +
			`[^0-9\n]{0,20}?(\d{3}[- ]?\d{2}[- ]?\d{4})\b`,
	)

	routingPattern = regexp.MustCompile(
		`(?i)\b(?:routing|aba|rtn|transit)\b(?:\s*(?:number|no\.?|num|#))?` +
			`[^0-9\n]{0,20}?(\d{9})\b`,
	)

	accountPattern = regexp.MustCompile(
		`(?i)\b(?:account|acct|a/c)\b(?:\s*(?:number|no\.?|num|#))?` +
			`[^0-9\n]{0,20}?(\d(?:[ -]?\d){3,16})\b`,
	)

	ibanPattern = regexp.MustCompile(`\b[A-Z]{2}\d{2}(?: ?[A-Z0-9]){11,30}\b`)
)

type locator func(candidate string) (start, end int, ok bool)

type rule struct {
	pattern *regexp.Regexp
	group   int
	locate  locator
	mask    string
}

var rules = []rule{
	{pattern: ibanPattern, group: 0, locate: locateIBAN, mask: MaskAccount},
	{pattern: cardPattern, group: 0, locate: locateCard, mask: MaskCard},
	{pattern: formattedSSNPattern, group: 0, locate: whole(validFormattedSSN), mask: MaskSSN},
	{pattern: labelledSSNPattern, group: 1, locate: whole(validSSNDigits), mask: MaskSSN},
	{pattern: routingPattern, group: 1, locate: whole(validRoutingNumber), mask: MaskRouting},
	{pattern: accountPattern, group: 1, locate: whole(always), mask: MaskAccount},
}

func Scrub(text string) string {
	if !worthScanning(text) {
		return text
	}

	for idx := range rules {
		text = rules[idx].apply(text)
	}

	return text
}

func ScrubAll(texts []string) []string {
	out := make([]string, len(texts))
	for idx, text := range texts {
		out[idx] = Scrub(text)
	}

	return out
}

func worthScanning(text string) bool {
	digits := 0
	for idx := 0; idx < len(text); idx++ {
		if isDigit(text[idx]) {
			digits++
			if digits >= minScanDigits {
				return true
			}
		}
	}

	return false
}

func (r *rule) apply(text string) string {
	matches := r.pattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var b strings.Builder
	cursor := 0
	for _, match := range matches {
		groupStart, groupEnd := match[2*r.group], match[2*r.group+1]
		if groupStart < 0 {
			continue
		}

		start, end, ok := r.locate(text[groupStart:groupEnd])
		if !ok {
			continue
		}

		if b.Len() == 0 {
			b.Grow(len(text))
		}
		b.WriteString(text[cursor : groupStart+start])
		b.WriteString(r.mask)
		cursor = groupStart + end
	}

	if cursor == 0 {
		return text
	}

	b.WriteString(text[cursor:])

	return b.String()
}

func whole(accept func(candidate string) bool) locator {
	return func(candidate string) (int, int, bool) {
		return 0, len(candidate), accept(candidate)
	}
}

func always(string) bool { return true }

type digitGroup struct {
	start int
	end   int
}

func locateCard(candidate string) (int, int, bool) {
	groups := splitGroups(candidate, isDigit)

	for size := len(groups); size > 0; size-- {
		for first := 0; first+size <= len(groups); first++ {
			window := groups[first : first+size]
			if !cardShaped(window) {
				continue
			}
			start, end := window[0].start, window[size-1].end
			if validCard(candidate[start:end]) {
				return start, end, true
			}
		}
	}

	return 0, 0, false
}

func cardShaped(groups []digitGroup) bool {
	if len(groups) == 1 {
		return true
	}
	if groups[0].len() != cardLeadGroupDigits {
		return false
	}
	for _, group := range groups {
		if group.len() > maxCardGroupDigits {
			return false
		}
	}

	return true
}

func (g digitGroup) len() int { return g.end - g.start }

func splitGroups(candidate string, member func(byte) bool) []digitGroup {
	groups := make([]digitGroup, 0, 8)
	start := -1
	for idx := 0; idx < len(candidate); idx++ {
		if member(candidate[idx]) {
			if start < 0 {
				start = idx
			}

			continue
		}
		if start >= 0 {
			groups = append(groups, digitGroup{start: start, end: idx})
			start = -1
		}
	}
	if start >= 0 {
		groups = append(groups, digitGroup{start: start, end: len(candidate)})
	}

	return groups
}

func locateIBAN(candidate string) (int, int, bool) {
	groups := splitGroups(candidate, isIBANChar)
	for size := len(groups); size > 0; size-- {
		end := groups[size-1].end
		if validIBAN(candidate[:end]) {
			return 0, end, true
		}
	}

	return 0, 0, false
}

func isIBANChar(ch byte) bool {
	return isDigit(ch) || (ch >= 'A' && ch <= 'Z')
}

func validCard(span string) bool {
	digits := 0
	sum := 0
	double := false
	for idx := len(span) - 1; idx >= 0; idx-- {
		ch := span[idx]
		if !isDigit(ch) {
			continue
		}
		digits++
		value := int(ch - '0')
		if double {
			value *= 2
			if value > 9 {
				value -= 9
			}
		}
		sum += value
		double = !double
	}

	if digits < minCardDigits || digits > maxCardDigits {
		return false
	}
	if allSameDigit(span) {
		return false
	}

	return sum%10 == 0
}

func allSameDigit(span string) bool {
	var first byte
	for idx := 0; idx < len(span); idx++ {
		ch := span[idx]
		if !isDigit(ch) {
			continue
		}
		if first == 0 {
			first = ch

			continue
		}
		if ch != first {
			return false
		}
	}

	return true
}

func validFormattedSSN(candidate string) bool {
	if len(candidate) != 11 || candidate[3] != candidate[6] {
		return false
	}

	return validSSNDigits(candidate)
}

func validSSNDigits(candidate string) bool {
	digits := make([]byte, 0, ssnDigits)
	for idx := 0; idx < len(candidate); idx++ {
		if isDigit(candidate[idx]) {
			digits = append(digits, candidate[idx])
		}
	}
	if len(digits) != ssnDigits {
		return false
	}

	area := string(digits[:3])
	group := string(digits[3:5])
	serial := string(digits[5:])

	switch {
	case area == "000", area == "666", digits[0] == '9':
		return false
	case group == "00":
		return false
	case serial == "0000":
		return false
	default:
		return true
	}
}

func validRoutingNumber(candidate string) bool {
	if len(candidate) != routingDigits {
		return false
	}

	weights := [routingDigits]int{3, 7, 1, 3, 7, 1, 3, 7, 1}
	sum := 0
	for idx := range routingDigits {
		ch := candidate[idx]
		if !isDigit(ch) {
			return false
		}
		sum += int(ch-'0') * weights[idx]
	}

	return sum%10 == 0 && !allSameDigit(candidate)
}

func validIBAN(candidate string) bool {
	compact := make([]byte, 0, len(candidate))
	for idx := 0; idx < len(candidate); idx++ {
		if candidate[idx] != ' ' {
			compact = append(compact, candidate[idx])
		}
	}
	if len(compact) < ibanMinLength || len(compact) > ibanMaxLength {
		return false
	}

	rotated := make([]byte, 0, len(compact))
	rotated = append(rotated, compact[4:]...)
	rotated = append(rotated, compact[:4]...)

	remainder := 0
	for _, ch := range rotated {
		switch {
		case isDigit(ch):
			remainder = (remainder*10 + int(ch-'0')) % ibanCheckModulus
		case ch >= 'A' && ch <= 'Z':
			value := int(ch-'A') + 10
			remainder = (remainder*100 + value) % ibanCheckModulus
		default:
			return false
		}
	}

	return remainder == 1
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
