package board

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type Summary struct {
	Label    string
	Subtitle string
	Status   string
	Measures map[Measure]RowMeasure
}

type AnalysisRead struct {
	Title   string
	Display analysis.Display
}

type TableRead struct {
	Title string
	Table analysis.Table
}

type Reads struct {
	Now       int64
	Readable  map[permission.Resource]bool
	Summaries map[permission.Resource]map[pulid.ID]Summary
	Metrics   map[string]*Value
	Analyses  map[pulid.ID]*AnalysisRead
	Tables    map[pulid.ID]*TableRead
	Events    map[pulid.ID][]Event
}

func (r *Reads) summary(resource permission.Resource, id pulid.ID) (Summary, bool) {
	summary, ok := r.Summaries[resource][id]

	return summary, ok && r.Readable[resource]
}

func Assemble(board *Board, reads *Reads) *Filled {
	numerals := board.Numerals()
	filled := &Filled{
		Title:  board.Title,
		Blocks: make([]FilledBlock, 0, len(board.Blocks)),
		ReadAt: reads.Now,
	}
	for idx := range board.Blocks {
		block := assembleBlock(board, idx, reads)
		block.Numeral = numerals[idx]
		filled.Blocks = append(filled.Blocks, block)
	}
	filled.Verdict = filled.segments(board.Verdict)
	for idx := range board.Blocks {
		if board.Blocks[idx].Type == BlockProse {
			filled.Blocks[idx].Text = filled.segments(board.Blocks[idx].Text)
		}
	}

	return filled
}

func (f *Filled) segments(text string) []FilledSegment {
	raw := Segments(text)
	out := make([]FilledSegment, 0, len(raw))
	for _, segment := range raw {
		shown := FilledSegment{Text: segment.Text}
		if segment.Cite > 0 && segment.Cite <= len(f.Blocks) {
			cited := &f.Blocks[segment.Cite-1]
			shown.Numeral = cited.Numeral
			shown.Struck = cited.State == StateUnreadable
			shown.Tone = ToneInfo
			if cited.Tone.Severity() >= ToneWarn.Severity() {
				shown.Tone = ToneWarn
			}
		}
		out = append(out, shown)
	}

	return out
}

func assembleBlock(board *Board, idx int, reads *Reads) FilledBlock {
	spec := &board.Blocks[idx]
	block := FilledBlock{Type: spec.Type, Title: spec.Title, Half: spec.Half, State: StateReady}
	switch spec.Type {
	case BlockFigures:
		assembleFigures(&block, spec, reads)
	case BlockRecords:
		assembleRecords(&block, spec, reads)
	case BlockCompare:
		assembleCompare(&block, spec, reads)
	case BlockTable:
		assembleTable(&block, spec, reads)
	case BlockChart:
		assembleChart(&block, spec, reads)
	case BlockTimeline:
		assembleTimeline(&block, spec, reads)
	case BlockChecks:
		assembleChecks(&block, spec, reads)
	case BlockMeter:
		assembleMeter(&block, board, spec, reads)
	case BlockProse:
	case BlockNote:
		block.NoteTone = spec.Tone
		block.Text = []FilledSegment{{Text: PlainText(spec.Text)}}
		if !spec.Records.Empty() {
			block.Chips, _ = rowsOf(spec.Records, reads)
		}
	}

	return block
}

func assembleFigures(block *FilledBlock, spec *Block, reads *Reads) {
	block.Figures = make([]FilledFigure, 0, len(spec.Figures))
	kinds := make([]SourceKind, 0, len(spec.Figures))
	for idx := range spec.Figures {
		source := &spec.Figures[idx]
		figure := FilledFigure{Label: source.Label}
		switch {
		case source.Metric != "":
			figure.Value = reads.Metrics[source.Metric]
			kinds = append(kinds, SourceMetrics)
		case source.Records != nil:
			figure.Value, figure.Tone = aggregate(source, reads)
			kinds = append(kinds, SourceRecords)
			block.Source.Resource = source.Records.Resource
			block.Source.Count = max(block.Source.Count, len(source.Records.IDs))
		case source.Analysis.IsNotNil():
			figure.Value = analysisFigure(reads.Analyses[source.Analysis], source.Index)
			kinds = append(kinds, SourceAnalysis)
			if read := reads.Analyses[source.Analysis]; read != nil {
				block.Source.Title = read.Title
			}
		}
		if figure.Tone.Severity() > block.Tone.Severity() {
			block.Tone = figure.Tone
		}
		block.Figures = append(block.Figures, figure)
	}
	block.Source.Kind = commonKind(kinds)
	if !slices.ContainsFunc(block.Figures, func(f FilledFigure) bool { return f.Value != nil }) {
		block.State = StateUnreadable
	}
}

