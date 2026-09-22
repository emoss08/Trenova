package stringutils

import (
	"strings"
	"unicode"
)

// maxMessageIDLength is the longest msg-id a header line can carry.
const maxMessageIDLength = 998

var automatedLocalParts = []string{
	"noreply", "no-reply", "no_reply",
	"donotreply", "do-not-reply", "do_not_reply",
	"mailer-daemon", "postmaster", "bounce", "bounces",
}

// MessageIDToken normalizes a Message-ID to its bracketed header form and
// reports whether it is one a header may carry.
//
// The value comes from whoever sent the mail, so it is checked rather than
// trusted: anything with whitespace or a control character in it — the shape
// of a header injection — or without the local@domain form is refused.
func MessageIDToken(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "<")
	value = strings.TrimSuffix(value, ">")
	if value == "" || len(value)+2 > maxMessageIDLength {
		return "", false
	}

	at := strings.IndexByte(value, '@')
	if at <= 0 || at == len(value)-1 || strings.Count(value, "@") != 1 {
		return "", false
	}
	for _, r := range value {
		if r > unicode.MaxASCII || unicode.IsSpace(r) || unicode.IsControl(r) ||
			r == '<' || r == '>' {
			return "", false
		}
	}

	return "<" + value + ">", true
}

// ReplyReferences builds the References header for a reply: the thread the
// original carried, then the original itself, keeping only the newest limit
// entries so a long thread cannot grow the header without bound.
func ReplyReferences(references []string, messageID string, limit int) string {
	if limit <= 0 {
		return ""
	}

	tokens := make([]string, 0, len(references)+1)
	seen := make(map[string]struct{}, len(references)+1)
	for _, raw := range append(references, messageID) {
		token, ok := MessageIDToken(raw)
		if !ok {
			continue
		}
		if _, repeated := seen[token]; repeated {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	if len(tokens) > limit {
		tokens = tokens[len(tokens)-limit:]
	}

	return strings.Join(tokens, " ")
}

// ReplySubject is the subject a reply carries: the original's, with one "Re:"
// in front. A subject that already starts with one keeps it, so a thread does
// not grow a "Re: Re: Re:" chain.
func ReplySubject(subject string) string {
	subject = strings.Join(strings.Fields(subject), " ")
	if subject == "" {
		return "Re: your message"
	}
	if len(subject) >= 3 && strings.EqualFold(subject[:3], "re:") {
		return "Re:" + subject[3:]
	}

	return "Re: " + subject
}

// IsAutomatedSender reports whether an address is one nobody reads: a bounce
// handler, a postmaster, a no-reply sender. Answering one is at best wasted and
// at worst the start of a loop between two machines.
func IsAutomatedSender(address string) bool {
	address = NormalizeEmailAddress(address)
	at := strings.LastIndexByte(address, '@')
	if at <= 0 {
		return false
	}
	local := address[:at]
	if plus := strings.IndexByte(local, '+'); plus > 0 {
		local = local[:plus]
	}

	for _, automated := range automatedLocalParts {
		if local == automated || strings.HasPrefix(local, automated+".") ||
			strings.HasPrefix(local, automated+"-") {
			return true
		}
	}

	return strings.Contains(local, "noreply") || strings.Contains(local, "no-reply") ||
		strings.Contains(local, "donotreply")
}
