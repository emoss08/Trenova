package assistantartifact

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const maxSlugRunes = 60

// SlugBase is the readable part of a lineage's link: its title in lower case,
// words joined by hyphens, accents dropped. A title with nothing to keep
// gives "artifact".
func SlugBase(title string) string {
	var b strings.Builder
	hyphen := false
	for _, r := range norm.NFKD.String(strings.ToLower(title)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			if hyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			hyphen = false
			b.WriteRune(r)
		default:
			hyphen = true
		}
		if b.Len() >= maxSlugRunes {
			break
		}
	}

	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "artifact"
	}

	return slug
}

// UniqueSlug is base, or base with the first free number after it, given
// the slugs the conversation already uses.
func UniqueSlug(base string, taken map[string]bool) string {
	if !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		candidate := base + "-" + strconv.Itoa(n)
		if !taken[candidate] {
			return candidate
		}
	}
}
