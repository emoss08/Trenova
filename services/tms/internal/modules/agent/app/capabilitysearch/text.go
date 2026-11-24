package capabilitysearch

import (
	"strings"
	"unicode"
)

var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "the": {}, "of": {}, "to": {}, "for": {}, "in": {}, "on": {},
	"at": {}, "by": {}, "with": {}, "from": {}, "is": {}, "are": {}, "was": {}, "were": {},
	"be": {}, "been": {}, "it": {}, "its": {}, "this": {}, "that": {}, "these": {}, "those": {},
	"me": {}, "my": {}, "we": {}, "our": {}, "us": {}, "you": {}, "your": {}, "i": {},
	"please": {}, "can": {}, "could": {}, "would": {}, "should": {}, "do": {}, "does": {},
	"did": {}, "have": {}, "has": {}, "had": {}, "get": {}, "give": {}, "tell": {}, "need": {},
	"want": {}, "some": {}, "them": {}, "their": {}, "there": {}, "then": {}, "so": {},
	"just": {}, "right": {}, "now": {}, "up": {}, "about": {}, "into": {},
	"will": {}, "or": {}, "if": {}, "as": {}, "like": {}, "got": {},
	"tool": {}, "tools": {}, "data": {}, "one": {}, "ones": {}, "thing": {}, "things": {},
}

var readWords = map[string]struct{}{
	"list": {}, "show": {}, "find": {}, "which": {}, "what": {}, "whats": {}, "who": {},
	"whos": {}, "how": {}, "many": {}, "count": {}, "where": {}, "any": {}, "all": {},
	"unpaid": {}, "late": {}, "overdue": {}, "due": {}, "expiring": {}, "status": {},
	"everything": {}, "detail": {}, "details": {}, "past": {}, "open": {}, "pending": {},
	"failed": {}, "rejected": {}, "waiting": {}, "stuck": {}, "lookup": {}, "look": {},
	"search": {}, "view": {}, "read": {}, "check": {}, "running": {}, "free": {}, "available": {},
	"owe": {}, "owes": {}, "unread": {}, "unanswered": {}, "missing": {}, "total": {},
}

var writeWords = map[string][]string{
	"put":        {"place"},
	"hold":       {"place", "hold"},
	"comment":    {"add", "comment"},
	"email":      {"send", "email"},
	"text":       {"send", "message"},
	"message":    {"send", "message"},
	"remember":   {"create", "memory"},
	"post":       {"post"},
	"send":       {"send"},
	"void":       {"void"},
	"record":     {"record", "post"},
	"approve":    {"approve"},
	"assign":     {"assign"},
	"dispatch":   {"assign"},
	"tender":     {"send", "tender"},
	"issue":      {"issue"},
	"mark":       {"set", "status"},
	"create":     {"create"},
	"add":        {"add", "create"},
	"cancel":     {"cancel"},
	"rerate":     {"rerate"},
	"renew":      {"request", "renewal"},
	"dispute":    {"open", "dispute"},
	"generate":   {"generate"},
	"accept":     {"accept"},
	"reject":     {"reject"},
	"leave":      {"add"},
	"update":     {"update"},
	"change":     {"update", "set"},
	"delete":     {"delete"},
	"remove":     {"remove", "delete"},
	"apply":      {"apply", "post"},
	"bill":       {"approve", "transfer"},
	"pay":        {"record", "payment"},
	"schedule":   {"schedule", "create"},
	"reschedule": {"reschedule"},
	"release":    {"release"},
	"split":      {"split"},
	"duplicate":  {"duplicate"},
	"share":      {"share"},
	"file":       {"file"},
	"reply":      {"reply", "send"},
	"block":      {"block"},
	"resolve":    {"resolve"},
	"dismiss":    {"dismiss"},
	"reopen":     {"reopen"},
	"close":      {"close"},
	"lock":       {"lock"},
	"unlock":     {"unlock"},
	"vet":        {"vet"},
	"quote":      {"quote"},
	"forgive":    {"forgive"},
	"waive":      {"waive"},
	"escalate":   {"escalate"},
	"snooze":     {"snooze"},
}

var getWords = map[string]struct{}{
	"everything": {}, "detail": {}, "details": {}, "about": {}, "where": {}, "who": {},
	"whos": {}, "tracking": {}, "paperwork": {}, "comment": {}, "comments": {},
}

func Tokens(text string) []string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, "'", "")
	text = strings.ReplaceAll(text, "’", "")
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	return fields
}

func Stem(word string) string {
	switch {
	case len(word) <= 3:
		return word
	case strings.HasSuffix(word, "ies") && len(word) > 4:
		return word[:len(word)-3] + "y"
	case strings.HasSuffix(word, "sses"):
		return word[:len(word)-2]
	case strings.HasSuffix(word, "ches"), strings.HasSuffix(word, "shes"), strings.HasSuffix(word, "xes"):
		return word[:len(word)-2]
	case strings.HasSuffix(word, "ss"), strings.HasSuffix(word, "us"), strings.HasSuffix(word, "is"):
		return word
	case strings.HasSuffix(word, "s"):
		return word[:len(word)-1]
	}

	return word
}

func Terms(text string) []string {
	raw := Tokens(text)
	out := make([]string, 0, len(raw))
	for _, word := range raw {
		if _, stop := stopWords[word]; stop {
			continue
		}
		out = append(out, Stem(word))
	}

	return out
}

func phrase(text string) string {
	return strings.Join(Terms(text), " ")
}

func isNumber(word string) bool {
	if len(word) < 3 {
		return false
	}
	digits := 0
	for _, r := range word {
		if unicode.IsDigit(r) {
			digits++
		}
	}

	return digits*2 >= len(word)
}

var imperatives = map[string]struct{}{
	"draft": {}, "log": {}, "enter": {}, "fix": {}, "locate": {}, "pause": {}, "resume": {},
	"onboard": {}, "write": {}, "copy": {}, "amend": {}, "simulate": {}, "discard": {},
	"reverse": {}, "map": {}, "unmap": {}, "skip": {}, "backfill": {}, "match": {},
	"attach": {}, "turn": {}, "stop": {}, "make": {}, "flag": {}, "correct": {}, "import": {},
	"open": {}, "reset": {}, "set": {}, "start": {}, "submit": {}, "transfer": {}, "move": {},
	"unassign": {}, "uncancel": {}, "pin": {}, "unpin": {}, "revoke": {}, "run": {},
	"retry": {}, "replay": {}, "reprocess": {}, "link": {}, "unlink": {}, "upload": {},
	"request": {}, "invite": {}, "enable": {}, "disable": {}, "archive": {}, "restore": {},
	"commit": {}, "post": {}, "rename": {}, "edit": {}, "book": {}, "plan": {}, "notify": {},
	"ask": {}, "tell": {}, "call": {}, "watch": {}, "confirm": {}, "verify": {}, "award": {},
}
