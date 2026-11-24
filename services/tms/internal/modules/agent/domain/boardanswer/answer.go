package boardanswer

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/agnivade/levenshtein"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardhint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
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
	NameAll     = "all"
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
	fieldMetric  = "metric"
	fieldFor     = "for"
	fieldPeriod  = "period"
)

var (
	timedTotal = regexp.MustCompile(
		`(?i)\{total(?:_\d)?\}\s+(?:behind|late|ago|past|overdue|old|left|lapsed)\b`,
	)
	nameMark = regexp.MustCompile(`\{([^{}\s]{1,60})\}`)
	emphasis = strings.NewReplacer("**", "", "__", "", "`", "")
)

type Answer struct {
	Board board.Board
	Words string
}

type group struct {
	title  string
	note   string
	set    *board.RecordSet
	metric *boardcatalog.Metric
	forID  pulid.ID
	period string
}

type translator struct {
	problems []string
	kind     recordkind.Kind
	units    []string
	wholes   map[pulid.ID]pulid.ID
}

func (t *translator) addf(format string, args ...any) {
	t.problems = append(t.problems, fmt.Sprintf(format, args...))
}

func Translate(arguments map[string]any) (answered *Answer, problems []string) {
	t := &translator{wholes: boardtyped.PartsOf(arguments[boardtyped.PartOfField])}
	title := t.text(arguments, fieldTitle, fieldTitle, board.MaxTitleRunes, false)
	answer := t.text(arguments, fieldAnswer, fieldAnswer, MaxAnswerRunes, true)
	groups := t.groups(arguments[fieldGroups])
	title = t.titled(title, groups)
	measure, dropped := t.measure(arguments)
	actions := t.actions(arguments[fieldActions])

	t.noNames(fieldTitle, title)
	for idx := range groups {
		t.noNames(fmt.Sprintf("groups[%d].title", idx+1), groups[idx].title)
	}
	t.typedWholes(&answer, groups)
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
	t.noRecordFigures(used, groups)
	t.wholesKnown(used, groups)
	t.partOf(used, groups, notesAnd(answer, groups)...)
	t.oneGroupNumbered(used, groups)
	measure = t.measured(used, measure, dropped, words(title, answer, groups))
	if measure == "" && len(t.problems) == 0 {
		if spec, ok := boardregistry.Catalog().DefaultMeasure(t.kind); ok {
			measure = board.Measure(spec.Name)
		}
	}
	used = recordMeasures(used, measure)
	used = t.said(used, measure, notesAnd(answer, groups)...)
	if stale := t.staleDays(&answer, groups, measure); stale != nil {
		used = append(used, *stale)
	}
	t.summable(used, groups, measure)
	t.ownFigures(used, groups, measure)
	t.timeWords(used, groups, measure, notesAnd(answer, groups)...)
	for _, typed := range t.units {
		mark := typed[:strings.IndexByte(typed, '}')+1]
		answer = strings.ReplaceAll(answer, typed, mark)
		for idx := range groups {
			groups[idx].note = strings.ReplaceAll(groups[idx].note, typed, mark)
		}
	}
	if len(t.problems) > 0 {
		return nil, t.problems
	}
	readings, keys := t.readings(used, groups, measure)
	if len(readings) > board.MaxReadings {
		t.addf("the answer and notes place %d figures; at most %d fit, so send it again "+
			"placing only the ones the question asks about: the board draws every other "+
			"figure and point%s", len(readings), board.MaxReadings, leadsOf(groups))
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
		if groups[idx].metric != nil {
			out = append(out, board.Block{
				Type:    board.BlockMetric,
				Title:   groups[idx].title,
				Half:    len(groups) == sideBySide,
				Metric:  groups[idx].metric.Name,
				For:     groups[idx].forID,
				Period:  groups[idx].period,
				Caption: groups[idx].note,
			})
			continue
		}
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
		t.addf("groups needs 1 to %d groups, each with the records it shows or a metric",
			MaxGroups)
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
			t.addf("%s is an object with a title, its records or a metric, and an optional note",
				at)
			continue
		}
		listed, _ := fields[fieldRecords].([]any)
		metric := strings.TrimSpace(stringOf(fields[fieldMetric]))
		if idx > 0 && len(listed) == 0 && metric == "" {
			continue
		}
		title := stringutils.WholeWordsWithin(
			t.text(fields, fieldTitle, at+".title", math.MaxInt32, false),
			board.MaxBlockTitleRunes,
		)
		note := t.note(fields, at+".note")
		if len(listed) > 0 || metric == "" {
			groups = append(groups, group{
				title: title,
				note:  note,
				set:   t.records(fields[fieldRecords], at+".records"),
			})
		}
		if metric != "" {
			grouped := t.metricGroup(fields, at, metric)
			grouped.title = title
			if len(listed) == 0 {
				grouped.note = note
			}
			groups = append(groups, grouped)
		}
	}
	if len(groups) > MaxGroups {
		t.addf("groups holds at most %d groups, and a group with both records and a metric "+
			"counts as two; give the metric its own group", MaxGroups)
		return nil
	}

	return groups
}

