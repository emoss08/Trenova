package report

import (
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
)

const CurrentIRVersion = 1

type Definition struct {
	IRVersion  int            `json:"irVersion"`
	Entity     string         `json:"entity"`
	Columns    []ColumnSpec   `json:"columns"`
	Filters    *FilterGroup   `json:"filters,omitempty"`
	Having     *FilterGroup   `json:"having,omitempty"`
	Sort       []SortSpec     `json:"sort,omitempty"`
	Limit      int            `json:"limit,omitempty"`
	Pivot      *PivotSpec     `json:"pivot,omitempty"`
	Parameters []ParameterDef `json:"parameters,omitempty"`
	// Totals asks for a grand-total row computed over the whole result set
	// rather than over the rows that survived the limit, so an average stays
	// an average instead of degrading into an average of averages.
	Totals bool        `json:"totals,omitempty"`
	Charts []ChartSpec `json:"charts,omitempty"`
}

type FieldRef struct {
	Path  []string `json:"path,omitempty"`
	Field string   `json:"field"`
}

func (r FieldRef) String() string {
	if len(r.Path) == 0 {
		return r.Field
	}
	return reportcatalog.PathKey(r.Path) + "." + r.Field
}

type ColumnKind string

const (
	ColumnKindDimension = ColumnKind("dimension")
	ColumnKindMeasure   = ColumnKind("measure")
	ColumnKindComputed  = ColumnKind("computed")
)

type ComputedOp string

const (
	ComputedOpAdd      = ComputedOp("add")
	ComputedOpSubtract = ComputedOp("subtract")
	ComputedOpMultiply = ComputedOp("multiply")
	ComputedOpDivide   = ComputedOp("divide")
)

func (o ComputedOp) IsValid() bool {
	switch o {
	case ComputedOpAdd, ComputedOpSubtract, ComputedOpMultiply, ComputedOpDivide:
		return true
	default:
		return false
	}
}

// ComputedOperand is one side of a calculation: either another column in the
// same report or a constant. Exactly one of the two is set — a goal has no
// column to point at, and a column carries its own value.
type ComputedOperand struct {
	ColumnID string
	Value    *float64
}

func (o ComputedOperand) IsLiteral() bool { return o.Value != nil }

func (o ComputedOperand) IsEmpty() bool { return o.ColumnID == "" && o.Value == nil }

func (o ComputedOperand) IsAmbiguous() bool { return o.ColumnID != "" && o.Value != nil }

func (o ComputedOperand) ValueOrZero() float64 {
	if o.Value == nil {
		return 0
	}
	return *o.Value
}

type ComputedSpec struct {
	Op      ComputedOp `json:"op"`
	LeftID  string     `json:"leftId,omitempty"`
	RightID string     `json:"rightId,omitempty"`
	// LeftValue and RightValue substitute a constant for the operand on that
	// side, which is what expresses a target: revenue measured against a
	// $250,000 goal has no second measure to point at.
	LeftValue  *float64                 `json:"leftValue,omitempty"`
	RightValue *float64                 `json:"rightValue,omitempty"`
	Format     reportcatalog.FormatHint `json:"format,omitempty"`
}

func (s *ComputedSpec) Left() ComputedOperand {
	return ComputedOperand{ColumnID: s.LeftID, Value: s.LeftValue}
}

func (s *ComputedSpec) Right() ComputedOperand {
	return ComputedOperand{ColumnID: s.RightID, Value: s.RightValue}
}

// Operands returns both sides in evaluation order, which is what lets every
// caller — validation, emission, typing, sensitivity — treat the two
// symmetrically instead of repeating a left/right pair.
func (s *ComputedSpec) Operands() [2]ComputedOperand {
	return [2]ComputedOperand{s.Left(), s.Right()}
}

type DateBucket string

const (
	DateBucketNone    = DateBucket("")
	DateBucketDay     = DateBucket("day")
	DateBucketWeek    = DateBucket("week")
	DateBucketMonth   = DateBucket("month")
	DateBucketQuarter = DateBucket("quarter")
	DateBucketYear    = DateBucket("year")
)

func (b DateBucket) IsValid() bool {
	switch b {
	case DateBucketNone, DateBucketDay, DateBucketWeek, DateBucketMonth,
		DateBucketQuarter, DateBucketYear:
		return true
	default:
		return false
	}
}

// TransformOp rewrites the value at the SQL layer, so the transformed value is
// what lands in every export and what sorting and measure filters see.
// Presentation-only concerns belong on Display instead.
type TransformOp string

const (
	TransformNone     = TransformOp("")
	TransformRound    = TransformOp("round")
	TransformCeil     = TransformOp("ceil")
	TransformFloor    = TransformOp("floor")
	TransformTruncate = TransformOp("truncate")
	TransformAbs      = TransformOp("abs")
	TransformScale    = TransformOp("scale")
	TransformUpper    = TransformOp("upper")
	TransformLower    = TransformOp("lower")
	TransformTitle    = TransformOp("title")
	TransformTrim     = TransformOp("trim")
)

