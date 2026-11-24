package boardtyped

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const dayCount = `(?:\d{1,4}|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve)` +
	`\s+(?:days?|weeks?)`

var daySpans = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bin\s+(` + dayCount + `)\b`),
	regexp.MustCompile(`(?i)\b(` + dayCount +
		`)\s+(?:ago|left|remaining|lapsed|overdue|late|away|out)\b`),
	regexp.MustCompile(`(?i)\((` + dayCount + `)\)`),
}

var (
	typedUnit = regexp.MustCompile(`^\s+(?i:minutes?|mins?|hours?|hrs?|days?)\b`)
	moneyWord = regexp.MustCompile(`(?i)\b(?:billed|billable|billing|charged|invoiced|` +
		`revenue|owed|paid|earned)\b|\$`)
	dayWord   = regexp.MustCompile(`(?i)\b(since|from|dated|on|until|as of)\s*$`)
	clauseEnd = regexp.MustCompile(`[.!?](?:\s|$)|\n|[,;:]|\s(?i:and|but|while|against|` +
		`whereas|versus|than)\s`)
)

const PartOfField = "partOf"

const countWords = `one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|\d{1,4}`

func Unit(after string) string {
	return typedUnit.FindString(after)
}

func MoneyWord(clause string) string {
	return moneyWord.FindString(clause)
}

func DayWord(before string) string {
	match := dayWord.FindStringSubmatch(before)
	if match == nil {
		return ""
	}

	return match[1]
}

func Clause(text string, at, past int) string {
	start := 0
	for _, loc := range clauseEnd.FindAllStringIndex(text[:at], -1) {
		start = loc[1]
	}
	end := len(text)
	if loc := clauseEnd.FindStringIndex(text[past:]); loc != nil {
		end = past + loc[0]
	}

	return text[start:end]
}

func Days(text string) []string {
	found := make([]string, 0, 1)
	for _, pattern := range daySpans {
		for _, match := range pattern.FindAllStringSubmatch(text, -1) {
			if !slices.Contains(found, match[1]) {
				found = append(found, match[1])
			}
		}
	}

	return found
}

func MarkDays(text, mark string) string {
	for _, pattern := range daySpans {
		text = pattern.ReplaceAllStringFunc(text, func(match string) string {
			span := pattern.FindStringSubmatch(match)[1]
			return strings.Replace(match, span, mark, 1)
		})
	}

	return text
}

func Quoted(texts []string) string {
	quoted := make([]string, 0, len(texts))
	for _, text := range texts {
		quoted = append(quoted, fmt.Sprintf("%q", text))
	}

	return strings.Join(quoted, ", ")
}

type Whole struct {
	Kind  recordkind.Kind
	Nouns []string
}

func WholeOf(part recordkind.Kind) (Whole, bool) {
	spec, ok := boardregistry.Catalog().KindOf(part)
	if !ok || spec.PartOf == "" {
		return Whole{}, false
	}

	return Whole{Kind: spec.PartOf, Nouns: NounsOf(spec.PartOf)}, true
}

func NounsOf(kind recordkind.Kind) []string {
	spec, known := boardregistry.Catalog().KindOf(kind)
	if !known {
		spec = boardcatalog.Kind{Kind: kind}
	}
	nouns := spec.Nouns()
	last := kind.Noun()
	if cut := strings.LastIndexByte(last, ' '); cut >= 0 {
		last = last[cut+1:]
	}
	for _, word := range []string{last, inflect.PluralNoun(last)} {
		if !slices.Contains(nouns, word) {
			nouns = append(nouns, word)
		}
	}

	return nouns
}

func (w *Whole) Names(word string) bool {
	return slices.ContainsFunc(w.Nouns, func(noun string) bool {
		return strings.EqualFold(noun, strings.TrimSpace(word))
	})
}

func (w *Whole) Place(
	text string,
	part recordkind.Kind,
	total int,
) (placed string, wrong []string) {
	nouns := make([]string, 0, len(w.Nouns))
	for _, noun := range w.Nouns {
		nouns = append(nouns, regexp.QuoteMeta(noun))
	}
	pattern := regexp.MustCompile(`(?i)\b(` + countWords + `)\s+(?:(?:more|other|different|` +
		`separate|distinct)\s+)?(?:` + strings.Join(nouns, "|") + `)\b`)
	own := Whole{Kind: part, Nouns: NounsOf(part)}
	var out strings.Builder
	wrong, last := make([]string, 0, 1), 0
	for _, loc := range pattern.FindAllStringSubmatchIndex(text, -1) {
		next := strings.Fields(text[loc[1]:])
		if len(next) > 0 && own.Names(strings.Trim(next[0], ".,;:!?")) {
			continue
		}
		if numberOf(text[loc[2]:loc[3]]) != total {
			wrong = append(wrong, text[loc[0]:loc[1]])
			continue
		}
		out.WriteString(text[last:loc[2]] + "{count:" + inflect.PluralNoun(w.Kind.Noun()) + "}")
		last = loc[3]
	}
	out.WriteString(text[last:])

	return out.String(), wrong
}

func (w *Whole) Counted(
	texts []*string,
	part recordkind.Kind,
	sets [][]pulid.ID,
	wholes map[pulid.ID]pulid.ID,
) string {
	if w.Kind == "" {
		return ""
	}
	plural := inflect.PluralNoun(w.Kind.Noun())
	all, counts := make([]pulid.ID, 0, 8), make([]string, 0, len(sets))
	for idx, set := range sets {
		all = append(all, set...)
		if ids, known := w.Of(set, wholes); known && len(set) > 0 {
			counts = append(counts, fmt.Sprintf("%d in group %d, {count_%d:%s}", len(ids), idx+1,
				idx+1, plural))
		}
	}
	total := -1
	if ids, known := w.Of(all, wholes); known {
		total = len(ids)
	}
	wrong := make([]string, 0, 1)
	for _, text := range texts {
		placed, typed := w.Place(*text, part, total)
		*text, wrong = placed, append(wrong, typed...)
	}
	if len(wrong) == 0 {
		return ""
	}

	if total < 0 {
		return fmt.Sprintf("the words type %s, a count of the %s these %s belong to; write "+
			"{count:%s} where it goes, never typed", Quoted(wrong), plural,
			inflect.PluralNoun(part.Noun()), plural)
	}

	return fmt.Sprintf("the words type %s, a count of the %s these %s belong to; they belong "+
		"to %d in all, {count:%s} (%s): place the one you mean, never typed", Quoted(wrong),
		plural, inflect.PluralNoun(part.Noun()), total, plural, strings.Join(counts, "; "))
}

func numberOf(word string) int {
	if number, err := strconv.Atoi(word); err == nil {
		return number
	}

	return slices.Index(strings.Split(countWords, "|"), strings.ToLower(word)) + 1
}

func (w *Whole) Reading(
	key, label string,
	parts []pulid.ID,
	wholes map[pulid.ID]pulid.ID,
) board.Reading {
	ids, _ := w.Of(parts, wholes)

	return board.Reading{Key: key, FigureSource: board.FigureSource{
		Label:     stringutils.TruncateRunes(label, board.MaxLabelRunes),
		Records:   &board.RecordSet{Kind: w.Kind, IDs: ids},
		Aggregate: board.AggregateCount,
	}}
}

func (w *Whole) Of(parts []pulid.ID, wholes map[pulid.ID]pulid.ID) ([]pulid.ID, bool) {
	ids := make([]pulid.ID, 0, len(parts))
	for _, part := range parts {
		whole, known := wholes[part]
		if !known {
			return nil, false
		}
		if !slices.Contains(ids, whole) {
			ids = append(ids, whole)
		}
	}

	return ids, true
}

func (w *Whole) Key(name string) string {
	return name + "_" + strings.ReplaceAll(w.Kind.Noun(), " ", "_")
}

func PartsOf(raw any) map[pulid.ID]pulid.ID {
	fields, _ := raw.(map[string]any)
	wholes := make(map[pulid.ID]pulid.ID, len(fields))
	for part, value := range fields {
		whole, _ := value.(string)
		if pulid.LooksLike(part) && pulid.LooksLike(whole) {
			wholes[pulid.ID(part)] = pulid.ID(whole)
		}
	}

	return wholes
}
