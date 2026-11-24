package board

import (
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardbar"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/shared/pulid"
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

type Source struct {
	Kind       SourceKind      `json:"kind"`
	RecordKind recordkind.Kind `json:"recordKind,omitempty"`
	Count      int             `json:"count,omitempty"`
	Readable   int             `json:"readable,omitempty"`
	Title      string          `json:"title,omitempty"`
}

type RowMeasure struct {
	Measure  Measure             `json:"measure"`
	Value    *boardcatalog.Value `json:"value,omitempty"`
	Standing boardreading.Words  `json:"standing"`
	Fact     *boardreading.Words `json:"fact,omitempty"`
	Tone     boardcatalog.Tone   `json:"tone,omitempty"`
	Bar      *boardbar.Bar       `json:"bar,omitempty"`
}

type Row struct {
	Kind       recordkind.Kind   `json:"kind"`
	ID         pulid.ID          `json:"id"`
	Readable   bool              `json:"readable"`
	Label      string            `json:"label,omitempty"`
	Subtitle   string            `json:"subtitle,omitempty"`
	Status     string            `json:"status,omitempty"`
	StatusTone boardcatalog.Tone `json:"statusTone,omitempty"`
	Opens      *boardcatalog.Ref `json:"opens,omitempty"`
	Measure    *RowMeasure       `json:"measure,omitempty"`
}

type WorkSignal struct {
	Lens  string            `json:"lens"`
	Title string            `json:"title"`
	Tone  boardcatalog.Tone `json:"tone,omitempty"`
	DueAt *int64            `json:"dueAt,omitempty"`
}

type WorkRow struct {
	Row
	Signals []WorkSignal      `json:"signals"`
	Tone    boardcatalog.Tone `json:"tone,omitempty"`
	DueAt   *int64            `json:"dueAt,omitempty"`
}

type FilledAction struct {
	Label   string `json:"label"`
	Ask     string `json:"ask"`
	Records []Row  `json:"records,omitempty"`
}

type FilledFigure struct {
	Label string              `json:"label"`
	Value *boardcatalog.Value `json:"value,omitempty"`
	Was   *boardcatalog.Value `json:"was,omitempty"`
	Tone  boardcatalog.Tone   `json:"tone,omitempty"`
}

type CompareLine struct {
	Label  string                `json:"label"`
	Better Better                `json:"better,omitempty"`
	Cells  []*boardcatalog.Value `json:"cells"`
	Wins   []bool                `json:"wins"`
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
	Principal string         `json:"principal,omitempty"`
	Comment   string         `json:"comment,omitempty"`
	Changes   map[string]any `json:"changes,omitempty"`
}

type FilledCheck struct {
	Title   string              `json:"title"`
	Tone    boardcatalog.Tone   `json:"tone"`
	Detail  string              `json:"detail,omitempty"`
	Count   *boardcatalog.Value `json:"count,omitempty"`
	Was     *boardcatalog.Value `json:"was,omitempty"`
	Records []Row               `json:"records,omitempty"`
	Hidden  int                 `json:"hidden,omitempty"`
}

type FilledMeter struct {
	Clear   int                 `json:"clear"`
	Total   int                 `json:"total"`
	Tones   []boardcatalog.Tone `json:"tones,omitempty"`
	Percent *boardcatalog.Value `json:"percent,omitempty"`
}

type FilledSegment struct {
	Text     string            `json:"text"`
	Numeral  int               `json:"numeral,omitempty"`
	Tone     boardcatalog.Tone `json:"tone,omitempty"`
	Struck   bool              `json:"struck,omitempty"`
	Reading  string            `json:"reading,omitempty"`
	Artifact string            `json:"artifact,omitempty"`
}

type FilledReading struct {
	Key       string              `json:"key"`
	Label     string              `json:"label"`
	Value     *boardcatalog.Value `json:"value,omitempty"`
	Was       *boardcatalog.Value `json:"was,omitempty"`
	Source    Source              `json:"source"`
	Measure   Measure             `json:"measure,omitempty"`
	Aggregate Aggregate           `json:"aggregate,omitempty"`
}

type FilledBlock struct {
	Type     BlockType         `json:"type"`
	Title    string            `json:"title,omitempty"`
	Half     bool              `json:"half,omitempty"`
	Numeral  int               `json:"numeral,omitempty"`
	State    BlockState        `json:"state"`
	Tone     boardcatalog.Tone `json:"tone,omitempty"`
	Source   Source            `json:"source"`
	Figures  []FilledFigure    `json:"figures,omitempty"`
	Measure  Measure           `json:"measure,omitempty"`
	Rows     []Row             `json:"rows,omitempty"`
	Compare  *FilledCompare    `json:"compare,omitempty"`
	Table    *FilledTable      `json:"table,omitempty"`
	Metric   *FilledMetric     `json:"metric,omitempty"`
	Events   []Event           `json:"events,omitempty"`
	Checks   []FilledCheck     `json:"checks,omitempty"`
	Meter    *FilledMeter      `json:"meter,omitempty"`
	Text     []FilledSegment   `json:"text,omitempty"`
	NoteTone boardcatalog.Tone `json:"noteTone,omitempty"`
	Chips    []Row             `json:"chips,omitempty"`
	Work     []WorkRow         `json:"work,omitempty"`
	Actions  []FilledAction    `json:"actions,omitempty"`
	Caption  []FilledSegment   `json:"caption,omitempty"`
}

type FilledMetric struct {
	Name string                   `json:"name,omitempty"`
	Form boardcatalog.Form        `json:"form"`
	Read *boardcatalog.MetricRead `json:"read"`
	Rows []Row                    `json:"rows,omitempty"`
}

func (m *FilledMetric) Row(ref *boardcatalog.Ref) *Row {
	if m == nil || ref == nil {
		return nil
	}
	for idx := range m.Rows {
		if m.Rows[idx].Kind == ref.Kind && m.Rows[idx].ID == ref.ID {
			return &m.Rows[idx]
		}
	}

	return nil
}

type Filled struct {
	Title    string          `json:"title"`
	Place    Place           `json:"place"`
	Verdict  []FilledSegment `json:"verdict"`
	Readings []FilledReading `json:"readings"`
	Blocks   []FilledBlock   `json:"blocks"`
	ReadAt   int64           `json:"readAt"`
}

type Snapshot struct {
	At     int64                         `json:"at"`
	Values map[string]boardcatalog.Value `json:"values"`
}

func FigureKey(block, figure int) string {
	return "f." + strconv.Itoa(block) + "." + strconv.Itoa(figure)
}

func CheckKey(block, check int) string {
	return "c." + strconv.Itoa(block) + "." + strconv.Itoa(check)
}

func (f *Filled) Snapshot() Snapshot {
	snapshot := Snapshot{At: f.ReadAt, Values: make(map[string]boardcatalog.Value, 8)}
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
	for idx := range f.Readings {
		if value := f.Readings[idx].Value; value != nil {
			snapshot.Values[ReadingKey(f.Readings[idx].Key)] = *value
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
	for idx := range f.Readings {
		reading := &f.Readings[idx]
		if was, kept := snapshot.Values[ReadingKey(reading.Key)]; kept && reading.Value != nil &&
			!reading.Value.Same(&was) {
			reading.Was = &was
		}
	}
	f.WriteReadings()
}