func (o TransformOp) IsValid() bool {
	switch o {
	case TransformNone, TransformRound, TransformCeil, TransformFloor, TransformTruncate,
		TransformAbs, TransformScale, TransformUpper, TransformLower, TransformTitle,
		TransformTrim:
		return true
	default:
		return false
	}
}

func (o TransformOp) IsNumeric() bool {
	switch o {
	case TransformRound, TransformCeil, TransformFloor, TransformTruncate,
		TransformAbs, TransformScale:
		return true
	case TransformNone, TransformUpper, TransformLower, TransformTitle, TransformTrim:
		return false
	default:
		return false
	}
}

func (o TransformOp) IsText() bool {
	switch o {
	case TransformUpper, TransformLower, TransformTitle, TransformTrim:
		return true
	case TransformNone, TransformRound, TransformCeil, TransformFloor,
		TransformTruncate, TransformAbs, TransformScale:
		return false
	default:
		return false
	}
}

func (o TransformOp) UsesPrecision() bool {
	switch o {
	case TransformRound, TransformCeil, TransformFloor, TransformTruncate:
		return true
	case TransformNone, TransformAbs, TransformScale, TransformUpper,
		TransformLower, TransformTitle, TransformTrim:
		return false
	default:
		return false
	}
}

func (o TransformOp) UsesFactor() bool {
	return o == TransformScale
}

// ProducesDecimal reports whether the transform widens an integer input to a
// numeric result, which drives how the executor decodes the column.
func (o TransformOp) ProducesDecimal() bool {
	return o.UsesPrecision() || o == TransformScale
}

const MaxTransformPrecision = 6

type TransformSpec struct {
	Op        TransformOp `json:"op"`
	Precision *int        `json:"precision,omitempty"`
	Factor    *float64    `json:"factor,omitempty"`
}

func (t *TransformSpec) PrecisionOrZero() int {
	if t == nil || t.Precision == nil {
		return 0
	}
	return *t.Precision
}

func (t *TransformSpec) FactorOrOne() float64 {
	if t == nil || t.Factor == nil {
		return 1
	}
	return *t.Factor
}

type ColumnSpec struct {
	ID     string                    `json:"id"`
	Ref    FieldRef                  `json:"ref"`
	Kind   ColumnKind                `json:"kind"`
	Agg    reportcatalog.Aggregation `json:"agg,omitempty"`
	Bucket DateBucket                `json:"bucket,omitempty"`
	Label  string                    `json:"label,omitempty"`

	// Band groups a numeric dimension into ranges, which is what turns a
	// per-shipment dwell time into a distribution — "0–2h, 2–4h, 4h+" — rather
	// than one row per distinct duration. It is the numeric counterpart of
	// Bucket and the two are mutually exclusive.
	Band *reportfmt.Band `json:"band,omitempty"`

	Computed  *ComputedSpec   `json:"computed,omitempty"`
	Transform *TransformSpec  `json:"transform,omitempty"`
	Display   *reportfmt.Spec `json:"display,omitempty"`

	// Filter narrows the rows this measure aggregates without narrowing the
	// report, which is what makes ratios of a subset — deadhead miles over
	// total miles, revenue from one customer tier — expressible in one row.
	Filter *FilterGroup `json:"filter,omitempty"`
}

type BoolOp string

const (
	BoolOpAnd = BoolOp("and")
	BoolOpOr  = BoolOp("or")
)

func (o BoolOp) IsValid() bool {
	return o == BoolOpAnd || o == BoolOpOr
}

type FilterGroup struct {
	Op      BoolOp        `json:"op"`
	Filters []FieldFilter `json:"filters,omitempty"`
	Groups  []FilterGroup `json:"groups,omitempty"`
}

func (g *FilterGroup) IsEmpty() bool {
	return g == nil || (len(g.Filters) == 0 && len(g.Groups) == 0)
}

func (g *FilterGroup) Walk(fn func(*FieldFilter) error) error {
	if g == nil {
		return nil
	}
	for i := range g.Filters {
		if err := fn(&g.Filters[i]); err != nil {
			return err
		}
	}
	for i := range g.Groups {
		if err := g.Groups[i].Walk(fn); err != nil {
			return err
		}
	}
	return nil
}

type FieldFilter struct {
	Ref      FieldRef                  `json:"ref"`
	Operator dbtype.Operator           `json:"operator"`
	Value    any                       `json:"value,omitempty"`
	Param    string                    `json:"param,omitempty"`
	Agg      reportcatalog.Aggregation `json:"agg,omitempty"`
	// Transform rewrites the column before comparison, which is what lets a
	// drill-through match the same normalized value the grouped row displayed.
	Transform *TransformSpec `json:"transform,omitempty"`
}

type SortSpec struct {
	ColumnID  string               `json:"columnId"`
	Direction dbtype.SortDirection `json:"direction"`
}

type PivotSpec struct {
	Ref          FieldRef `json:"ref"`
	Values       []string `json:"values"`
	Labels       []string `json:"labels,omitempty"`
	MeasureIDs   []string `json:"measureIds"`
	IncludeOther bool     `json:"includeOther"`
}

