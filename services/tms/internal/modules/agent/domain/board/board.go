package board

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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
	MaxFigures          = 4
	MaxChecks           = 12
	MaxCheckRecords     = 25
	MaxNoteRecords      = 8
	MaxTimelineEntries  = 12
	DefaultTimeline     = 8
	MaxTableRows        = 10
	DefaultTableRows    = 6
	MaxTitleRunes       = 80
	MaxVerdictRunes     = 320
	MaxBlockTitleRunes  = 60
	MaxProseRunes       = 600
	MaxNoteTitleRunes   = 80
	MaxNoteTextRunes    = 400
	MaxCheckTitleRunes  = 80
	MaxCheckDetailRunes = 160
	MaxLabelRunes       = 40
	MaxColumnRunes      = 120
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
)

func AllBlockTypes() []BlockType {
	return []BlockType{
		BlockFigures, BlockRecords, BlockCompare, BlockTable, BlockChart, BlockTimeline,
		BlockChecks, BlockMeter, BlockProse, BlockNote,
	}
}

func (t BlockType) IsValid() bool {
	return slices.Contains(AllBlockTypes(), t)
}

func (t BlockType) Words() bool {
	return t == BlockProse || t == BlockNote
}

type Tone string

const (
	ToneBad  = Tone("bad")
	ToneWarn = Tone("warn")
	ToneInfo = Tone("info")
	ToneOK   = Tone("ok")
)

func AllTones() []Tone {
	return []Tone{ToneBad, ToneWarn, ToneInfo, ToneOK}
}

func (t Tone) IsValid() bool {
	return slices.Contains(AllTones(), t)
}

func (t Tone) Severity() int {
	switch t {
	case ToneBad:
		return 3
	case ToneWarn:
		return 2
	case ToneInfo:
		return 1
	default:
		return 0
	}
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
	Resource permission.Resource `json:"resource"`
	IDs      []pulid.ID          `json:"ids"`
}

func (s *RecordSet) Empty() bool {
	return s == nil || len(s.IDs) == 0
}

type FigureSource struct {
	Label     string     `json:"label"`
	Metric    string     `json:"metric,omitempty"`
	Records   *RecordSet `json:"records,omitempty"`
	Measure   Measure    `json:"measure,omitempty"`
	Aggregate Aggregate  `json:"aggregate,omitempty"`
	Analysis  pulid.ID   `json:"analysis,omitempty"`
	Index     int        `json:"index,omitempty"`
}

type CompareRow struct {
	Column string `json:"column"`
	Label  string `json:"label,omitempty"`
	Better Better `json:"better,omitempty"`
}

type Check struct {
	Title   string     `json:"title"`
	Tone    Tone       `json:"tone"`
	Detail  string     `json:"detail,omitempty"`
	Records *RecordSet `json:"records,omitempty"`
	Metric  string     `json:"metric,omitempty"`
}

type Block struct {
	Type     BlockType      `json:"type"`
	Title    string         `json:"title,omitempty"`
	Half     bool           `json:"half,omitempty"`
	Figures  []FigureSource `json:"figures,omitempty"`
	Records  *RecordSet     `json:"records,omitempty"`
	Measure  Measure        `json:"measure,omitempty"`
	Order    Order          `json:"order,omitempty"`
	Analysis pulid.ID       `json:"analysis,omitempty"`
	Artifact pulid.ID       `json:"artifact,omitempty"`
	Index    int            `json:"index,omitempty"`
	Key      string         `json:"key,omitempty"`
	Rows     []CompareRow   `json:"rows,omitempty"`
	Limit    int            `json:"limit,omitempty"`
	Checks   []Check        `json:"checks,omitempty"`
	Of       int            `json:"of,omitempty"`
	Metric   string         `json:"metric,omitempty"`
	Text     string         `json:"text,omitempty"`
	Tone     Tone           `json:"tone,omitempty"`
}

type Board struct {
	Title   string  `json:"title"`
	Verdict string  `json:"verdict"`
	Blocks  []Block `json:"blocks"`
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

func (b *Board) Records() map[permission.Resource][]pulid.ID {
	refs := make(map[permission.Resource][]pulid.ID, 4)
	add := func(set *RecordSet) {
		if set.Empty() {
			return
		}
		for _, id := range set.IDs {
			if !slices.Contains(refs[set.Resource], id) {
				refs[set.Resource] = append(refs[set.Resource], id)
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
	}

	return refs
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

	return ids
}

func (b *Board) Metrics() []string {
	keys := make([]string, 0, 2)
	add := func(key string) {
		if key != "" && !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	for idx := range b.Blocks {
		block := &b.Blocks[idx]
		add(block.Metric)
		for figure := range block.Figures {
			add(block.Figures[figure].Metric)
		}
		for check := range block.Checks {
			add(block.Checks[check].Metric)
		}
	}

	return keys
}