func commonKind(kinds []SourceKind) SourceKind {
	if len(kinds) == 0 {
		return SourceNone
	}
	for _, kind := range kinds[1:] {
		if kind != kinds[0] {
			return SourceMixed
		}
	}

	return kinds[0]
}

func aggregate(source *FigureSource, reads *Reads) (value *Value, tone Tone) {
	if !reads.Readable[source.Records.Resource] {
		return nil, ""
	}
	values, readable := measured(source, reads)
	if source.Aggregate == AggregateSum || source.Aggregate == AggregateMax {
		value = combined(source, values)
	} else {
		value = counted(source, values, readable)
	}
	if value == nil {
		return nil, ""
	}
	if source.Measure.Adverse() && value.Number.IsPositive() {
		return value, ToneWarn
	}

	return value, ""
}

func measured(source *FigureSource, reads *Reads) (values []RowMeasure, readable int) {
	set := source.Records
	values = make([]RowMeasure, 0, len(set.IDs))
	for _, id := range set.IDs {
		summary, ok := reads.summary(set.Resource, id)
		if !ok {
			continue
		}
		readable++
		if measure, has := summary.Measures[source.Measure]; has &&
			measure.State != MeasureProjected {
			values = append(values, measure)
		}
	}

	return values, readable
}

func combined(source *FigureSource, values []RowMeasure) *Value {
	if len(values) == 0 {
		return &Value{Kind: source.Measure.Kind(), Number: decimal.Zero}
	}
	currency := values[0].Value.Currency
	total := values[0].Value.Number
	for _, measure := range values[1:] {
		if measure.Value.Currency != currency {
			return nil
		}
		if source.Aggregate == AggregateSum {
			total = total.Add(measure.Value.Number)
		} else if measure.Value.Number.GreaterThan(total) {
			total = measure.Value.Number
		}
	}

	return &Value{Kind: source.Measure.Kind(), Number: total, Currency: currency}
}

func counted(source *FigureSource, values []RowMeasure, readable int) *Value {
	count := readable
	if source.Measure != "" {
		count = 0
		for _, measure := range values {
			if measure.Value.Number.IsPositive() {
				count++
			}
		}
	}

	return &Value{Kind: ValueCount, Number: decimal.NewFromInt(int64(count))}
}

func analysisFigure(read *AnalysisRead, index int) *Value {
	if read == nil || index < 0 || index >= len(read.Display.Figures) {
		return nil
	}
	figure := &read.Display.Figures[index]

	return valueOf(figure.Value, figure.Kind, figure.Currency)
}

func valueOf(raw any, kind analysis.ValueKind, currency string) *Value {
	var number decimal.Decimal
	switch typed := raw.(type) {
	case float64:
		number = decimal.NewFromFloat(typed)
	case int:
		number = decimal.NewFromInt(int64(typed))
	case int64:
		number = decimal.NewFromInt(typed)
	case nil:
		return nil
	default:
		return &Value{Kind: ValueText, Text: fmt.Sprint(typed)}
	}
	switch kind {
	case analysis.ValueMoney:
		return &Value{Kind: ValueMoney, Number: number, Currency: currency}
	case analysis.ValueCount:
		return &Value{Kind: ValueCount, Number: number}
	case analysis.ValuePercent:
		return &Value{Kind: ValuePercent, Number: number}
	case analysis.ValueText, analysis.ValueNumber, analysis.ValueDate, analysis.ValueDateTime,
		analysis.ValueBool:
	}

	return &Value{Kind: ValueNumber, Number: number}
}

func rowsOf(set *RecordSet, reads *Reads) (rows []Row, readable int) {
	rows = make([]Row, 0, len(set.IDs))
	for _, id := range set.IDs {
		row := Row{Resource: set.Resource, ID: id}
		if summary, ok := reads.summary(set.Resource, id); ok {
			readable++
			row.Readable = true
			row.Label = summary.Label
			row.Subtitle = summary.Subtitle
			row.Status = summary.Status
		}
		rows = append(rows, row)
	}

	return rows, readable
}