func (t *translator) metricGroup(fields map[string]any, at, name string) group {
	metric, known := boardregistry.Catalog().MetricNamed(name)
	if !known {
		t.addf("%s.metric %q is not a metric Trenova keeps; the closest are %s", at, name,
			stringutils.JoinAlternatives(closestMetrics(name)))
		return group{}
	}
	out := group{metric: &metric, period: string(metric.DefaultPeriod())}
	period := strings.TrimSpace(stringOf(fields[fieldPeriod]))
	for _, taken := range metric.Periods {
		if strings.EqualFold(string(taken), period) {
			out.period = string(taken)
		}
	}
	if metric.For == "" {
		return out
	}
	text := strings.TrimSpace(stringOf(fields[fieldFor]))
	if text == "" {
		if metric.ForNeeded {
			t.addf("%s: the %s metric reads one %s; give for, the id a tool returned for it",
				at, metric.Name, metric.For.Noun())
		}
		return out
	}
	id, err := pulid.Parse(text)
	kind, knownKind := recordkind.OfIDPrefix(id.Prefix())
	if err != nil || id.IsNil() || !knownKind || kind != metric.For {
		if metric.ForNeeded {
			t.addf("%s.for holds %q; the %s metric reads one %s, by the id a tool returned "+
				"for it", at, text, metric.Name, metric.For.Noun())
		}
		return out
	}
	out.forID = id

	return out
}

func closestMetrics(name string) []string {
	names := boardregistry.Catalog().MetricNames()
	lowered := strings.ToLower(name)
	slices.SortStableFunc(names, func(a, b string) int {
		return levenshtein.ComputeDistance(lowered, strings.ToLower(a)) -
			levenshtein.ComputeDistance(lowered, strings.ToLower(b))
	})

	return names[:min(len(names), closestNamed)]
}

const closestNamed = 3

func stringOf(value any) string {
	text, _ := value.(string)

	return text
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
		kind, known := kindOf(text)
		if !known {
			t.addf(
				"%s holds %s, which is not a %s%s",
				path,
				text,
				joinKinds(),
				boardhint.Subject(id),
			)
			return nil
		}
		if t.kind == "" {
			t.kind = kind
		}
		if kind != t.kind {
			t.addf("%s mixes %s and %s; an answer shows one kind of record",
				path, inflect.PluralNoun(t.kind.Noun()), inflect.PluralNoun(kind.Noun()))
			return nil
		}
		set.Kind = kind
		if !slices.Contains(set.IDs, id) {
			set.IDs = append(set.IDs, id)
		}
	}

	return set
}

func (t *translator) measure(
	arguments map[string]any,
) (measure, dropped board.Measure) {
	text, _ := arguments[fieldMeasure].(string)
	asked := board.Measure(strings.TrimSpace(text))
	if asked == "" || t.kind == "" {
		return "", asked
	}
	if spec, named := boardregistry.Catalog().MeasureOf(t.kind, string(asked)); named {
		return board.Measure(spec.Name), ""
	}

	return "", asked
}

