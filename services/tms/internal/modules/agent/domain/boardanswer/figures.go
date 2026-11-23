package boardanswer

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const measureMark = ":"

type figure struct {
	raw     string
	name    string
	base    string
	number  int
	measure board.Measure
	record  pulid.ID
}

func valueLabel(measure board.Measure) string {
	switch measure {
	case board.MeasureLateness:
		return "Time behind"
	case board.MeasureAge:
		return "Age"
	case board.MeasurePendingFor:
		return "Time pending"
	case board.MeasureRevenue:
		return "Revenue"
	case board.MeasureAmountDue:
		return "Amount due"
	case board.MeasureTotal:
		return "Invoice total"
	case board.MeasureAmount:
		return "Amount"
	}

	return string(measure)
}

func singleLabel(measure board.Measure) (string, bool) {
	return valueLabel(measure), measure.Kind() == board.ValueDuration
}

func (t *translator) figures(path, text string, groups []group) []figure {
	found := make([]figure, 0, 2)
	for _, match := range nameMark.FindAllStringSubmatch(text, -1) {
		raw := match[1]
		if slices.ContainsFunc(found, func(other figure) bool { return other.raw == raw }) {
			continue
		}
		name, named, hasMeasure := strings.Cut(raw, measureMark)
		name, named = strings.TrimSpace(name), strings.TrimSpace(named)
		if own, isOwn := t.ownFigure(path, raw, name, named, groups); isOwn {
			if own != nil {
				found = append(found, *own)
			}
			continue
		}
		if !Known(name, len(groups)) {
			t.addf("%s places {%s}, which is not one of %s; one record's own figure is its "+
				"measure and its number, as {revenue:A-1001}, and any figure may name "+
				"its measure after a colon, as {total:revenue}", path, raw, joinNames(len(groups)))
			continue
		}
		base, number := splitName(name)
		current := figure{raw: raw, name: name, base: base, number: number}
		if hasMeasure && base != NameCount && t.kind != "" {
			current.measure = board.MeasureNamed(t.kind, named)
			if current.measure == "" {
				t.addf("%s places {%s}, and a %s has no measure %q; use one of %s",
					path, raw, t.kind.Noun(), named, joinMeasures(t.kind))
				continue
			}
		}
		found = append(found, current)
	}

	return found
}

func (t *translator) ownFigure(
	path, raw, left, right string,
	groups []group,
) (*figure, bool) {
	if t.kind == "" || right == "" {
		return nil, false
	}
	measure, record := board.MeasureNamed(t.kind, left), right
	if measure == "" {
		measure, record = board.MeasureNamed(t.kind, right), left
	}
	if measure == "" || Known(left, len(groups)) {
		return nil, false
	}
	id, err := pulid.Parse(record)
	if err != nil || id.IsNil() {
		t.addf("%s places {%s}; a record's own figure names a record this answer shows, "+
			"by the number a tool result gave it, as {%s:A-1001}", path, raw, measure)
		return nil, true
	}
	if !slices.Contains(union(groups).IDs, id) {
		t.addf("%s places {%s}, a record no group shows; put it in a group, or name one "+
			"the groups hold", path, raw)
		return nil, true
	}

	return &figure{raw: raw, measure: measure, record: id}, true
}

func (t *translator) ownFigures(used []figure, groups []group, measure board.Measure) {
	if len(groups) < sideBySide {
		return
	}
	marks := make([]string, 0, 2)
	read := measure
	for _, found := range used {
		if found.number == 0 || found.base == NameCount || found.number > len(groups) ||
			groups[found.number-1].set == nil || len(groups[found.number-1].set.IDs) != 1 {
			continue
		}
		marks = append(marks, "{"+found.raw+"}")
		if read == "" {
			read = found.measure
		}
	}
	if len(marks) == 0 {
		return
	}
	if read == "" {
		if measures := board.MeasuresOf(t.kind); len(measures) > 0 {
			read = measures[0]
		}
	}
	what := inflect.Pluralize("is the figure of a group that holds",
		"are the figures of groups that each hold", len(marks))
	t.addf("%s %s one record, and a group's place is easy to mistake for the record a "+
		"sentence names. Name the record instead: its measure and its own number, as "+
		"{%s:A-1001}", strings.Join(marks, ", "), what, read)
}

