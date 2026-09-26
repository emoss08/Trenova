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