func assembleRecords(block *FilledBlock, spec *Block, reads *Reads) {
	set := spec.Records
	block.Measure = spec.Measure
	block.Source = Source{Kind: SourceRecords, Resource: set.Resource, Count: len(set.IDs)}
	if !reads.Readable[set.Resource] {
		block.State = StateUnreadable
		return
	}
	rows, readable := rowsOf(set, reads)
	block.Source.Readable = readable
	for idx := range rows {
		if !rows[idx].Readable || spec.Measure == "" {
			continue
		}
		summary, _ := reads.summary(set.Resource, rows[idx].ID)
		if measure, measured := summary.Measures[spec.Measure]; measured {
			rows[idx].Measure = &measure
			if measure.Tone.Severity() > block.Tone.Severity() {
				block.Tone = measure.Tone
			}
		}
	}
	if spec.Order == OrderMeasure {
		slices.SortStableFunc(rows, func(a, b Row) int {
			return compareRows(&a, &b)
		})
	}
	block.Rows = rows
}

func compareRows(a, b *Row) int {
	switch {
	case a.Readable != b.Readable:
		if a.Readable {
			return -1
		}
		return 1
	case a.Measure == nil || b.Measure == nil:
		if a.Measure != nil {
			return -1
		}
		if b.Measure != nil {
			return 1
		}
		return 0
	default:
		return b.Measure.Value.Number.Cmp(a.Measure.Value.Number)
	}
}

func assembleCompare(block *FilledBlock, spec *Block, reads *Reads) {
	block.Source = Source{
		Kind:     SourceAnalysis,
		Resource: spec.Records.Resource,
		Count:    len(spec.Records.IDs),
	}
	read := reads.Analyses[spec.Analysis]
	if read == nil || spec.Index >= len(read.Display.Tables) ||
		!reads.Readable[spec.Records.Resource] {
		block.State = StateUnreadable
		return
	}
	block.Source.Title = read.Title
	table := &read.Display.Tables[spec.Index]
	key := columnIndex(table, spec.Key)
	columns, readable := rowsOf(spec.Records, reads)
	block.Source.Readable = readable
	if key < 0 || readable == 0 {
		block.State = StateUnreadable
		return
	}

	byRecord := make(map[pulid.ID][]any, len(columns))
	for _, row := range table.Rows {
		if key >= len(row) {
			continue
		}
		if id, err := pulid.Parse(strings.TrimSpace(fmt.Sprint(row[key]))); err == nil {
			byRecord[id] = row
		}
	}

	compare := &FilledCompare{Columns: columns, Wins: make([]int, len(columns))}
	for _, want := range spec.Rows {
		col := columnIndex(table, want.Column)
		if col < 0 {
			continue
		}
		column := &table.Columns[col]
		line := CompareLine{
			Label:  cmp.Or(want.Label, column.Name),
			Better: want.Better,
			Cells:  make([]*Value, len(columns)),
			Wins:   make([]bool, len(columns)),
		}
		for idx := range columns {
			row, found := byRecord[columns[idx].ID]
			if !columns[idx].Readable || !found || col >= len(row) {
				continue
			}
			line.Cells[idx] = valueOf(row[col], column.Kind, column.Currency)
		}
		markWinners(&line, compare.Wins)
		compare.Lines = append(compare.Lines, line)
	}
	block.Compare = compare
}

func markWinners(line *CompareLine, wins []int) {
	if line.Better == "" {
		return
	}
	var best *decimal.Decimal
	for _, cell := range line.Cells {
		if cell == nil || cell.Kind == ValueText {
			continue
		}
		if best == nil ||
			(line.Better == BetterHigher && cell.Number.GreaterThan(*best)) ||
			(line.Better == BetterLower && cell.Number.LessThan(*best)) {
			number := cell.Number
			best = &number
		}
	}
	if best == nil {
		return
	}
	for idx, cell := range line.Cells {
		if cell != nil && cell.Kind != ValueText && cell.Number.Equal(*best) {
			line.Wins[idx] = true
			wins[idx]++
		}
	}
}

func columnIndex(table *analysis.Table, name string) int {
	name = strings.TrimSpace(name)
	for idx := range table.Columns {
		if strings.EqualFold(strings.TrimSpace(table.Columns[idx].Name), name) {
			return idx
		}
	}

	return -1
}

