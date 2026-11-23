package boardanswer

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxAnswerRunes = 600
	MaxGroups      = 3
	sideBySide     = 2
	MaxActions     = 2
)

const (
	NameCount   = "count"
	NameTotal   = "total"
	NameLargest = "largest"
)

const (
	fieldTitle   = "title"
	fieldAnswer  = "answer"
	fieldGroups  = "groups"
	fieldRecords = "records"
	fieldNote    = "note"
	fieldMeasure = "measure"
	fieldActions = "actions"
)

var (
	nameMark = regexp.MustCompile(`\{([^{}\s]{1,60})\}`)
	emphasis = strings.NewReplacer("**", "", "__", "", "`", "")
)

type Answer struct {
	Board board.Board
	Words string
}

type group struct {
	title string
	note  string
	set   *board.RecordSet
}

type translator struct {
	problems []string
	kind     permission.Resource
}

func (t *translator) addf(format string, args ...any) {
	t.problems = append(t.problems, fmt.Sprintf(format, args...))
}

func Translate(arguments map[string]any) (*Answer, []string) {
	t := &translator{}
	title := t.text(arguments, fieldTitle, fieldTitle, board.MaxTitleRunes, true)
	answer := t.text(arguments, fieldAnswer, fieldAnswer, MaxAnswerRunes, true)
	groups := t.groups(arguments[fieldGroups])
	measure, dropped := t.measure(arguments)
	actions := t.actions(arguments[fieldActions])

	t.noNames(fieldTitle, title)
	for idx := range groups {
		t.noNames(fmt.Sprintf("groups[%d].title", idx+1), groups[idx].title)
	}
	used := t.figures(fieldAnswer, answer, groups)
	for idx := range groups {
		for _, found := range t.figures(fmt.Sprintf("groups[%d].note", idx+1), groups[idx].note,
			groups) {
			if !slices.ContainsFunc(
				used,
				func(other figure) bool { return other.raw == found.raw },
			) {
				used = append(used, found)
			}
		}
	}
	measure = t.measured(used, measure, dropped)
	t.summable(used, groups, measure)
	t.ownFigures(used, groups, measure)
	if len(t.problems) > 0 {
		return nil, t.problems
	}
	readings, keys := t.readings(used, groups, measure)
	if len(readings) > board.MaxReadings {
		t.addf("the answer and notes place %d figures; at most %d fit", len(readings),
			board.MaxReadings)
		return nil, t.problems
	}
	answer = keyed(answer, keys)
	for idx := range groups {
		groups[idx].note = keyed(groups[idx].note, keys)
	}

	layout := board.Board{
		Title:    title,
		Verdict:  answer,
		Place:    board.PlaceConversation,
		Readings: readings,
		Blocks:   blocks(groups, measure, actions),
	}

	return &Answer{Board: layout, Words: words(title, answer, groups)}, nil
}

func words(title, answer string, groups []group) string {
	texts := []string{title, answer}
	for idx := range groups {
		texts = append(texts, groups[idx].title, groups[idx].note)
	}
	texts = slices.DeleteFunc(texts, func(text string) bool { return text == "" })

	return strings.Join(texts, "\n\n")
}

func blocks(groups []group, measure board.Measure, actions []board.Action) []board.Block {
	order := board.OrderGiven
	if measure != "" {
		order = board.OrderMeasure
	}
	out := make([]board.Block, 0, len(groups)+1)
	for idx := range groups {
		out = append(out, board.Block{
			Type:    board.BlockRecords,
			Title:   groups[idx].title,
			Half:    len(groups) == sideBySide,
			Records: groups[idx].set,
			Measure: measure,
			Order:   order,
			Caption: groups[idx].note,
		})
	}
	if len(actions) > 0 {
		out = append(out, board.Block{Type: board.BlockActions, Actions: actions})
	}

	return out
}

func (t *translator) groups(raw any) []group {
	entries, isList := raw.([]any)
	switch {
	case !isList || len(entries) == 0:
		t.addf("groups needs 1 or 2 groups, each with the records it shows")
		return nil
	case len(entries) > MaxGroups:
		t.addf("groups holds at most %d groups: two are drawn side by side, three one under "+
			"another", MaxGroups)
		return nil
	}
	groups := make([]group, 0, len(entries))
	for idx, entry := range entries {
		at := fmt.Sprintf("groups[%d]", idx+1)
		fields, isObject := entry.(map[string]any)
		if !isObject {
			if idx > 0 && entry == nil {
				continue
			}
			t.addf("%s is an object with a title, its records and an optional note", at)
			continue
		}
		listed, _ := fields[fieldRecords].([]any)
		if idx > 0 && len(listed) == 0 {
			continue
		}
		groups = append(groups, group{
			title: t.text(fields, fieldTitle, at+".title", board.MaxBlockTitleRunes, false),
			note:  t.text(fields, fieldNote, at+".note", board.MaxCaptionRunes, false),
			set:   t.records(fields[fieldRecords], at+".records"),
		})
	}

	return groups
}

