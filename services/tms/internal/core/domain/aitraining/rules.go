package aitraining

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	anonymousURL    = "https://www.example.com"
	anonymousDomain = "example.com"
	minMixedDigits  = 2
	minMixedLetters = 1
	minMixedRunes   = 6
)

type rule func(p *pseudonymizer, text string) string

var (
	emailPattern  = regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)
	urlPattern    = regexp.MustCompile(`(?i)\b(?:https?://|www\.)[^\s<>"'\x{E000}-\x{E0FF}]+`)
	domainPattern = regexp.MustCompile(
		`(?i)\b(?:[a-z0-9](?:[a-z0-9\-]{0,61}[a-z0-9])?\.)+(?:com|net|org|io|us|biz|info|ca|mx)\b`,
	)
	carrierNumberPattern = regexp.MustCompile(
		`(?i)\b(?:us\s?dot|dot|mc|mx|ff)\s*(?:#|no\.?|number|num)?\s*[:\-]?\s*(\d{4,8})\b`,
	)
	phonePattern = regexp.MustCompile(
		`(?:\+?1[\s.\-]?)?(?:\(\d{3}\)\s?|\b\d{3}[\s.\-])\d{3}[\s.\-]\d{4}\b`,
	)
	moneyPattern = regexp.MustCompile(
		`(?i)(?:\$\s?|\busd\s?)\d{1,3}(?:,\d{3})+(?:\.\d{2})?\b|` +
			`(?i)(?:\$\s?|\busd\s?)\d+(?:\.\d{2})?\b|` +
			`\b\d{1,3}(?:,\d{3})+\.\d{2}\b`,
	)
	taxIDPattern  = regexp.MustCompile(`\b\d{2}-\d{7}\b|\b\d{3}-\d{2}-\d{4}\b`)
	poBoxPattern  = regexp.MustCompile(`(?i)\bp\.?\s?o\.?\s*box\s+(\d+)\b`)
	streetPattern = regexp.MustCompile(
		`(?i)\b\d{1,6}\s+(?:[a-z0-9.'\-]+\s+){0,4}?` +
			`(?:st|street|ave|avenue|rd|road|blvd|boulevard|dr|drive|ln|lane|hwy|highway|` +
			`pkwy|parkway|court|place|cir|circle|trl|trail|pike|terrace)\b\.?`,
	)
	stateZipPattern   = regexp.MustCompile(`\b[A-Z]{2}[\s,]+(\d{5})(?:-(\d{4}))?\b`)
	digitRunPattern   = regexp.MustCompile(`\b\d{6,}\b`)
	mixedTokenPattern = regexp.MustCompile(`\b[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?\b`)

	addressRules = []rule{
		replaceEmails,
		replaceURLs,
		replaceDomains,
	}

	genericRules = []rule{
		replaceTaxIDs,
		replaceCarrierNumbers,
		replacePhones,
		replaceMoney,
		replacePOBoxes,
		replaceStreets,
		replaceStateZips,
		replaceDigitRuns,
		replaceMixedTokens,
	}
)

func replaceEmails(p *pseudonymizer, text string) string {
	return replaceMatches(text, emailPattern, nil, func(match []int) string {
		key := strings.ToLower(text[match[0]:match[1]])
		return p.hold(p.surrogate(categoryEmail, key, func() string { return emailSurrogate(p.rng) }))
	})
}

func replaceURLs(p *pseudonymizer, text string) string {
	return replaceMatches(text, urlPattern, nil, func([]int) string {
		return p.hold(anonymousURL)
	})
}

func replaceDomains(p *pseudonymizer, text string) string {
	return replaceMatches(text, domainPattern, nil, func([]int) string {
		return p.hold(anonymousDomain)
	})
}

func replaceTaxIDs(p *pseudonymizer, text string) string {
	return replaceMatches(text, taxIDPattern, nil, func(match []int) string {
		original := text[match[0]:match[1]]
		return p.hold(p.identifier(stringutils.NormalizeIdentifier(original), original))
	})
}

func replaceCarrierNumbers(p *pseudonymizer, text string) string {
	return replaceMatches(text, carrierNumberPattern, nil, func(match []int) string {
		digits := text[match[2]:match[3]]
		return text[match[0]:match[2]] + p.hold(p.identifier(digits, digits)) + text[match[3]:match[1]]
	})
}

func replacePhones(p *pseudonymizer, text string) string {
	return replaceMatches(text, phonePattern, nil, func(match []int) string {
		original := text[match[0]:match[1]]
		key := stringutils.DigitsOnly(original)
		value := p.surrogate(categoryPhone, key, func() string { return shapeSurrogate(p.rng, key) })
		return p.hold(reshape(original, value))
	})
}

func replaceMoney(p *pseudonymizer, text string) string {
	return replaceMatches(text, moneyPattern, moneyBounded, func(match []int) string {
		original := text[match[0]:match[1]]
		parsed, err := decimalutils.ParseMoneyText(strings.TrimLeftFunc(original, isNotDigitRune))
		if err != nil || !parsed.Valid {
			return original
		}
		return p.hold(p.money(original, parsed.Decimal.Round(2)))
	})
}

func replacePOBoxes(p *pseudonymizer, text string) string {
	return replaceMatches(text, poBoxPattern, nil, func(match []int) string {
		digits := text[match[2]:match[3]]
		return text[match[0]:match[2]] + p.hold(p.identifier(digits, digits)) + text[match[3]:match[1]]
	})
}

func replaceStreets(p *pseudonymizer, text string) string {
	return replaceMatches(text, streetPattern, nil, func(match []int) string {
		original := text[match[0]:match[1]]
		key := canonicalStreet(strings.Fields(stringutils.NormalizeName(original)))
		return p.hold(styleLike(p.surrogate(categoryStreet, key, p.street), original))
	})
}

func replaceStateZips(p *pseudonymizer, text string) string {
	return replaceMatches(text, stateZipPattern, nil, func(match []int) string {
		five := text[match[2]:match[3]]
		extension := ""
		if match[4] >= 0 {
			extension = text[match[3]:match[1]]
		}
		return text[match[0]:match[2]] + p.hold(p.postal(five, extension))
	})
}

func replaceDigitRuns(p *pseudonymizer, text string) string {
	return replaceMatches(text, digitRunPattern, nil, func(match []int) string {
		digits := text[match[0]:match[1]]
		if looksLikeCompactDate(digits) {
			return digits
		}
		return p.hold(p.identifier(digits, digits))
	})
}

func replaceMixedTokens(p *pseudonymizer, text string) string {
	return replaceMatches(text, mixedTokenPattern, nil, func(match []int) string {
		token := text[match[0]:match[1]]
		if !identifierLike(token) {
			return token
		}
		return p.hold(p.identifier(stringutils.NormalizeIdentifier(token), token))
	})
}

func identifierLike(token string) bool {
	if len(token) < minMixedRunes || unitSuffixPattern.MatchString(token) {
		return false
	}
	digits, letters := 0, 0
	for _, r := range token {
		switch {
		case isDigitRune(r):
			digits++
		case unicode.IsLetter(r):
			letters++
		}
	}

	return digits >= minMixedDigits && letters >= minMixedLetters
}

func isNotDigitRune(r rune) bool {
	return !isDigitRune(r)
}
