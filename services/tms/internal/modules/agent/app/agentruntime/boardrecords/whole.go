package boardrecords

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	keptNote  = "is kept as"
	wordsBack = 3
)

var (
	keptLink = regexp.MustCompile(`\(artifact:(art_[0-9A-Za-z]+)\)`)
	partWord = []string{"these", "shown", "listed", "here", "below"}
)

type WholeList struct {
	Table pulid.ID
	Total int
}

func keptTable(content string) pulid.ID {
	at := strings.LastIndex(content, keptNote)
	if at < 0 {
		return pulid.Nil
	}
	match := keptLink.FindStringSubmatch(content[at:])
	if match == nil {
		return pulid.Nil
	}

	return pulid.ID(match[1])
}

func (r Read) WholeOf(ids []pulid.ID) (WholeList, bool) {
	best, held := WholeList{}, 0
	for _, page := range r.pages {
		in := 0
		for _, id := range ids {
			if page.ids[id.String()] {
				in++
			}
		}
		if in == 0 || in < held || page.table.IsNil() || page.total <= len(ids) {
			continue
		}
		best, held = WholeList{Table: page.table, Total: page.total}, in
	}

	return best, held > 0
}

func PointsAtShown(text, mark string) bool {
	starts := regexp.MustCompile(regexp.QuoteMeta(mark)).FindAllStringIndex(text, -1)
	if len(starts) == 0 {
		return false
	}
	for _, at := range starts {
		before := wordsOf(text[:at[0]])
		before = before[max(0, len(before)-wordsBack):]
		after := wordsOf(clauseAfter(text[at[1]:]))
		if !slices.ContainsFunc(before, isPartWord) && !slices.ContainsFunc(after, isPartWord) {
			return false
		}
	}

	return true
}

func isPartWord(word string) bool {
	return slices.Contains(partWord, word)
}

func clauseAfter(text string) string {
	if end := strings.IndexAny(text, ".,;:!?()\n"); end >= 0 {
		return text[:end]
	}

	return text
}

func wordsOf(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	})
}

func (r Read) SumTiedToWhole(text, mark string, ids []pulid.ID) (int, bool) {
	total, longer := r.Longer(ids)
	if !longer {
		return 0, false
	}
	for offset := 0; ; {
		idx := strings.Index(text[offset:], mark)
		if idx < 0 {
			return total, false
		}
		at := offset + idx
		clause := clauseOf(text, at, at+len(mark))
		if strings.Contains(clause, "{all}") || (total > 0 &&
			(stringutils.ContainsWords(clause, strconv.Itoa(total)) ||
				stringutils.ContainsWords(clause, grouped(total)))) {
			return total, true
		}
		offset = at + len(mark)
	}
}

func clauseOf(text string, at, past int) string {
	start := strings.LastIndexAny(text[:at], ".;:!?,\n") + 1
	end := len(text)
	if idx := strings.IndexAny(text[past:], ".;:!?,\n"); idx >= 0 {
		end = past + idx
	}

	return text[start:end]
}

func AllOf(arguments map[string]any, table pulid.ID) map[string]any {
	listed, _ := rewriteFigures(arguments, func(_ string, _ int, mark string) string {
		if mark == "{all}" {
			return "{all:" + table.String() + "}"
		}
		return mark
	})

	return listed
}
