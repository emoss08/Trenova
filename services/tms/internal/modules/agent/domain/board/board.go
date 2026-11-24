package board

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxBlocks           = 8
	MaxNotes            = 2
	MaxBoardRecords     = 120
	MaxListRecords      = 25
	MinCompareRecords   = 2
	MaxCompareRecords   = 4
	MaxCompareRows      = 12
	MaxFigures          = 8
	MaxChecks           = 12
	MaxCheckRecords     = 25
	MaxNoteRecords      = 8
	MaxTimelineEntries  = 12
	DefaultTimeline     = 8
	MaxTableRows        = 10
	DefaultTableRows    = 6
	MaxTitleRunes       = 80
	MaxVerdictRunes     = 600
	MaxBlockTitleRunes  = 60
	MaxCaptionRunes     = 300
	MaxProseRunes       = 600
	MaxNoteTitleRunes   = 80
	MaxNoteTextRunes    = 400
	MaxCheckTitleRunes  = 80
	MaxCheckDetailRunes = 160
	MaxLabelRunes       = 40
	MaxColumnRunes      = 120
	MaxWorkRecords      = 12
	MaxActions          = 3
	MaxActionLabelRunes = 40
	MaxActionAskRunes   = 400
	MaxActionRecords    = 25
)

type BlockType string

const (
	BlockFigures  = BlockType("figures")
	BlockRecords  = BlockType("records")
	BlockCompare  = BlockType("compare")
	BlockTable    = BlockType("table")
	BlockChart    = BlockType("chart")
	BlockTimeline = BlockType("timeline")
	BlockChecks   = BlockType("checks")
	BlockMeter    = BlockType("meter")
	BlockProse    = BlockType("prose")
	BlockNote     = BlockType("note")
	BlockWork     = BlockType("work")
	BlockActions  = BlockType("actions")
	BlockMetric   = BlockType("metric")
)

func AllBlockTypes() []BlockType {
	return []BlockType{
		BlockFigures, BlockRecords, BlockCompare, BlockTable, BlockChart, BlockTimeline,
		BlockChecks, BlockMeter, BlockProse, BlockNote, BlockWork, BlockActions, BlockMetric,
	}
}

func (t BlockType) IsValid() bool {
	return slices.Contains(AllBlockTypes(), t)
}

func (t BlockType) Words() bool {
	return t == BlockProse || t == BlockNote || t == BlockActions
}

type Order string

const (
	OrderGiven   = Order("given")
	OrderMeasure = Order("measure")
)

type Aggregate string

const (
	AggregateCount = Aggregate("count")
	AggregateSum   = Aggregate("sum")
	AggregateMax   = Aggregate("max")
)

func AllAggregates() []Aggregate {
	return []Aggregate{AggregateCount, AggregateSum, AggregateMax}
}

type Better string

const (
	BetterHigher = Better("higher")
	BetterLower  = Better("lower")
)

type RecordSet struct {
	Kind recordkind.Kind `json:"kind"`
	IDs  []pulid.ID      `json:"ids"`
}

func (s *RecordSet) Empty() bool {
	return s == nil || len(s.IDs) == 0
}

type MetricAsk struct {
	Metric string   `json:"metric"`
	For    pulid.ID `json:"for,omitempty"`
	Period string   `json:"period,omitempty"`
}

func AskOf(metric string, forID pulid.ID, period string) MetricAsk {
	if metric == "" {
		return MetricAsk{}
	}
	if period == "" {
		if spec, ok := boardregistry.Catalog().MetricNamed(metric); ok {
			period = string(spec.DefaultPeriod())
		}
	}

	return MetricAsk{Metric: metric, For: forID, Period: period}
}

type FigureSource struct {
	Label     string     `json:"label"`
	Metric    string     `json:"metric,omitempty"`
	Figure    string     `json:"figure,omitempty"`
	For       pulid.ID   `json:"for,omitempty"`
	Period    string     `json:"period,omitempty"`
	Records   *RecordSet `json:"records,omitempty"`
	Measure   Measure    `json:"measure,omitempty"`
	Aggregate Aggregate  `json:"aggregate,omitempty"`
	Analysis  pulid.ID   `json:"analysis,omitempty"`
	Index     int        `json:"index,omitempty"`
	List      pulid.ID   `json:"list,omitempty"`
}

func (f *FigureSource) Ask() MetricAsk {
	return AskOf(f.Metric, f.For, f.Period)
}

type CompareRow struct {
	Column string `json:"column"`
	Label  string `json:"label,omitempty"`
	Better Better `json:"better,omitempty"`
}

type Check struct {
	Title   string            `json:"title"`
	Tone    boardcatalog.Tone `json:"tone"`
	Detail  string            `json:"detail,omitempty"`
	Records *RecordSet        `json:"records,omitempty"`
	Metric  string            `json:"metric,omitempty"`
	Figure  string            `json:"figure,omitempty"`
}

func (c *Check) Ask() MetricAsk {
	return AskOf(c.Metric, pulid.Nil, "")
}

type Action struct {
	Label   string     `json:"label"`
	Ask     string     `json:"ask"`
	Records *RecordSet `json:"records,omitempty"`
}

