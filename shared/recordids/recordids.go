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
	bare      = regexp.MustCompile(`,?\s*` + wrapPattern + idPattern + wrapPattern)
	statement = regexp.MustCompile(`(,?[ \t]+and[ \t]+|[ \t]*)\b` +
		`(?:[Ii]ts|[Tt]he|[Tt]heir|[Tt]his|[Tt]hat|[Yy]our|[Oo]ur|` +
		`[A-Z]\w*(?:[ \t]+[A-Z]\w*){0,3}['’]s)` +
		`(?:[ \t]+[\w'’]+){0,2}[ \t]+(?:[A-Za-z]*(?:ID|Id)|id)` +
		`(?:[ \t]+(?:is|was)[ \t]+|[ \t]*:[ \t]*)` +
		wrapPattern + idPattern + wrapPattern + `([.!?]?)`)

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

	out := removeStatements(text)
	out = remove(aside, out)
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

func removeStatements(text string) string {
	matches := statement.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var out strings.Builder
	out.Grow(len(text))
	last := 0
	for _, match := range matches {
		if !standsAlone(text, match[4], match[5]) || !endsSentence(text, match) {
			continue
		}
		start, end := match[0], match[1]
		switch {
		case strings.Contains(text[match[2]:match[3]], "and"):
			end = match[6]
		case !startsSentence(text, start):
			continue
		case start == 0 || text[start-1] == '\n':
			end += len(text[end:]) - len(strings.TrimLeft(text[end:], " \t"))
			if end < len(text) && text[end] == '\n' {
				end++
			}
		}
		out.WriteString(text[last:start])
		last = end
	}
	out.WriteString(text[last:])

	return out.String()
}

func endsSentence(text string, match []int) bool {
	rest := strings.TrimLeft(text[match[1]:], " \t")
	if match[7] > match[6] {
		return match[1] == len(text) || rest != text[match[1]:] || text[match[1]] == '\n'
	}

	return rest == "" || rest[0] == '\n'
}

func startsSentence(text string, start int) bool {
	before := strings.TrimRight(text[:start], " \t")

	return before == "" || strings.IndexByte(".!?\n", before[len(before)-1]) >= 0
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

var found = regexp.MustCompile(`\b` + idPattern + `\b`)

// Find is the record ids the text holds, each once and in the order they first
// appear, at most limit of them.
func Find(text string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	matches := found.FindAllString(text, -1)
	ids := make([]string, 0, min(len(matches), limit))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range matches {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		if len(ids) == limit {
			break
		}
	}

	return ids
}

var anyCase = regexp.MustCompile(`(?i)\b` + idPattern + `\b`)

func Mentioned(text string) bool {
	return anyCase.MatchString(text)
}
