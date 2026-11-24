package runtimetools

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolresult"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
)

const minIDBodyRunes = 20

func misprefixed(read, unread []string) string {
	var b strings.Builder
	for _, id := range unread {
		cut := strings.IndexByte(id, '_')
		if cut < 0 || len(id)-cut-1 < minIDBodyRunes {
			continue
		}
		wanted, body := id[:cut+1], id[cut+1:]
		actual, found := actualPrefix(read, wanted, body)
		if !found {
			continue
		}
		actualKind, knownActual := recordkind.ResourceOfIDPrefix(actual)
		wantedKind, knownWanted := recordkind.ResourceOfIDPrefix(wanted)
		if !knownActual || !knownWanted {
			continue
		}
		fmt.Fprintf(&b, " %s is %s's id written as %s's", id, withArticle(actualKind.Noun()),
			withArticle(wantedKind.Noun()))
		field, sibling, carried := siblingID(read, actual+body, wanted)
		if !carried {
			b.WriteString(".")
			continue
		}
		fmt.Fprintf(&b, ": that %s's %s is %s. Send the %s.", actualKind.Noun(),
			wantedKind.Noun(), sibling, field)
	}

	return b.String()
}

func withArticle(noun string) string {
	if noun != "" && strings.ContainsRune("aeiou", rune(noun[0])) {
		return "an " + noun
	}

	return "a " + noun
}

func actualPrefix(read []string, wanted, body string) (string, bool) {
	pattern := regexp.MustCompile(`\b([a-z][a-z0-9]*_)` + regexp.QuoteMeta(body) + `\b`)
	for _, text := range read {
		for _, match := range pattern.FindAllStringSubmatch(text, -1) {
			if match[1] != wanted {
				return match[1], true
			}
		}
	}

	return "", false
}

func siblingID(read []string, actualID, wanted string) (field, id string, found bool) {
	ids := make(map[string]string, 1)
	for _, text := range read {
		for _, object := range toolresult.ObjectsHolding(text, actualID) {
			for _, key := range slices.Sorted(maps.Keys(object)) {
				value, isText := object[key].(string)
				if !isText || !strings.HasPrefix(value, wanted) ||
					len(value)-len(wanted) < minIDBodyRunes {
					continue
				}
				if _, seen := ids[value]; !seen {
					ids[value] = key
				}
			}
		}
	}
	if len(ids) != 1 {
		return "", "", false
	}
	for value, key := range ids {
		return key, value, true
	}

	return "", "", false
}

func misprefixRepairs(read, unread []string) (map[string]string, bool) {
	repairs := make(map[string]string, len(unread))
	for _, id := range unread {
		cut := strings.IndexByte(id, '_')
		if cut < 0 || len(id)-cut-1 < minIDBodyRunes {
			return nil, false
		}
		wanted, body := id[:cut+1], id[cut+1:]
		actual, found := actualPrefix(read, wanted, body)
		if !found {
			return nil, false
		}
		_, sibling, carried := siblingID(read, actual+body, wanted)
		if !carried {
			return nil, false
		}
		repairs[id] = sibling
	}

	return repairs, len(repairs) > 0
}