func (t *translator) records(raw any, path string) *board.RecordSet {
	entries, _ := raw.([]any)
	if len(entries) == 0 || len(entries) > board.MaxListRecords {
		t.addf("%s needs 1 to %d record ids a tool in this conversation returned",
			path, board.MaxListRecords)
		return nil
	}
	set := &board.RecordSet{IDs: make([]pulid.ID, 0, len(entries))}
	for _, entry := range entries {
		text, _ := entry.(string)
		text = strings.TrimSpace(text)
		id, err := pulid.Parse(text)
		if err != nil || id.IsNil() {
			t.addf("%s holds %q, which is not a record id; a record id comes from a tool "+
				"result, such as shp_01J…", path, text)
			return nil
		}
		resource, known := resourceOf(text)
		if !known {
			t.addf("%s holds %s, which is not a %s", path, text, joinResources())
			return nil
		}
		if t.kind == "" {
			t.kind = resource
		}
		if resource != t.kind {
			t.addf("%s mixes %s and %s; an answer shows one kind of record",
				path, inflect.PluralNoun(t.kind.Noun()), inflect.PluralNoun(resource.Noun()))
			return nil
		}
		set.Resource = resource
		if !slices.Contains(set.IDs, id) {
			set.IDs = append(set.IDs, id)
		}
	}

	return set
}

func (t *translator) measure(arguments map[string]any) (board.Measure, board.Measure) {
	text, _ := arguments[fieldMeasure].(string)
	measure := board.Measure(strings.TrimSpace(text))
	if measure == "" || t.kind == "" {
		return "", measure
	}
	if named := board.MeasureNamed(t.kind, string(measure)); named != "" {
		return named, ""
	}

	return "", measure
}

func (t *translator) measured(used []figure, measure, dropped board.Measure) board.Measure {
	if measure != "" || t.kind == "" {
		return measure
	}
	for _, found := range used {
		if found.base == NameCount || found.measure != "" {
			continue
		}
		measures := board.MeasuresOf(t.kind)
		switch {
		case dropped != "":
			t.addf("{%s} adds up the measure, and a %s has no measure %q; use one of %s, "+
				"or write {count}", found.raw, t.kind.Noun(), dropped, joinMeasures(t.kind))
		case len(measures) == 1:
			return measures[0]
		case len(measures) > 1:
			t.addf("{%s} adds up the measure, so set measure to one of %s, or name it in the "+
				"figure, as {%s:%s}", found.raw, joinMeasures(t.kind), found.name, measures[0])
		default:
			t.addf("{%s} adds up the measure, and a %s has none; write {count} instead",
				found.raw, t.kind.Noun())
		}

		return ""
	}
	for _, found := range used {
		if found.measure != "" {
			return found.measure
		}
	}

	return ""
}

func (t *translator) actions(raw any) []board.Action {
	entries, _ := raw.([]any)
	actions := make([]board.Action, 0, MaxActions)
	for idx, entry := range entries {
		if len(actions) == MaxActions {
			break
		}
		fields, isObject := entry.(map[string]any)
		if !isObject {
			continue
		}
		at := fmt.Sprintf("actions[%d]", idx+1)
		action := board.Action{
			Label: t.text(fields, "label", at+".label", board.MaxActionLabelRunes, false),
			Ask:   t.text(fields, "ask", at+".ask", board.MaxActionAskRunes, false),
		}
		if action.Label == "" || action.Ask == "" {
			continue
		}
		if listed, _ := fields[fieldRecords].([]any); len(listed) > 0 {
			action.Records = actionRecords(listed)
		}
		actions = append(actions, action)
	}

	return actions
}