func (t *translator) summable(used []figure, groups []group, measure board.Measure) {
	for _, found := range used {
		if found.base != NameTotal {
			continue
		}
		read := found.measureOr(measure)
		set := scopeOf(&found, groups)
		if read == "" || read.Summable() || set == nil || len(set.IDs) < sideBySide {
			continue
		}
		t.addf("{%s} would add up %s across %d %s, and %s added across records is not a "+
			"figure anyone reads. For the most, write {largest%s}; for one record's own, write "+
			"its measure and its number, as {%s:A-1001}%s",
			found.raw, read, len(set.IDs), inflect.PluralNoun(t.kind.Noun()),
			strings.ToLower(durationLabel(read)), measureSuffix(read, measure), read,
			t.amountHint())
	}
}

func measureSuffix(read, measure board.Measure) string {
	if read == measure {
		return ""
	}

	return measureMark + string(read)
}

func (t *translator) amountHint() string {
	for _, candidate := range board.MeasuresOf(t.kind) {
		if candidate.Kind() == board.ValueMoney {
			return fmt.Sprintf("; for an amount, name its measure, as {total:%s}", candidate)
		}
	}

	return ""
}

func (f *figure) measureOr(measure board.Measure) board.Measure {
	if f.measure != "" {
		return f.measure
	}

	return measure
}

func (f *figure) key(measure board.Measure, groups []group) string {
	if f.record.IsNotNil() {
		return fmt.Sprintf("%s_of_%d", f.measure,
			slices.Index(union(groups).IDs, f.record)+1)
	}
	read := f.measureOr(measure)
	if f.base == NameCount || read == measure {
		return f.name
	}

	return f.name + "_" + string(read)
}

func scopeOf(found *figure, groups []group) *board.RecordSet {
	if found.number > 0 && found.number <= len(groups) {
		return groups[found.number-1].set
	}

	return union(groups)
}

func (t *translator) readings(
	used []figure,
	groups []group,
	measure board.Measure,
) (readings []board.Reading, keys map[string]string) {
	order := Names(MaxGroups)
	sorted := slices.Clone(used)
	slices.SortStableFunc(sorted, func(a, b figure) int {
		if byName := slices.Index(order, a.name) - slices.Index(order, b.name); byName != 0 {
			return byName
		}
		if byMeasure := strings.Compare(
			string(a.measureOr(measure)), string(b.measureOr(measure)),
		); byMeasure != 0 {
			return byMeasure
		}
		return strings.Compare(a.raw, b.raw)
	})
	keys = make(map[string]string, len(sorted))
	readings = make([]board.Reading, 0, len(sorted))
	for _, found := range sorted {
		key := found.key(measure, groups)
		keys[found.raw] = key
		if slices.ContainsFunc(readings, func(reading board.Reading) bool {
			return reading.Key == key
		}) {
			continue
		}
		readings = append(readings, t.reading(&found, key, groups, found.measureOr(measure)))
	}

	return readings, keys
}

func (t *translator) reading(
	found *figure,
	key string,
	groups []group,
	measure board.Measure,
) board.Reading {
	if found.record.IsNotNil() {
		return board.Reading{Key: key, FigureSource: board.FigureSource{
			Label:     valueLabel(measure),
			Records:   &board.RecordSet{Resource: t.kind, IDs: []pulid.ID{found.record}},
			Measure:   measure,
			Aggregate: board.AggregateMax,
		}}
	}
	set := scopeOf(found, groups)
	scope := ""
	if found.number > 0 && found.number <= len(groups) {
		scope = groups[found.number-1].title
		if scope == "" {
			scope = fmt.Sprintf("group %d", found.number)
		}
	}
	source := board.FigureSource{Records: set, Aggregate: board.AggregateCount}
	label := capitalized(inflect.PluralNoun(t.kind.Noun()))
	switch found.base {
	case NameTotal:
		source.Measure, source.Aggregate = measure, board.AggregateSum
		label = totalLabels[measure]
		if single, ok := singleLabel(measure); ok {
			label = single
		}
	case NameLargest:
		source.Measure, source.Aggregate = measure, board.AggregateMax
		label = largestLabels[measure]
	}
	if scope != "" {
		label += ", " + scope
	}
	source.Label = stringutils.TruncateRunes(label, board.MaxLabelRunes)

	return board.Reading{Key: key, FigureSource: source}
}

func keyed(text string, keys map[string]string) string {
	for raw, key := range keys {
		if raw != key {
			text = strings.ReplaceAll(text, "{"+raw+"}", "{"+key+"}")
		}
	}

	return text
}

func durationLabel(measure board.Measure) string {
	label, _ := singleLabel(measure)

	return label
}