func (t *translator) measured(
	used []figure,
	measure, dropped board.Measure,
	text string,
) board.Measure {
	if measure != "" || t.kind == "" {
		return measure
	}
	if summed := t.soleSummable(used); dropped == "" && summed != "" &&
		!timedTotal.MatchString(text) {
		return summed
	}
	for _, found := range used {
		if found.counts() || found.measure != "" || found.metric != noMetric {
			continue
		}
		measures := boardregistry.Catalog().MeasuresOf(t.kind)
		switch {
		case dropped != "":
			t.addf("{%s} adds up the measure, and a %s has no measure %q; use one of %s, "+
				"or write {count}", found.raw, t.kind.Noun(), dropped, joinMeasures(t.kind))
		case len(measures) == 1:
			return board.Measure(measures[0].Name)
		case len(measures) > 1 && hasDefault(t.kind):
			return ""
		case len(measures) > 1:
			t.addf("{%s} adds up the measure, so set measure to one of %s, or name it in the "+
				"figure, as {%s:%s}", found.raw, joinMeasures(t.kind), found.name,
				exampleMeasure(measures, found.base))
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
		kind, known := kindOf(text)
		if err != nil || !known || (set.Kind != "" && set.Kind != kind) ||
			len(set.IDs) == board.MaxActionRecords {
			continue
		}
		set.Kind = kind
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
	names := []string{NameCount, NameAll, NameTotal, NameLargest}
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

func splitName(name string) (base string, number int) {
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

func union(groups []group) *board.RecordSet {
	set := &board.RecordSet{}
	for idx := range groups {
		if groups[idx].set == nil {
			continue
		}
		set.Kind = groups[idx].set.Kind
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

func (t *translator) note(fields map[string]any, path string) string {
	text := t.text(fields, fieldNote, path, math.MaxInt32, false)
	if utf8.RuneCountInString(text) <= board.MaxCaptionRunes {
		return text
	}
	fitted := stringutils.WholeSentences(text, board.MaxCaptionRunes)
	if fitted == "" {
		t.addf("%s is longer than %d characters", path, board.MaxCaptionRunes)
	}

	return fitted
}

func (t *translator) soleSummable(used []figure) board.Measure {
	totals := slices.ContainsFunc(used, func(found figure) bool {
		return found.base == NameTotal && found.measure == "" && found.metric == noMetric &&
			found.record.IsNil()
	})
	if !totals {
		return ""
	}
	summable := slices.DeleteFunc(
		boardregistry.Catalog().MeasuresOf(t.kind),
		func(spec boardcatalog.Measure) bool {
			return !spec.Summable
		},
	)
	if len(summable) != 1 {
		return ""
	}

	return board.Measure(summable[0].Name)
}

func exampleMeasure(measures []boardcatalog.Measure, base string) string {
	for idx := range measures {
		if base != NameTotal || measures[idx].Summable {
			return measures[idx].Name
		}
	}

	return measures[0].Name
}

func (t *translator) titled(title string, groups []group) string {
	if title != "" {
		return title
	}
	for idx := range groups {
		if groups[idx].title != "" {
			return groups[idx].title
		}
	}
	t.addf("%s is required", fieldTitle)

	return ""
}

func hasDefault(kind recordkind.Kind) bool {
	_, ok := boardregistry.Catalog().DefaultMeasure(kind)

	return ok
}

func recordMeasures(used []figure, measure board.Measure) []figure {
	for idx := range used {
		if used[idx].record.IsNotNil() && used[idx].measure == "" {
			used[idx].measure = measure
		}
	}

	return used
}

func kindOf(id string) (recordkind.Kind, bool) {
	cut := strings.IndexByte(id, '_')
	if cut < 0 {
		return "", false
	}
	kind, known := recordkind.OfIDPrefix(id[:cut+1])
	if !known || !boardregistry.Catalog().Supports(kind) {
		return "", false
	}

	return kind, true
}

func joinKinds() string {
	kinds := boardregistry.Catalog().Kinds()
	names := make([]string, 0, len(kinds))
	for idx := range kinds {
		names = append(names, kinds[idx].Kind.Noun())
	}

	return stringutils.JoinAlternatives(names)
}

func joinMeasures(kind recordkind.Kind) string {
	measures := boardregistry.Catalog().MeasuresOf(kind)
	names := make([]string, 0, len(measures))
	for idx := range measures {
		names = append(names, measures[idx].Name)
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

func notesAnd(answer string, groups []group) []string {
	texts := []string{answer}
	for idx := range groups {
		texts = append(texts, groups[idx].note)
	}

	return texts
}
