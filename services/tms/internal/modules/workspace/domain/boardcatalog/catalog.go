package boardcatalog

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/shared/inflect"
)

type Shape string

const (
	ShapeAmount    = Shape("amount")
	ShapeTrack     = Shape("track")
	ShapeCountdown = Shape("countdown")
	ShapeLimit     = Shape("limit")
	ShapeBounded   = Shape("bounded")
	ShapeCentred   = Shape("centred")
	ShapeSpread    = Shape("spread")
)

func (s Shape) Bar() bool {
	return s != ShapeAmount && s != ""
}

type Form string

const (
	FormTrend   = Form("trend")
	FormBars    = Form("bars")
	FormColumns = Form("columns")
	FormSplit   = Form("split")
	FormGrid    = Form("grid")
	FormDots    = Form("dots")
	FormSpans   = Form("spans")
	FormLog     = Form("log")
	FormChecks  = Form("checks")
	FormTable   = Form("table")
	FormFigures = Form("figures")
)

func AllForms() []Form {
	return []Form{
		FormTrend, FormBars, FormColumns, FormSplit, FormGrid, FormDots, FormSpans, FormLog,
		FormChecks, FormTable, FormFigures,
	}
}

func (f Form) Charted() bool {
	return f == FormTrend || f == FormBars || f == FormColumns || f == FormDots || f == FormSpans
}

type Period string

const (
	PeriodToday       = Period("today")
	PeriodYesterday   = Period("yesterday")
	PeriodThisWeek    = Period("thisWeek")
	PeriodLastWeek    = Period("lastWeek")
	PeriodNextWeek    = Period("nextWeek")
	PeriodThisMonth   = Period("thisMonth")
	PeriodLastMonth   = Period("lastMonth")
	PeriodThisQuarter = Period("thisQuarter")
	PeriodLastQuarter = Period("lastQuarter")
	PeriodNext48Hours = Period("next48Hours")
	PeriodSlip        = Period("slip")
	PeriodNext30Days  = Period("next30Days")
)

func AllPeriods() []Period {
	return []Period{
		PeriodToday, PeriodYesterday, PeriodThisWeek, PeriodLastWeek, PeriodNextWeek,
		PeriodThisMonth, PeriodLastMonth, PeriodThisQuarter, PeriodLastQuarter, PeriodNext48Hours,
		PeriodSlip, PeriodNext30Days,
	}
}

type Kind struct {
	Kind   recordkind.Kind
	Words  string
	PartOf recordkind.Kind
	Called []string
}

func (k *Kind) Nouns() []string {
	nouns := []string{k.Kind.Noun(), k.Plural()}
	for _, word := range k.Called {
		if !slices.Contains(nouns, word) {
			nouns = append(nouns, word)
		}
	}

	return nouns
}

func (k *Kind) Plural() string {
	if k.Words != "" {
		return k.Words
	}

	return inflect.PluralNoun(k.Kind.Noun())
}

type Field struct {
	Resource permission.Resource
	Name     string
}

func Read(resource permission.Resource, names ...string) []Field {
	fields := make([]Field, 0, len(names))
	for _, name := range names {
		fields = append(fields, Field{Resource: resource, Name: name})
	}

	return fields
}

type Measure struct {
	Name      string
	Kind      recordkind.Kind
	Shape     Shape
	Value     ValueKind
	Summable  bool
	Adverse   bool
	Ascending bool
	Default   bool
	Aliases   []string
	Requires  []permission.Resource
	Reads     []Field
	Place     *PlacePart
	Words     string
	Label     string
	Total     string
	Largest   string
	Heading   string
	FollowUp  *FollowUp
	Says      []string
}

type FollowUp struct {
	Label string
	Ask   string
}

type PlacePart struct {
	Requires []permission.Resource
	Reads    []Field
}

type Sketch struct {
	Rows int
}

type Figure struct {
	Name    string
	Label   string
	Value   ValueKind
	Feature platformcatalog.FeatureKey
	Reads   []Field
}

type Metric struct {
	Name      string
	Form      Form
	For       recordkind.Kind
	ForNeeded bool
	Periods   []Period
	Figures   []Figure
	Requires  []permission.Resource
	Reads     []Field
	Unscoped  []permission.Resource
	Feature   platformcatalog.FeatureKey
	Aliases   []string
	Words     string
	Sketch    Sketch
	Records   *MetricRecords
	Leads     []string
}

type MetricRecords struct {
	Kind    recordkind.Kind
	Measure string
	Words   string
	From    string
}

func MoneyFigure(name, label string, reads []Field) Figure {
	return Figure{Name: name, Label: label, Value: ValueMoney, Reads: reads}
}

func (m *Metric) Figure(name string) (Figure, bool) {
	for _, figure := range m.Figures {
		if strings.EqualFold(figure.Name, name) {
			return figure, true
		}
	}

	return Figure{}, false
}

