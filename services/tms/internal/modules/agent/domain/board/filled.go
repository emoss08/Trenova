package board

import (
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type BlockState string

const (
	StateReady      = BlockState("ready")
	StateEmpty      = BlockState("empty")
	StateUnreadable = BlockState("unreadable")
)

type SourceKind string

const (
	SourceRecords  = SourceKind("records")
	SourceAnalysis = SourceKind("analysis")
	SourceMetrics  = SourceKind("metrics")
	SourceActivity = SourceKind("activity")
	SourceTable    = SourceKind("table")
	SourceChecks   = SourceKind("checks")
	SourceMixed    = SourceKind("mixed")
	SourceNone     = SourceKind("")
)

type MeasureState string

const (
	MeasureLate      = MeasureState("late")
	MeasureHeld      = MeasureState("held")
	MeasureOnTime    = MeasureState("onTime")
	MeasureDelivered = MeasureState("delivered")
	MeasureOpen      = MeasureState("open")
	MeasureSettled   = MeasureState("settled")
)

type Value struct {
	Kind     ValueKind       `json:"kind"`
	Number   decimal.Decimal `json:"number"`
	Currency string          `json:"currency,omitempty"`
	Text     string          `json:"text,omitempty"`
}

func (v *Value) Same(other *Value) bool {
	if v == nil || other == nil {
		return v == other
	}

	return v.Kind == other.Kind && v.Currency == other.Currency && v.Text == other.Text &&
		v.Number.Equal(other.Number)
}

type Source struct {
	Kind     SourceKind          `json:"kind"`
	Resource permission.Resource `json:"resource,omitempty"`
	Count    int                 `json:"count,omitempty"`
	Readable int                 `json:"readable,omitempty"`
	Title    string              `json:"title,omitempty"`
}

type RowMeasure struct {
	Measure Measure      `json:"measure"`
	Value   Value        `json:"value"`
	State   MeasureState `json:"state,omitempty"`
	At      int64        `json:"at,omitempty"`
	Tone    Tone         `json:"tone,omitempty"`
}

type Row struct {
	Resource permission.Resource `json:"resource"`
	ID       pulid.ID            `json:"id"`
	Readable bool                `json:"readable"`
	Label    string              `json:"label,omitempty"`
	Subtitle string              `json:"subtitle,omitempty"`
	Status   string              `json:"status,omitempty"`
	Measure  *RowMeasure         `json:"measure,omitempty"`
}

type FilledFigure struct {
	Label string `json:"label"`
	Value *Value `json:"value,omitempty"`
	Was   *Value `json:"was,omitempty"`
	Tone  Tone   `json:"tone,omitempty"`
}

type CompareLine struct {
	Label  string   `json:"label"`
	Better Better   `json:"better,omitempty"`
	Cells  []*Value `json:"cells"`
	Wins   []bool   `json:"wins"`
}

type FilledCompare struct {
	Columns []Row         `json:"columns"`
	Lines   []CompareLine `json:"lines"`
	Wins    []int         `json:"wins"`
}

type FilledTable struct {
	Table    analysis.Table `json:"table"`
	Total    int            `json:"total"`
	Artifact pulid.ID       `json:"artifact,omitempty"`
}

type Event struct {
	At        int64          `json:"at"`
	Operation string         `json:"operation"`
	Actor     string         `json:"actor,omitempty"`
	Comment   string         `json:"comment,omitempty"`
	Changes   map[string]any `json:"changes,omitempty"`
}

type FilledCheck struct {
	Title   string `json:"title"`
	Tone    Tone   `json:"tone"`
	Detail  string `json:"detail,omitempty"`
	Count   *Value `json:"count,omitempty"`
	Was     *Value `json:"was,omitempty"`
	Records []Row  `json:"records,omitempty"`
	Hidden  int    `json:"hidden,omitempty"`
}

type FilledMeter struct {
	Clear   int    `json:"clear"`
	Total   int    `json:"total"`
	Tones   []Tone `json:"tones,omitempty"`
	Percent *Value `json:"percent,omitempty"`
}

type FilledSegment struct {
	Text    string `json:"text"`
	Numeral int    `json:"numeral,omitempty"`
	Tone    Tone   `json:"tone,omitempty"`
	Struck  bool   `json:"struck,omitempty"`
}

type FilledBlock struct {
	Type     BlockType       `json:"type"`
	Title    string          `json:"title,omitempty"`
	Half     bool            `json:"half,omitempty"`
	Numeral  int             `json:"numeral,omitempty"`
	State    BlockState      `json:"state"`
	Tone     Tone            `json:"tone,omitempty"`
	Source   Source          `json:"source"`
	Figures  []FilledFigure  `json:"figures,omitempty"`
	Measure  Measure         `json:"measure,omitempty"`
	Rows     []Row           `json:"rows,omitempty"`
	Compare  *FilledCompare  `json:"compare,omitempty"`
	Table    *FilledTable    `json:"table,omitempty"`
	Chart    *analysis.Chart `json:"chart,omitempty"`
	Events   []Event         `json:"events,omitempty"`
	Checks   []FilledCheck   `json:"checks,omitempty"`
	Meter    *FilledMeter    `json:"meter,omitempty"`
	Text     []FilledSegment `json:"text,omitempty"`
	NoteTone Tone            `json:"noteTone,omitempty"`
	Chips    []Row           `json:"chips,omitempty"`
}

type Filled struct {
	Title   string          `json:"title"`
	Verdict []FilledSegment `json:"verdict"`
	Blocks  []FilledBlock   `json:"blocks"`
	ReadAt  int64           `json:"readAt"`
}

type Snapshot struct {
	At     int64            `json:"at"`
	Values map[string]Value `json:"values"`
}

func FigureKey(block, figure int) string {
	return "f." + strconv.Itoa(block) + "." + strconv.Itoa(figure)
}

func CheckKey(block, check int) string {
	return "c." + strconv.Itoa(block) + "." + strconv.Itoa(check)
}

func (f *Filled) Snapshot() Snapshot {
	snapshot := Snapshot{At: f.ReadAt, Values: make(map[string]Value, 8)}
	for b := range f.Blocks {
		block := &f.Blocks[b]
		for idx := range block.Figures {
			if value := block.Figures[idx].Value; value != nil {
				snapshot.Values[FigureKey(b, idx)] = *value
			}
		}
		for idx := range block.Checks {
			if value := block.Checks[idx].Count; value != nil {
				snapshot.Values[CheckKey(b, idx)] = *value
			}
		}
	}

	return snapshot
}

func (f *Filled) Compare(snapshot *Snapshot) {
	if snapshot == nil || len(snapshot.Values) == 0 {
		return
	}
	for b := range f.Blocks {
		block := &f.Blocks[b]
		for idx := range block.Figures {
			figure := &block.Figures[idx]
			if was, kept := snapshot.Values[FigureKey(b, idx)]; kept && figure.Value != nil &&
				!figure.Value.Same(&was) {
				figure.Was = &was
			}
		}
		for idx := range block.Checks {
			check := &block.Checks[idx]
			if was, kept := snapshot.Values[CheckKey(b, idx)]; kept && check.Count != nil &&
				!check.Count.Same(&was) {
				check.Was = &was
			}
		}
	}
}