func actionRecords(listed []any) *board.RecordSet {
	set := &board.RecordSet{IDs: make([]pulid.ID, 0, len(listed))}
	for _, entry := range listed {
		text, _ := entry.(string)
		text = strings.TrimSpace(text)
		id, err := pulid.Parse(text)
		resource, known := resourceOf(text)
		if err != nil || !known || (set.Resource != "" && set.Resource != resource) ||
			len(set.IDs) == board.MaxActionRecords {
			continue
		}
		set.Resource = resource
		if !slices.Contains(set.IDs, id) {
			set.IDs = append(set.IDs, id)
		}
	}
	if len(set.IDs) == 0 {
		return nil
	}

	return set
}

func (t *translator) noNames(path, text string) {
	for _, match := range nameMark.FindAllStringSubmatch(text, -1) {
		t.addf("%s holds {%s}; a figure goes in the answer or a group's note", path, match[1])
	}
}

func Names(groups int) []string {
	names := []string{NameCount, NameTotal, NameLargest}
	if groups < sideBySide {
		return names
	}
	for _, base := range []string{NameCount, NameTotal, NameLargest} {
		for number := 1; number <= min(groups, MaxGroups); number++ {
			names = append(names, fmt.Sprintf("%s_%d", base, number))
		}
	}

	return names
}

func Known(name string, groups int) bool {
	if slices.Contains(Names(groups), name) {
		return true
	}
	base, number := splitName(name)

	return groups == 1 && number == 1 && slices.Contains(Names(1), base)
}

func splitName(name string) (string, int) {
	base, suffix, found := strings.Cut(name, "_")
	if !found {
		return name, 0
	}
	number, err := strconv.Atoi(suffix)
	if err != nil || number < 1 || number > MaxGroups || suffix != strconv.Itoa(number) {
		return name, 0
	}

	return base, number
}

var totalLabels = map[board.Measure]string{
	board.MeasureLateness:   "Total time behind",
	board.MeasureRevenue:    "Total revenue",
	board.MeasureAmountDue:  "Total due",
	board.MeasureTotal:      "Total billed",
	board.MeasureAge:        "Total age",
	board.MeasureAmount:     "Total amount",
	board.MeasurePendingFor: "Total time pending",
}

var largestLabels = map[board.Measure]string{
	board.MeasureLateness:   "Furthest behind",
	board.MeasureRevenue:    "Largest revenue",
	board.MeasureAmountDue:  "Largest amount due",
	board.MeasureTotal:      "Largest invoice total",
	board.MeasureAge:        "Oldest",
	board.MeasureAmount:     "Largest amount",
	board.MeasurePendingFor: "Longest pending",
}

func union(groups []group) *board.RecordSet {
	set := &board.RecordSet{}
	for idx := range groups {
		if groups[idx].set == nil {
			continue
		}
		set.Resource = groups[idx].set.Resource
		for _, id := range groups[idx].set.IDs {
			if !slices.Contains(set.IDs, id) {
				set.IDs = append(set.IDs, id)
			}
		}
	}

	return set
}

func (t *translator) text(
	fields map[string]any,
	key, path string,
	most int,
	required bool,
) string {
	value, present := fields[key]
	text, isText := value.(string)
	if present && value != nil && !isText {
		t.addf("%s is text", path)
		return ""
	}
	text = board.PlainText(emphasis.Replace(text))
	text = strings.Join(strings.Fields(text), " ")
	switch {
	case text == "" && required:
		t.addf("%s is required", path)
	case utf8.RuneCountInString(text) > most:
		t.addf("%s is longer than %d characters", path, most)
	}

	return text
}

func resourceOf(id string) (permission.Resource, bool) {
	cut := strings.IndexByte(id, '_')
	if cut < 0 {
		return "", false
	}
	resource, known := recordkind.ResourceOfIDPrefix(id[:cut+1])
	if !known || !board.Supports(resource) {
		return "", false
	}

	return resource, true
}

func capitalized(text string) string {
	first, size := utf8.DecodeRuneInString(text)
	if size == 0 {
		return text
	}

	return string(unicode.ToUpper(first)) + text[size:]
}

func joinResources() string {
	names := make([]string, 0, len(board.Resources()))
	for _, resource := range board.Resources() {
		names = append(names, resource.Noun())
	}

	return stringutils.JoinAlternatives(names)
}

func joinMeasures(kind permission.Resource) string {
	names := make([]string, 0, 3)
	for _, measure := range board.MeasuresOf(kind) {
		names = append(names, string(measure))
	}

	return stringutils.JoinAlternatives(names)
}

func joinNames(groups int) string {
	names := make([]string, 0, 9)
	for _, name := range Names(groups) {
		names = append(names, "{"+name+"}")
	}

	return strings.Join(names, ", ")
}
