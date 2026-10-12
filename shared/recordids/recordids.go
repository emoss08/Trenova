// Package recordids takes Trenova's internal record ids out of text a person
// reads.
//
// It is the server's copy of the web client's withoutRecordIds
// (client/apps/web/src/lib/record-ids.ts), case for case. The agents are
// told never to show an id, and the client strips one that slips through as
// it renders; a reply read anywhere else (an email digest, the API, a
// transcript export) had them in it, so the reply is stripped where it is
// recorded too.
package recordids

import (
	"regexp"
	"strings"
)

const (
	// idPattern is a record's internal id as Trenova writes them: a short
	// lowercase prefix, an underscore and 26 characters of Crockford base32,
	// such as inv_01M42BNHACKS99T13QKY88TVXV. Go's regexp has no
	// lookaround, so the client's boundaries are checked by standsAlone.
	idPattern = `([a-z]{2,8}_[0-9A-HJKMNP-TV-Z]{26})`
	// wrapPattern is the marks a model wraps an id in: code ticks, bold.
	wrapPattern = "[`*_]*"
	// labelPattern is what a model calls an id beside it: "ID", "id:",
	// "invoiceId", "shipment ID".
	labelPattern = `(?:[A-Za-z]+\s)?[A-Za-z]*(?:ID|Id|id)\b:?\s*`
)

var (
	present = regexp.MustCompile(`[a-z]_[0-9A-HJKMNP-TV-Z]{26}`)
	// aside is "(ID **inv_…**)", "(`shp_…`)", "(invoiceId inv_…)": the whole
	// aside goes.
	aside = regexp.MustCompile(
		`\s*\(\s*(?:` + labelPattern + `)?` + wrapPattern + idPattern + wrapPattern + `\s*\)`,
	)
	// labelled is "ID **inv_…**" or ", id: ap_…" in a sentence: the label goes
	// with it.
	labelled = regexp.MustCompile(`,?\s*\b` + labelPattern + wrapPattern + idPattern + wrapPattern)
	// bare is any id left on its own, with the comma or space before it.
	bare = regexp.MustCompile(`,?\s*` + wrapPattern + idPattern + wrapPattern)

	spaceBeforePunctuation = regexp.MustCompile(`[ \t]+([,.;:!?)])`)
	emptyParentheses       = regexp.MustCompile(`\(\s*\)`)
	runOfSpaces            = regexp.MustCompile(`[ \t]{2,}`)
)

// Strip is the text with every internal record id taken out, and the words
// around each closed up. An id inside a link's address, an artifact link or
// a URL, is left alone, and text that held no id to take out comes back as
// it was, spacing and all: a reply with an artifact link and an aligned
// table is not an id to strip.
func Strip(text string) string {
	if !present.MatchString(text) {
		return text
	}

	out := remove(aside, text)
	out = remove(labelled, out)
	out = remove(bare, out)
	if out == text {
		return text
	}
	out = spaceBeforePunctuation.ReplaceAllString(out, "$1")
	out = emptyParentheses.ReplaceAllString(out, "")

	return runOfSpaces.ReplaceAllString(out, " ")
}

// Contains reports whether the text holds an id Strip would take out.
func Contains(text string) bool {
	return Strip(text) != text
}

// remove deletes each match whose id stands on its own: not inside a word,
// and not part of an address (after a colon, a slash, a dot, a query's = or
// & or #, or a hyphen).
func remove(pattern *regexp.Regexp, text string) string {
	matches := pattern.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var out strings.Builder
	out.Grow(len(text))
	last := 0
	for _, match := range matches {
		if !standsAlone(text, match[2], match[3]) {
			continue
		}
		out.WriteString(text[last:match[0]])
		last = match[1]
	}
	out.WriteString(text[last:])

	return out.String()
}

func standsAlone(text string, start, end int) bool {
	if start > 0 {
		switch before := text[start-1]; {
		case isWord(before):
			return false
		case strings.IndexByte(":/.=?&#-", before) >= 0:
			return false
		}
	}

	return end >= len(text) || !isWord(text[end])
}

func isWord(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
