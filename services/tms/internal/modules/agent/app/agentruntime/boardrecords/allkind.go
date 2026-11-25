package boardrecords

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	allMark      = "{all}"
	allNounWords = 4
	itemsKey     = "items"
)

var nounPhraseEnd = []string{
	"are", "is", "was", "were", "be", "been", "have", "has", "had", "will", "would", "can",
	"could", "may", "might", "need", "needs", "for", "to", "in", "on", "at", "with", "from",
	"by", "that", "which", "who", "whose", "and", "or", "but", "than", "per", "each",
}

type allNoun struct {
	word  string
	kinds []recordkind.Kind
}

func (r Read) AllNamed(
	tool, words string,
	held recordkind.Kind,
	ids []pulid.ID,
) (whole WholeList, elsewhere bool, refusal string) {
	named, mixed := allNouns(words)
	if mixed != "" {
		return WholeList{}, false, fmt.Sprintf("Tool %q was not run: {all} is one list's "+
			"number, but the words count both %s with it. Keep {all} for one of them and "+
			"write the other count you read as a number.", tool, mixed)
	}
	if named.word == "" || slices.Contains(named.kinds, held) {
		return WholeList{}, false, ""
	}
	lists := r.listsOf(named.kinds)
	switch {
	case len(lists) == 1 && lists[0].Table.IsNotNil():
		return lists[0], true, ""
	case len(lists) == 1:
		refusal = fmt.Sprintf("Write the count of %s you read (%d), or name the %s in a group.",
			named.word, lists[0].Total, named.word)
	case len(lists) > 1:
		totals := make([]string, 0, len(lists))
		for _, list := range lists {
			totals = append(totals, strconv.Itoa(list.Total))
		}
		refusal = fmt.Sprintf("This conversation read %d lists of them (%s): write the "+
			"count you mean as a number, or name the %s in a group.", len(lists),
			strings.Join(totals, ", "), named.word)
	default:
		refusal = fmt.Sprintf("This conversation read no list of %s that reported its "+
			"total: read it first, or name the %s in a group.", named.word, named.word)
	}

	return WholeList{}, false, fmt.Sprintf("Tool %q was not run: {all} reads the whole list "+
		"of the records your groups hold, which are %s (%s), but the words count %s. %s",
		tool, inflect.PluralNoun(held.Noun()), r.heldCount(ids), named.word, refusal)
}

func (r Read) heldCount(ids []pulid.ID) string {
	if total, longer := r.Longer(ids); longer && total > 0 {
		return strconv.Itoa(total) + " read"
	}

	return strconv.Itoa(len(ids)) + " shown"
}

func (r Read) listsOf(kinds []recordkind.Kind) []WholeList {
	lists := make([]WholeList, 0, 1)
	for _, page := range r.pages {
		if page.total <= 0 || !slices.Contains(kinds, page.kind) {
			continue
		}
		found := WholeList{Table: page.table, Total: page.total}
		at := slices.IndexFunc(lists, func(list WholeList) bool { return list.Total == page.total })
		switch {
		case at < 0:
			lists = append(lists, found)
		case page.table.IsNotNil():
			lists[at] = found
		}
	}

	return lists
}

func allNouns(text string) (named allNoun, mixed string) {
	for offset := 0; ; {
		idx := strings.Index(text[offset:], allMark)
		if idx < 0 {
			return named, ""
		}
		offset += idx + len(allMark)
		found := nounAfter(text[offset:])
		if found.word == "" {
			continue
		}
		if named.word == "" {
			named = found
			continue
		}
		shared := slices.DeleteFunc(slices.Clone(named.kinds), func(kind recordkind.Kind) bool {
			return !slices.Contains(found.kinds, kind)
		})
		if len(shared) == 0 {
			return allNoun{}, named.word + " and " + found.word
		}
		named.kinds = shared
	}
}

func nounAfter(text string) allNoun {
	words := wordsOf(clauseAfter(text))
	named := allNoun{}
	for _, word := range words[:min(len(words), allNounWords)] {
		word = strings.Trim(word, "'")
		if slices.Contains(nounPhraseEnd, word) {
			break
		}
		kinds := kindsCalled(word)
		if len(kinds) == 0 {
			if named.word != "" {
				break
			}
			continue
		}
		named = allNoun{word: word, kinds: kinds}
	}

	return named
}

func kindsCalled(word string) []recordkind.Kind {
	var own, called []recordkind.Kind
	same := func(noun string) bool { return strings.EqualFold(noun, word) }
	for _, spec := range boardregistry.Catalog().Kinds() {
		if slices.ContainsFunc(boardtyped.NounsOf(spec.Kind), same) {
			own = append(own, spec.Kind)
			continue
		}
		entity, known := agentcapability.EntityNamed(string(spec.Kind))
		if known && slices.ContainsFunc(entity.Synonyms, same) {
			called = append(called, spec.Kind)
		}
	}
	if len(own) > 0 {
		return own
	}

	return called
}

func pageKind(payload any, ids map[string]bool) recordkind.Kind {
	fields, _ := payload.(map[string]any)
	rows, _ := fields[itemsKey].([]any)
	listed := make([]string, 0, len(rows))
	for _, row := range rows {
		record, _ := row.(map[string]any)
		if id, _ := record[idKey].(string); pulid.LooksLike(id) {
			listed = append(listed, id)
		}
	}
	if len(listed) == 0 {
		for id := range ids {
			listed = append(listed, id)
		}
	}
	kind := recordkind.Kind("")
	for _, id := range listed {
		of, known := recordkind.OfIDPrefix(pulid.ID(id).Prefix())
		if !known || (kind != "" && of != kind) {
			return ""
		}
		kind = of
	}

	return kind
}
