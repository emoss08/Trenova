package boardcatalog

import (
	"github.com/shopspring/decimal"
)

type SeriesRole string

const (
	RoleNow   = SeriesRole("now")
	RoleWas   = SeriesRole("was")
	RoleStack = SeriesRole("stack")
)

type PartKind string

const (
	PartUsed   = PartKind("used")
	PartOver   = PartKind("over")
	PartCredit = PartKind("credit")
)

type CellState string

const (
	CellCovered = CellState("covered")
	CellShort   = CellState("short")
	CellDeep    = CellState("deep")
	CellMaybe   = CellState("maybe")
)

type SpanKind string

const (
	SpanCover = SpanKind("cover")
	SpanGap   = SpanKind("gap")
)

type CellMark string

const (
	MarkHot = CellMark("hot")
	MarkOff = CellMark("off")
	MarkW   = CellMark("w")
)

type MetricRead struct {
	Title   string                 `json:"title"`
	Source  string                 `json:"source"`
	Figures map[string]*Value      `json:"figures"`
	Changes map[string]FigureShift `json:"changes,omitempty"`
	Chart   *Chart                 `json:"chart,omitempty"`
	Split   *Split                 `json:"split,omitempty"`
	Grid    *Grid                  `json:"grid,omitempty"`
	Log     []LogEntry             `json:"log,omitempty"`
	Checks  []Check                `json:"checks,omitempty"`
	Table   *Table                 `json:"table,omitempty"`
}

type FigureShift struct {
	Was   *Value `json:"was,omitempty"`
	WasAt int64  `json:"wasAt,omitempty"`
	Tone  Tone   `json:"tone,omitempty"`
}

type Tick struct {
	At     int64  `json:"at,omitempty"`
	Label  string `json:"label,omitempty"`
	Marked bool   `json:"marked,omitempty"`
}

type Series struct {
	Name   string             `json:"name"`
	Role   SeriesRole         `json:"role"`
	Points []*decimal.Decimal `json:"points"`
}

type Band struct {
	From *int             `json:"from,omitempty"`
	To   *int             `json:"to,omitempty"`
	Low  *decimal.Decimal `json:"low,omitempty"`
	High *decimal.Decimal `json:"high,omitempty"`
}

type Line struct {
	Value decimal.Decimal `json:"value"`
	Label string          `json:"label"`
}

type Span struct {
	From int64    `json:"from"`
	To   int64    `json:"to"`
	Kind SpanKind `json:"kind"`
}

type Dot struct {
	At     int64            `json:"at,omitempty"`
	Value  *decimal.Decimal `json:"value,omitempty"`
	Marked bool             `json:"marked,omitempty"`
	Ghost  bool             `json:"ghost,omitempty"`
	Label  string           `json:"label,omitempty"`
}

type ChartRow struct {
	Label  string `json:"label"`
	Ref    *Ref   `json:"ref,omitempty"`
	Value  *Value `json:"value,omitempty"`
	Marked bool   `json:"marked,omitempty"`
	Rest   bool   `json:"rest,omitempty"`
	Spans  []Span `json:"spans,omitempty"`
	Dots   []Dot  `json:"dots,omitempty"`
}

type Chart struct {
	Unit     ValueKind  `json:"unit"`
	Currency string     `json:"currency,omitempty"`
	X        []Tick     `json:"x"`
	Series   []Series   `json:"series"`
	Marked   []int      `json:"marked,omitempty"`
	Band     *Band      `json:"band,omitempty"`
	Target   *Line      `json:"target,omitempty"`
	Ref      *Line      `json:"ref,omitempty"`
	Ghost    bool       `json:"ghost,omitempty"`
	Rows     []ChartRow `json:"rows,omitempty"`
}

type Part struct {
	Label string   `json:"label"`
	Value Value    `json:"value"`
	Kind  PartKind `json:"kind"`
}

type Split struct {
	Total Value  `json:"total"`
	Parts []Part `json:"parts"`
}

type GridHead struct {
	Label    string `json:"label"`
	Sublabel string `json:"sublabel,omitempty"`
}

type Cell struct {
	Have     int       `json:"have"`
	Need     int       `json:"need"`
	NeedHigh int       `json:"needHigh"`
	State    CellState `json:"state"`
}

type GridRow struct {
	Label string `json:"label"`
	Cells []Cell `json:"cells"`
}

type Grid struct {
	Columns []GridHead `json:"columns"`
	Rows    []GridRow  `json:"rows"`
}

type LogEntry struct {
	At     int64  `json:"at"`
	Tone   Tone   `json:"tone,omitempty"`
	Text   string `json:"text"`
	Detail string `json:"detail,omitempty"`
	Refs   []Ref  `json:"refs,omitempty"`
}

type Check struct {
	Title  string `json:"title"`
	Tone   Tone   `json:"tone"`
	Detail string `json:"detail,omitempty"`
	Count  *Value `json:"count,omitempty"`
	Refs   []Ref  `json:"refs,omitempty"`
	Hidden int    `json:"hidden,omitempty"`
}

type ColumnKind string

const (
	ColumnText  = ColumnKind("text")
	ColumnValue = ColumnKind("value")
)

type Column struct {
	Label string     `json:"label"`
	Kind  ColumnKind `json:"kind"`
}

type TableCell struct {
	Value *Value   `json:"value,omitempty"`
	Text  string   `json:"text,omitempty"`
	Mark  CellMark `json:"mark,omitempty"`
}

type Table struct {
	Columns []Column      `json:"columns"`
	Rows    [][]TableCell `json:"rows"`
}
