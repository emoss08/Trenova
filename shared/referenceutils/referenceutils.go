// Package referenceutils pulls the tokens out of free text that could be a
// freight reference: a pro number, a BOL, a load or order number.
package referenceutils

import (
	"regexp"
	"strings"
)

const (
	minReferenceLength = 4
	maxReferenceLength = 100
)

// token is a run of characters that could be a pro number or a BOL.
//
// It is deliberately loose. Pro numbers are free text the organization formats
// however it likes, so there is no pattern to match against: the extractor
// proposes and the database disposes. A candidate that matches no record is
// simply not a match, which costs one indexed lookup and nothing else.
var token = regexp.MustCompile(`\b[A-Za-z0-9]+(?:[-_/][A-Za-z0-9]+)*\b`)

// mention is a reference as a payer writes it in a memo: segments joined by a
// hyphen, an underscore or a dot. A slash, a comma or a space separates one
// reference from the next, so "INV-1001/INV-1002" names two invoices.
var mention = regexp.MustCompile(`[A-Za-z0-9]+(?:[-_.][A-Za-z0-9]+)*`)

// separators are the joiners a payer swaps freely; "INV.1001" and "INV_1001"
// are the reference "INV-1001" written another way.
var separators = strings.NewReplacer(".", "-", "_", "-")

// hasDigit is what separates a reference from a word. "Thursday" and "confirm"
// are not reference numbers; "88213" and "SEED-SHP-001" are.
var hasDigit = regexp.MustCompile(`[0-9]`)

// Candidates returns up to limit distinct tokens worth looking up, in the
// order they appear across sources, so the earliest mention wins.
func Candidates(limit int, sources ...string) []string {
	if limit <= 0 {
		return nil
	}

	seen := make(map[string]struct{}, limit)
	candidates := make([]string, 0, limit)

	for _, source := range sources {
		for _, match := range token.FindAllString(source, -1) {
			if len(candidates) >= limit {
				return candidates
			}
			if !Plausible(match) {
				continue
			}

			key := strings.ToUpper(match)
			if _, repeated := seen[key]; repeated {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, match)
		}
	}

	return candidates
}

// Plausible keeps the tokens that could name a record: a digit in it, and long
// enough not to be a quantity. "2" and "48" are pallet counts and hours;
// "88213" and "BOL-2026-0001" are things to look up.
func Plausible(candidate string) bool {
	if len(candidate) < minReferenceLength || len(candidate) > maxReferenceLength {
		return false
	}

	return hasDigit.MatchString(candidate)
}

func Same(a, b string) bool {
	left := canonical(a)

	return left != "" && left == canonical(b)
}

func Mentions(reference string, sources ...string) bool {
	want := canonical(reference)
	if want == "" {
		return false
	}

	for _, source := range sources {
		for _, match := range mention.FindAllString(source, -1) {
			if canonical(match) == want {
				return true
			}
		}
	}

	return false
}

func canonical(reference string) string {
	return strings.ToUpper(separators.Replace(strings.TrimSpace(reference)))
}