func (p *PivotSpec) LabelFor(index int, fallback string) string {
	if p == nil || index < 0 || index >= len(p.Labels) {
		return fallback
	}
	if p.Labels[index] == "" {
		return fallback
	}
	return p.Labels[index]
}

// ChartType names a visual encoding of columns the report already returns —
// charts never change what is queried, only how the result is drawn.
type ChartType string

const (
	ChartBar     = ChartType("bar")
	ChartHBar    = ChartType("hbar")
	ChartLine    = ChartType("line")
	ChartArea    = ChartType("area")
	ChartPie     = ChartType("pie")
	ChartDonut   = ChartType("donut")
	ChartScatter = ChartType("scatter")
	ChartKPI     = ChartType("kpi")
	// ChartMap plots rows at coordinates the report already returns. Freight
	// is geographic, and a lane or a stop density reads as a map long before
	// it reads as a table of latitudes.
	ChartMap = ChartType("map")
)

func (t ChartType) IsValid() bool {
	switch t {
	case ChartBar, ChartHBar, ChartLine, ChartArea, ChartPie,
		ChartDonut, ChartScatter, ChartKPI, ChartMap:
		return true
	default:
		return false
	}
}

// NeedsDimension reports whether the chart plots series against a grouping
// column; KPI tiles and scatter plots read their x position elsewhere.
func (t ChartType) NeedsDimension() bool {
	switch t {
	case ChartBar, ChartHBar, ChartLine, ChartArea, ChartPie, ChartDonut:
		return true
	// A map reads its position from coordinates, and a KPI or scatter from
	// elsewhere — none of them plot against a grouping column.
	case ChartScatter, ChartKPI, ChartMap:
		return false
	default:
		return false
	}
}

// SingleSeries reports whether the encoding can only carry one measure.
func (t ChartType) SingleSeries() bool {
	switch t {
	case ChartPie, ChartDonut, ChartKPI, ChartScatter, ChartMap:
		return true
	case ChartBar, ChartHBar, ChartLine, ChartArea:
		return false
	default:
		return false
	}
}

const (
	MaxCharts        = 6
	MaxChartSeries   = 12
	MaxChartCategory = 200
)

// ChartGoal draws a reference line (or a KPI target) from either a constant or
// another column in the same row, which is how fleet goals land on the chart.
type ChartGoal struct {
	Value    *float64 `json:"value,omitempty"`
	ColumnID string   `json:"columnId,omitempty"`
	Label    string   `json:"label,omitempty"`
}

func (g *ChartGoal) IsEmpty() bool {
	return g == nil || (g.Value == nil && g.ColumnID == "")
}

type ChartSpec struct {
	ID         string     `json:"id"`
	Type       ChartType  `json:"type"`
	Title      string     `json:"title,omitempty"`
	XColumnID  string     `json:"xColumnId,omitempty"`
	SeriesIDs  []string   `json:"seriesIds,omitempty"`
	Stacked    bool       `json:"stacked,omitempty"`
	HideLegend bool       `json:"hideLegend,omitempty"`
	ShowValues bool       `json:"showValues,omitempty"`
	Curved     bool       `json:"curved,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	CompareID  string     `json:"compareId,omitempty"`
	Goal       *ChartGoal `json:"goal,omitempty"`
	// LatColumnID and LngColumnID position each row on a map. They are named
	// explicitly rather than folded into SeriesIDs because a coordinate pair
	// is not a series — swapping them silently plots the world sideways.
	LatColumnID string `json:"latColumnId,omitempty"`
	LngColumnID string `json:"lngColumnId,omitempty"`
	// LabelColumnID names each point; empty falls back to the first dimension.
	LabelColumnID string `json:"labelColumnId,omitempty"`
}

// NeedsCoordinates reports whether the chart positions rows geographically.
func (t ChartType) NeedsCoordinates() bool { return t == ChartMap }

type ParameterDef struct {
	Name          string                  `json:"name"`
	Label         string                  `json:"label,omitempty"`
	Type          reportcatalog.FieldType `json:"type"`
	Required      bool                    `json:"required"`
	Default       any                     `json:"default,omitempty"`
	Multi         bool                    `json:"multi,omitempty"`
	AllowedValues []string                `json:"allowedValues,omitempty"`
	RefEntity     string                  `json:"refEntity,omitempty"`
}

func (d *Definition) ColumnByID(id string) (*ColumnSpec, bool) {
	for i := range d.Columns {
		if d.Columns[i].ID == id {
			return &d.Columns[i], true
		}
	}
	return nil, false
}

func (d *Definition) HasMeasures() bool {
	for i := range d.Columns {
		if d.Columns[i].Kind == ColumnKindMeasure {
			return true
		}
	}
	return false
}

func (d *Definition) Parameter(name string) (*ParameterDef, bool) {
	for i := range d.Parameters {
		if d.Parameters[i].Name == name {
			return &d.Parameters[i], true
		}
	}
	return nil, false
}

func (d *Definition) WalkFilters(fn func(*FieldFilter) error) error {
	if err := d.Filters.Walk(fn); err != nil {
		return err
	}
	return d.Having.Walk(fn)
}
