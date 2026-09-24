package stringutils

import (
	"iter"
	"regexp"
	"strings"
)

var quotedReplyIntro = regexp.MustCompile(`^On .{4,200} wrote:$`)

// MailPreview is the line or two of an email a list row shows: the sender's own
// words, with whitespace folded, quoted lines dropped, and nothing from the
// point a quoted reply, a forwarded header or a signature begins.
func MailPreview(body string, maxLength int) string {
	if maxLength <= 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(min(len(body), maxLength*4+4))

	for line := range mailOwnLines(body) {
		for _, word := range strings.Fields(line) {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(word)
		}

		if b.Len() > maxLength*4 {
			break
		}
	}

	return Ellipsize(b.String(), maxLength)
}

func MailBody(body string) string {
	var b strings.Builder
	b.Grow(len(body))

	for line := range mailOwnLines(body) {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}

	return b.String()
}

func mailOwnLines(body string) iter.Seq[string] {
	return func(yield func(string) bool) {
		for line := range strings.Lines(body) {
			line = strings.TrimRight(line, "\r\n")
			if isMailBoundary(line) {
				return
			}

			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, ">") {
				continue
			}

			if !yield(trimmed) {
				return
			}
		}
	}
}

func isMailBoundary(line string) bool {
	if line == "-- " || line == "--" {
		return true
	}

	trimmed := strings.TrimSpace(line)
	switch {
	case strings.HasPrefix(trimmed, "-----Original Message-----"),
		strings.HasPrefix(trimmed, "---------- Forwarded message"),
		strings.HasPrefix(trimmed, "________________________________"):
		return true
	}

	return quotedReplyIntro.MatchString(trimmed)
}