func (m *Metric) Led() (leads, rest []Figure) {
	if len(m.Leads) == 0 {
		return m.Figures, nil
	}
	for _, name := range m.Leads {
		if figure, ok := m.Figure(name); ok {
			leads = append(leads, figure)
		}
	}
	for _, figure := range m.Figures {
		if !slices.ContainsFunc(m.Leads, func(name string) bool {
			return strings.EqualFold(name, figure.Name)
		}) {
			rest = append(rest, figure)
		}
	}

	return leads, rest
}

func (m *Metric) DefaultPeriod() Period {
	if len(m.Periods) == 0 {
		return ""
	}

	return m.Periods[0]
}

func (m *Metric) Takes(period Period) bool {
	return slices.Contains(m.Periods, period)
}

type Entries struct {
	Kinds    []Kind
	Measures []Measure
	Metrics  []Metric
}

type Catalog struct {
	kinds    []Kind
	measures []Measure
	metrics  []Metric
	kindAt   map[recordkind.Kind]int
}

func New(entries Entries) *Catalog {
	catalog := &Catalog{
		kinds:    slices.Clone(entries.Kinds),
		measures: slices.Clone(entries.Measures),
		metrics:  slices.Clone(entries.Metrics),
		kindAt:   make(map[recordkind.Kind]int, len(entries.Kinds)),
	}
	for idx := range catalog.kinds {
		catalog.kindAt[catalog.kinds[idx].Kind] = idx
	}

	return catalog
}

func (c *Catalog) Kinds() []Kind {
	return slices.Clone(c.kinds)
}

func (c *Catalog) Measures() []Measure {
	return slices.Clone(c.measures)
}

func (c *Catalog) Metrics() []Metric {
	return slices.Clone(c.metrics)
}

func (c *Catalog) KindOf(kind recordkind.Kind) (Kind, bool) {
	idx, ok := c.kindAt[kind]
	if !ok {
		return Kind{}, false
	}

	return c.kinds[idx], true
}

func (c *Catalog) Supports(kind recordkind.Kind) bool {
	_, ok := c.kindAt[kind]

	return ok
}

func (c *Catalog) MeasuresOf(kind recordkind.Kind) []Measure {
	out := make([]Measure, 0, 3)
	for idx := range c.measures {
		if c.measures[idx].Kind == kind {
			out = append(out, c.measures[idx])
		}
	}

	return out
}

func (c *Catalog) MeasureOf(kind recordkind.Kind, name string) (Measure, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Measure{}, false
	}
	for idx := range c.measures {
		measure := &c.measures[idx]
		if measure.Kind != kind {
			continue
		}
		if strings.EqualFold(measure.Name, name) {
			return *measure, true
		}
	}
	for idx := range c.measures {
		measure := &c.measures[idx]
		if measure.Kind != kind {
			continue
		}
		if slices.ContainsFunc(measure.Aliases, func(alias string) bool {
			return strings.EqualFold(alias, name)
		}) {
			return *measure, true
		}
	}

	return Measure{}, false
}

func (c *Catalog) DefaultMeasure(kind recordkind.Kind) (Measure, bool) {
	for idx := range c.measures {
		if c.measures[idx].Kind == kind && c.measures[idx].Default {
			return c.measures[idx], true
		}
	}

	return Measure{}, false
}

func (c *Catalog) MeasureNames() []string {
	names := make([]string, 0, len(c.measures))
	for idx := range c.measures {
		if !slices.Contains(names, c.measures[idx].Name) {
			names = append(names, c.measures[idx].Name)
		}
	}

	return names
}

func (c *Catalog) MetricNamed(name string) (Metric, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Metric{}, false
	}
	for idx := range c.metrics {
		if strings.EqualFold(c.metrics[idx].Name, name) {
			return c.metrics[idx], true
		}
	}
	for idx := range c.metrics {
		if slices.ContainsFunc(c.metrics[idx].Aliases, func(alias string) bool {
			return strings.EqualFold(alias, name)
		}) {
			return c.metrics[idx], true
		}
	}

	return Metric{}, false
}

func (c *Catalog) FigureNamed(name string) (Metric, Figure, bool) {
	var (
		found  Metric
		figure Figure
		hits   int
	)
	for idx := range c.metrics {
		if candidate, ok := c.metrics[idx].Figure(name); ok {
			found, figure = c.metrics[idx], candidate
			hits++
		}
	}
	if hits != 1 {
		return Metric{}, Figure{}, false
	}

	return found, figure, true
}

func (c *Catalog) FigureNames(form Form) []string {
	names := make([]string, 0, 16)
	for idx := range c.metrics {
		if c.metrics[idx].Form != form {
			continue
		}
		for _, figure := range c.metrics[idx].Figures {
			names = append(names, figure.Name)
		}
	}

	return names
}

func (c *Catalog) MetricNames() []string {
	names := make([]string, 0, len(c.metrics))
	for idx := range c.metrics {
		names = append(names, c.metrics[idx].Name)
	}

	return names
}
