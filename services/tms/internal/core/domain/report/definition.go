package report

import (
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/reportcatalog"
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

type ComputedSpec struct {
	Op      ComputedOp               `json:"op"`
	LeftID  string                   `json:"leftId"`
	RightID string                   `json:"rightId"`
	Format  reportcatalog.FormatHint `json:"format,omitempty"`
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

type ColumnSpec struct {
	ID       string                    `json:"id"`
	Ref      FieldRef                  `json:"ref"`
	Kind     ColumnKind                `json:"kind"`
	Agg      reportcatalog.Aggregation `json:"agg,omitempty"`
	Bucket   DateBucket                `json:"bucket,omitempty"`
	Label    string                    `json:"label,omitempty"`
	Computed *ComputedSpec             `json:"computed,omitempty"`
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
}

type SortSpec struct {
	ColumnID  string               `json:"columnId"`
	Direction dbtype.SortDirection `json:"direction"`
}

type PivotSpec struct {
	Ref          FieldRef `json:"ref"`
	Values       []string `json:"values"`
	MeasureIDs   []string `json:"measureIds"`
	IncludeOther bool     `json:"includeOther"`
}

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
