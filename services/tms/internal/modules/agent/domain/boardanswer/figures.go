package boardanswer

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardtyped"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	measureMark = ":"
	listPrefix  = "art_"
)

type figure struct {
	raw     string
	name    string
	base    string
	number  int
	measure board.Measure
	record  pulid.ID
	metric  int
	named   string
	whole   bool
}

const noMetric = -1

func valueLabel(kind recordkind.Kind, measure board.Measure) string {
	if spec, ok := measure.Spec(kind); ok && spec.Label != "" {
		return spec.Label
	}

	return string(measure)
}

func singleLabel(kind recordkind.Kind, measure board.Measure) (string, bool) {
	spec, ok := measure.Spec(kind)

	return valueLabel(kind, measure), ok && (spec.Value == boardcatalog.ValueDuration ||
		spec.Value == boardcatalog.ValueDays || spec.Value == boardcatalog.ValueHours)
}

func metricGroupOf(groups []group, name string) int {
	metric, known := boardregistry.Catalog().MetricNamed(name)
	if !known {
		return noMetric
	}
	for idx := range groups {
		if groups[idx].metric != nil && groups[idx].metric.Name == metric.Name {
			return idx
		}
	}

	return noMetric
}

func groupDeclaring(groups []group, name string) int {
	found := noMetric
	for idx := range groups {
		if groups[idx].metric == nil {
			continue
		}
		if _, ok := groups[idx].metric.Figure(name); !ok {
			continue
		}
		if found != noMetric {
			return noMetric
		}
		found = idx
	}

	return found
}

func metricFigureNames(metric *boardcatalog.Metric) string {
	names := make([]string, 0, len(metric.Figures))
	for _, figure := range metric.Figures {
		names = append(names, figure.Name)
	}

	return stringutils.JoinAlternatives(names)
}

func (t *translator) metricFigure(
	path, raw, left, right string,
	colon bool,
	groups []group,
) (*figure, bool) {
	declared := func(at int, name string) *figure {
		spec, _ := groups[at].metric.Figure(name)
		return &figure{raw: raw, metric: at, named: spec.Name}
	}
	if !colon {
		if Known(left, len(groups)) {
			return nil, false
		}
		if at := groupDeclaring(groups, left); at != noMetric {
			return declared(at, left), true
		}
		return nil, false
	}
	for _, sides := range [][2]string{{left, right}, {right, left}} {
		name, metricName := sides[0], sides[1]
		at := metricGroupOf(groups, metricName)
		if at == noMetric {
			continue
		}
		if _, ok := groups[at].metric.Figure(name); ok {
			return declared(at, name), true
		}
		if other := groupDeclaring(groups, name); other != noMetric {
			return declared(other, name), true
		}
		t.addf("%s places {%s}, and the %s metric's figures are %s", path, raw,
			groups[at].metric.Name, metricFigureNames(groups[at].metric))
		return nil, true
	}
	for _, name := range []string{right, left} {
		if metric, known := boardregistry.Catalog().MetricNamed(name); known {
			t.addf("%s places {%s}, a figure of the %s metric, which no group shows; add a "+
				"group with metric %s", path, raw, metric.Name, metric.Name)
			return nil, true
		}
	}

	return nil, false
}