type Block struct {
	Type     BlockType         `json:"type"`
	Title    string            `json:"title,omitempty"`
	Half     bool              `json:"half,omitempty"`
	Figures  []FigureSource    `json:"figures,omitempty"`
	Records  *RecordSet        `json:"records,omitempty"`
	Measure  Measure           `json:"measure,omitempty"`
	Order    Order             `json:"order,omitempty"`
	Analysis pulid.ID          `json:"analysis,omitempty"`
	Artifact pulid.ID          `json:"artifact,omitempty"`
	Index    int               `json:"index,omitempty"`
	Key      string            `json:"key,omitempty"`
	Rows     []CompareRow      `json:"rows,omitempty"`
	Limit    int               `json:"limit,omitempty"`
	Checks   []Check           `json:"checks,omitempty"`
	Of       int               `json:"of,omitempty"`
	Metric   string            `json:"metric,omitempty"`
	Figure   string            `json:"figure,omitempty"`
	For      pulid.ID          `json:"for,omitempty"`
	Period   string            `json:"period,omitempty"`
	Text     string            `json:"text,omitempty"`
	Tone     boardcatalog.Tone `json:"tone,omitempty"`
	Actions  []Action          `json:"actions,omitempty"`
	Caption  string            `json:"caption,omitempty"`
}

type Board struct {
	Title    string    `json:"title"`
	Verdict  string    `json:"verdict"`
	Place    Place     `json:"place,omitempty"`
	Readings []Reading `json:"readings,omitempty"`
	Blocks   []Block   `json:"blocks"`
}

func (b *Board) Numerals() []int {
	numerals := make([]int, len(b.Blocks))
	next := 0
	for idx := range b.Blocks {
		if b.Blocks[idx].Type.Words() {
			continue
		}
		next++
		numerals[idx] = next
	}

	return numerals
}

func (b *Block) Ask() MetricAsk {
	return AskOf(b.Metric, b.For, b.Period)
}

func (b *Board) Records() map[recordkind.Kind][]pulid.ID {
	refs := make(map[recordkind.Kind][]pulid.ID, 4)
	add := func(set *RecordSet) {
		if set.Empty() {
			return
		}
		for _, id := range set.IDs {
			if !slices.Contains(refs[set.Kind], id) {
				refs[set.Kind] = append(refs[set.Kind], id)
			}
		}
	}
	for idx := range b.Blocks {
		block := &b.Blocks[idx]
		add(block.Records)
		for figure := range block.Figures {
			add(block.Figures[figure].Records)
		}
		for check := range block.Checks {
			add(block.Checks[check].Records)
		}
		for action := range block.Actions {
			add(block.Actions[action].Records)
		}
	}
	for idx := range b.Readings {
		add(b.Readings[idx].Records)
	}

	return refs
}

func (b *Board) WorkRecords() []pulid.ID {
	ids := make([]pulid.ID, 0, MaxWorkRecords)
	for idx := range b.Blocks {
		block := &b.Blocks[idx]
		if block.Type != BlockWork || block.Records.Empty() {
			continue
		}
		for _, id := range block.Records.IDs {
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}

	return ids
}

func (b *Board) Artifacts() []pulid.ID {
	ids := make([]pulid.ID, 0, 2)
	add := func(id pulid.ID) {
		if id.IsNotNil() && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	for idx := range b.Blocks {
		block := &b.Blocks[idx]
		add(block.Analysis)
		add(block.Artifact)
		for figure := range block.Figures {
			add(block.Figures[figure].Analysis)
		}
	}
	for idx := range b.Readings {
		add(b.Readings[idx].Analysis)
	}

	return ids
}

func (b *Board) Lists() []pulid.ID {
	ids := make([]pulid.ID, 0, 1)
	add := func(id pulid.ID) {
		if id.IsNotNil() && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	for idx := range b.Blocks {
		for figure := range b.Blocks[idx].Figures {
			add(b.Blocks[idx].Figures[figure].List)
		}
	}
	for idx := range b.Readings {
		add(b.Readings[idx].List)
	}

	return ids
}

func (b *Board) Metrics() []MetricAsk {
	asks := make([]MetricAsk, 0, 2)
	add := func(ask MetricAsk) {
		if ask.Metric != "" && !slices.Contains(asks, ask) {
			asks = append(asks, ask)
		}
	}
	for idx := range b.Blocks {
		block := &b.Blocks[idx]
		if block.Type == BlockMetric || block.Type == BlockMeter {
			add(block.Ask())
		}
		for figure := range block.Figures {
			add(block.Figures[figure].Ask())
		}
		for check := range block.Checks {
			add(block.Checks[check].Ask())
		}
	}
	for idx := range b.Readings {
		add(b.Readings[idx].Ask())
	}

	return asks
}

func (b *Board) Measures() map[recordkind.Kind][]Measure {
	out := make(map[recordkind.Kind][]Measure, 2)
	add := func(set *RecordSet, measure Measure) {
		if set.Empty() || measure == "" || slices.Contains(out[set.Kind], measure) {
			return
		}
		out[set.Kind] = append(out[set.Kind], measure)
	}
	for idx := range b.Blocks {
		block := &b.Blocks[idx]
		if block.Type == BlockRecords {
			add(block.Records, block.Measure)
		}
		for figure := range block.Figures {
			add(block.Figures[figure].Records, block.Figures[figure].Measure)
		}
	}
	for idx := range b.Readings {
		add(b.Readings[idx].Records, b.Readings[idx].Measure)
	}

	return out
}