func assembleTable(block *FilledBlock, spec *Block, reads *Reads) {
	var (
		table analysis.Table
		title string
	)
	switch {
	case spec.Artifact.IsNotNil():
		block.Source = Source{Kind: SourceTable}
		read := reads.Tables[spec.Artifact]
		if read == nil {
			block.State = StateUnreadable
			return
		}
		table, title = read.Table, read.Title
	default:
		block.Source = Source{Kind: SourceAnalysis}
		read := reads.Analyses[spec.Analysis]
		if read == nil || spec.Index >= len(read.Display.Tables) {
			block.State = StateUnreadable
			return
		}
		table, title = read.Display.Tables[spec.Index], read.Title
	}
	block.Source.Title = title
	total := max(table.TotalRows, len(table.Rows))
	if len(table.Rows) > spec.Limit {
		table.Rows = table.Rows[:spec.Limit]
	}
	table.TotalRows = total
	block.Table = &FilledTable{Table: table, Total: total, Artifact: spec.Artifact}
	if spec.Artifact.IsNil() {
		block.Table.Artifact = spec.Analysis
	}
	if total == 0 {
		block.State = StateEmpty
	}
}

func assembleChart(block *FilledBlock, spec *Block, reads *Reads) {
	block.Source = Source{Kind: SourceAnalysis}
	read := reads.Analyses[spec.Analysis]
	if read == nil || spec.Index >= len(read.Display.Charts) {
		block.State = StateUnreadable
		return
	}
	block.Source.Title = read.Title
	chart := read.Display.Charts[spec.Index]
	block.Chart = &chart
	if len(chart.X.Values) == 0 || len(chart.Series) == 0 {
		block.State = StateEmpty
	}
}

func assembleTimeline(block *FilledBlock, spec *Block, reads *Reads) {
	set := spec.Records
	block.Source = Source{Kind: SourceActivity, Resource: set.Resource, Count: 1}
	rows, readable := rowsOf(set, reads)
	events, read := reads.Events[set.IDs[0]]
	if readable == 0 || !read {
		block.State = StateUnreadable
		return
	}
	block.Rows = rows
	block.Source.Readable = readable
	if len(events) > spec.Limit {
		events = events[len(events)-spec.Limit:]
	}
	block.Events = events
	if len(events) == 0 {
		block.State = StateEmpty
	}
}

func assembleChecks(block *FilledBlock, spec *Block, reads *Reads) {
	block.Source = Source{Kind: SourceChecks}
	kinds := make([]SourceKind, 0, len(spec.Checks))
	block.Checks = make([]FilledCheck, 0, len(spec.Checks))
	for idx := range spec.Checks {
		source := &spec.Checks[idx]
		check := FilledCheck{Title: source.Title, Tone: source.Tone, Detail: source.Detail}
		switch {
		case source.Metric != "":
			kinds = append(kinds, SourceMetrics)
			check.Count = reads.Metrics[source.Metric]
		case source.Records != nil:
			kinds = append(kinds, SourceRecords)
			block.Source.Resource = source.Records.Resource
			block.Source.Count += len(source.Records.IDs)
			if reads.Readable[source.Records.Resource] {
				rows, readable := rowsOf(source.Records, reads)
				check.Records = slices.DeleteFunc(rows, func(row Row) bool { return !row.Readable })
				check.Hidden = len(source.Records.IDs) - readable
				check.Count = &Value{Kind: ValueCount, Number: decimal.NewFromInt(int64(readable))}
				block.Source.Readable += readable
			}
		}
		if check.Tone.Severity() > block.Tone.Severity() {
			block.Tone = check.Tone
		}
		block.Checks = append(block.Checks, check)
	}
	block.Source.Kind = commonKind(kinds)
	if block.Source.Kind == SourceMixed || block.Source.Kind == SourceMetrics {
		block.Source.Resource = ""
	}
}

func assembleMeter(block *FilledBlock, board *Board, spec *Block, reads *Reads) {
	if spec.Metric != "" {
		block.Source = Source{Kind: SourceMetrics}
		value := reads.Metrics[spec.Metric]
		if value == nil {
			block.State = StateUnreadable
			return
		}
		block.Meter = &FilledMeter{Percent: value}
		return
	}

	block.Source = Source{Kind: SourceChecks}
	if spec.Of < 1 || spec.Of > len(board.Blocks) {
		block.State = StateUnreadable
		return
	}
	checks := board.Blocks[spec.Of-1].Checks
	meter := &FilledMeter{Total: len(checks), Tones: make([]Tone, 0, len(checks))}
	for idx := range checks {
		meter.Tones = append(meter.Tones, checks[idx].Tone)
		if checks[idx].Tone == ToneOK {
			meter.Clear++
		}
	}
	slices.SortStableFunc(meter.Tones, func(a, b Tone) int {
		return cmp.Compare(a.Severity(), b.Severity())
	})
	for _, tone := range meter.Tones {
		if tone.Severity() > block.Tone.Severity() {
			block.Tone = tone
		}
	}
	block.Meter = meter
}