func (t *translator) noRecordFigures(used []figure, groups []group) {
	if len(groups) == 0 {
		return
	}
	records := slices.ContainsFunc(groups, func(g group) bool { return g.set != nil })
	for _, found := range used {
		if found.metric != noMetric || found.record.IsNotNil() {
			continue
		}
		switch {
		case !records:
			t.addf("{%s} counts or adds up records, and no group holds records; write a "+
				"metric's figure as {figure:metric}", found.raw)
		case found.number > 0 && found.number <= len(groups) &&
			groups[found.number-1].metric != nil:
			t.addf("{%s} reads group %d, which is a metric; write its figure as "+
				"{figure:%s}", found.raw, found.number, groups[found.number-1].metric.Name)
		}
	}
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
		if metric, isMetric := t.metricFigure(
			path,
			raw,
			name,
			named,
			hasMeasure,
			groups,
		); isMetric {
			if metric != nil {
				found = append(found, *metric)
			}
			continue
		}
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
		current := figure{raw: raw, name: name, base: base, number: number, metric: noMetric}
		if base == NameAll && strings.HasPrefix(named, listPrefix) {
			current.named = named
		}
		current.whole = base == NameCount && hasMeasure && t.namesWhole(named)
		if hasMeasure && base != NameCount && base != NameAll && t.kind != "" {
			spec, ok := boardregistry.Catalog().MeasureOf(t.kind, named)
			current.measure = board.Measure(spec.Name)
			if !ok {
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
	if Known(left, len(groups)) {
		return figureOfRecord(raw, left, right, groups)
	}
	spec, ok := boardregistry.Catalog().MeasureOf(t.kind, left)
	record := right
	if !ok {
		spec, ok = boardregistry.Catalog().MeasureOf(t.kind, right)
		record = left
	}
	if !ok {
		return nil, false
	}
	measure := board.Measure(spec.Name)
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

	return &figure{raw: raw, measure: measure, record: id, metric: noMetric}, true
}

func figureOfRecord(raw, left, right string, groups []group) (*figure, bool) {
	base, _ := splitName(left)
	id, err := pulid.Parse(right)
	if base == NameCount || base == NameAll || err != nil || id.IsNil() ||
		!slices.Contains(union(groups).IDs, id) {
		return nil, false
	}

	return &figure{raw: raw, name: left, base: base, record: id, metric: noMetric}, true
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
		if measures := boardregistry.Catalog().MeasuresOf(t.kind); len(measures) > 0 {
			read = board.Measure(measures[0].Name)
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
		if found.base != NameTotal || found.record.IsNotNil() {
			continue
		}
		read := found.measureOr(measure)
		set := scopeOf(&found, groups)
		spec, known := read.Spec(t.kind)
		if read == "" || !known || spec.Summable || set == nil || len(set.IDs) < sideBySide {
			continue
		}
		t.addf("{%s} would add up %s across %d %s, and %s added across records is not a "+
			"figure anyone reads. For the most, write {largest%s}; for one record's own, write "+
			"its measure and its number, as {%s:A-1001}%s",
			found.raw, read, len(set.IDs), inflect.PluralNoun(t.kind.Noun()),
			strings.ToLower(t.durationLabel(read)), measureSuffix(read, measure), read,
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
	measures := boardregistry.Catalog().MeasuresOf(t.kind)
	for idx := range measures {
		if measures[idx].Value == boardcatalog.ValueMoney {
			return fmt.Sprintf("; for an amount, name its measure, as {total:%s}",
				measures[idx].Name)
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

func (f *figure) key(measure board.Measure, groups []group, kind recordkind.Kind) string {
	if f.metric != noMetric {
		return stringutils.TruncateRunes(f.named, metricKeyRunes) + "_" + strconv.Itoa(f.metric+1)
	}
	if f.whole {
		whole, _ := boardtyped.WholeOf(kind)
		return whole.Key(f.name)
	}
	if f.record.IsNotNil() {
		return fmt.Sprintf("%s_of_%d", f.measure,
			slices.Index(union(groups).IDs, f.record)+1)
	}
	read := f.measureOr(measure)
	if f.counts() || read == measure {
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
		key := found.key(measure, groups, t.kind)
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
	if found.metric != noMetric {
		held := &groups[found.metric]
		spec, _ := held.metric.Figure(found.named)
		return board.Reading{Key: key, FigureSource: board.FigureSource{
			Label:  stringutils.TruncateRunes(spec.Label, board.MaxLabelRunes),
			Metric: held.metric.Name,
			Figure: spec.Name,
			For:    held.forID,
			Period: held.period,
		}}
	}
	if found.record.IsNotNil() {
		return board.Reading{Key: key, FigureSource: board.FigureSource{
			Label:     valueLabel(t.kind, measure),
			Records:   &board.RecordSet{Kind: t.kind, IDs: []pulid.ID{found.record}},
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
	if found.whole {
		return t.wholeReading(found, key, scope, groups)
	}
	source := board.FigureSource{Records: set, Aggregate: board.AggregateCount}
	label := stringutils.UpperFirst(inflect.PluralNoun(t.kind.Noun()))
	spec, _ := measure.Spec(t.kind)
	switch found.base {
	case NameTotal:
		source.Measure, source.Aggregate = measure, board.AggregateSum
		label = spec.Total
		if single, ok := singleLabel(t.kind, measure); ok {
			label = single
		}
	case NameLargest:
		source.Measure, source.Aggregate = measure, board.AggregateMax
		label = spec.Largest
	case NameAll:
		label += " in the whole list"
		if strings.HasPrefix(found.named, listPrefix) {
			source = board.FigureSource{
				List:      pulid.ID(found.named),
				Aggregate: board.AggregateCount,
			}
		}
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

func (t *translator) durationLabel(measure board.Measure) string {
	label, _ := singleLabel(t.kind, measure)

	return label
}

const metricKeyRunes = 28
