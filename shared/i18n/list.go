package i18n

import (
	"context"
	"strings"
)

// The marks List formats a list pattern with, to read the separators a
// translation puts between its items.
const (
	listFirst  = "\x00"
	listSecond = "\x01"
	listLast   = "\x02"
)

// List joins items the way a sentence in the request's language lists them:
// "code, name and address". The separators come from the translations of
// two patterns, so adding a language needs no code: a list of three or more
// takes the separator between the first two items of "{0}, {1} and {2}" for
// every join but the last, and the one before its last item for that.
func List(ctx context.Context, items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return T(ctx, "{0} and {1}", items[0], items[1])
	}

	pattern := T(ctx, "{0}, {1} and {2}", listFirst, listSecond, listLast)
	first := strings.Index(pattern, listFirst)
	second := strings.Index(pattern, listSecond)
	last := strings.Index(pattern, listLast)
	if first < 0 || second < first || last < second {
		head := strings.Join(items[:len(items)-1], ", ")
		return T(ctx, "{0} and {1}", head, items[len(items)-1])
	}
	between := pattern[first+len(listFirst) : second]
	before := pattern[second+len(listSecond) : last]

	return pattern[:first] +
		strings.Join(items[:len(items)-1], between) +
		before + items[len(items)-1] +
		pattern[last+len(listLast):]
}
