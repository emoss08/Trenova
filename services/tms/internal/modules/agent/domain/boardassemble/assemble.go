package boardassemble

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type Summary struct {
	Label    string
	Subtitle string
	Status   string
	Measures map[board.Measure]board.RowMeasure
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
	Metrics   map[string]*board.Value
	Analyses  map[pulid.ID]*AnalysisRead
	Tables    map[pulid.ID]*TableRead
	Events    map[pulid.ID][]board.Event
}

func (r *Reads) summary(resource permission.Resource, id pulid.ID) (Summary, bool) {
	summary, ok := r.Summaries[resource][id]

	return summary, ok && r.Readable[resource]
}

func Assemble(layout *board.Board, reads *Reads) *board.Filled {
	numerals := layout.Numerals()
	filled := &board.Filled{
		Title:  layout.Title,
		Blocks: make([]board.FilledBlock, 0, len(layout.Blocks)),
		ReadAt: reads.Now,
	}
	for idx := range layout.Blocks {
		block := assembleBlock(layout, idx, reads)
		block.Numeral = numerals[idx]
		filled.Blocks = append(filled.Blocks, block)
	}
	filled.Verdict = segmentsOf(filled, layout.Verdict)
	for idx := range layout.Blocks {
		if layout.Blocks[idx].Type == board.BlockProse {
			filled.Blocks[idx].Text = segmentsOf(filled, layout.Blocks[idx].Text)
		}
	}

	return filled
}

func segmentsOf(f *board.Filled, text string) []board.FilledSegment {
	raw := board.Segments(text)
	out := make([]board.FilledSegment, 0, len(raw))
	for _, segment := range raw {
		shown := board.FilledSegment{Text: segment.Text}
		if segment.Cite > 0 && segment.Cite <= len(f.Blocks) {
			cited := &f.Blocks[segment.Cite-1]
			shown.Numeral = cited.Numeral
			shown.Struck = cited.State == board.StateUnreadable
			shown.Tone = board.ToneInfo
			if cited.Tone.Severity() >= board.ToneWarn.Severity() {
				shown.Tone = board.ToneWarn
			}
		}
		out = append(out, shown)
	}

	return out
}

func assembleBlock(layout *board.Board, idx int, reads *Reads) board.FilledBlock {
	spec := &layout.Blocks[idx]
	block := board.FilledBlock{
		Type:  spec.Type,
		Title: spec.Title,
		Half:  spec.Half,
		State: board.StateReady,
	}
	switch spec.Type {
	case board.BlockFigures:
		assembleFigures(&block, spec, reads)
	case board.BlockRecords:
		assembleRecords(&block, spec, reads)
	case board.BlockCompare:
		assembleCompare(&block, spec, reads)
	case board.BlockTable:
		assembleTable(&block, spec, reads)
	case board.BlockChart:
		assembleChart(&block, spec, reads)
	case board.BlockTimeline:
		assembleTimeline(&block, spec, reads)
	case board.BlockChecks:
		assembleChecks(&block, spec, reads)
	case board.BlockMeter:
		assembleMeter(&block, layout, spec, reads)
	case board.BlockProse:
	case board.BlockNote:
		block.NoteTone = spec.Tone
		block.Text = []board.FilledSegment{{Text: board.PlainText(spec.Text)}}
		if !spec.Records.Empty() {
			block.Chips, _ = rowsOf(spec.Records, reads)
		}
	}

	return block
}

func assembleFigures(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	block.Figures = make([]board.FilledFigure, 0, len(spec.Figures))
	kinds := make([]board.SourceKind, 0, len(spec.Figures))
	for idx := range spec.Figures {
		source := &spec.Figures[idx]
		figure := board.FilledFigure{Label: source.Label}
		switch {
		case source.Metric != "":
			figure.Value = reads.Metrics[source.Metric]
			kinds = append(kinds, board.SourceMetrics)
		case source.Records != nil:
			figure.Value, figure.Tone = aggregate(source, reads)
			kinds = append(kinds, board.SourceRecords)
			block.Source.Resource = source.Records.Resource
			block.Source.Count = max(block.Source.Count, len(source.Records.IDs))
		case source.Analysis.IsNotNil():
			figure.Value = analysisFigure(reads.Analyses[source.Analysis], source.Index)
			kinds = append(kinds, board.SourceAnalysis)
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
	if !slices.ContainsFunc(
		block.Figures,
		func(f board.FilledFigure) bool { return f.Value != nil },
	) {
		block.State = board.StateUnreadable
	}
}

func commonKind(kinds []board.SourceKind) board.SourceKind {
	if len(kinds) == 0 {
		return board.SourceNone
	}
	for _, kind := range kinds[1:] {
		if kind != kinds[0] {
			return board.SourceMixed
		}
	}

	return kinds[0]
}

func aggregate(source *board.FigureSource, reads *Reads) (value *board.Value, tone board.Tone) {
	if !reads.Readable[source.Records.Resource] {
		return nil, ""
	}
	values, readable := measured(source, reads)
	if source.Aggregate == board.AggregateSum || source.Aggregate == board.AggregateMax {
		value = combined(source, values)
	} else {
		value = counted(source, values, readable)
	}
	if value == nil {
		return nil, ""
	}
	if source.Measure.Adverse() && value.Number.IsPositive() {
		return value, board.ToneWarn
	}

	return value, ""
}

func measured(source *board.FigureSource, reads *Reads) (values []board.RowMeasure, readable int) {
	set := source.Records
	values = make([]board.RowMeasure, 0, len(set.IDs))
	for _, id := range set.IDs {
		summary, ok := reads.summary(set.Resource, id)
		if !ok {
			continue
		}
		readable++
		if measure, has := summary.Measures[source.Measure]; has &&
			measure.State != board.MeasureProjected {
			values = append(values, measure)
		}
	}

	return values, readable
}

func combined(source *board.FigureSource, values []board.RowMeasure) *board.Value {
	if len(values) == 0 {
		return &board.Value{Kind: source.Measure.Kind(), Number: decimal.Zero}
	}
	currency := values[0].Value.Currency
	total := values[0].Value.Number
	for i := 1; i < len(values); i++ {
		measure := &values[i]
		if measure.Value.Currency != currency {
			return nil
		}
		if source.Aggregate == board.AggregateSum {
			total = total.Add(measure.Value.Number)
		} else if measure.Value.Number.GreaterThan(total) {
			total = measure.Value.Number
		}
	}

	return &board.Value{Kind: source.Measure.Kind(), Number: total, Currency: currency}
}

func counted(source *board.FigureSource, values []board.RowMeasure, readable int) *board.Value {
	count := readable
	if source.Measure != "" {
		count = 0
		for i := range values {
			if values[i].Value.Number.IsPositive() {
				count++
			}
		}
	}

	return &board.Value{Kind: board.ValueCount, Number: decimal.NewFromInt(int64(count))}
}

func analysisFigure(read *AnalysisRead, index int) *board.Value {
	if read == nil || index < 0 || index >= len(read.Display.Figures) {
		return nil
	}
	figure := &read.Display.Figures[index]

	return valueOf(figure.Value, figure.Kind, figure.Currency)
}

func valueOf(raw any, kind analysis.ValueKind, currency string) *board.Value {
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
		return &board.Value{Kind: board.ValueText, Text: fmt.Sprint(typed)}
	}
	switch kind {
	case analysis.ValueMoney:
		return &board.Value{Kind: board.ValueMoney, Number: number, Currency: currency}
	case analysis.ValueCount:
		return &board.Value{Kind: board.ValueCount, Number: number}
	case analysis.ValuePercent:
		return &board.Value{Kind: board.ValuePercent, Number: number}
	case analysis.ValueText, analysis.ValueNumber, analysis.ValueDate, analysis.ValueDateTime,
		analysis.ValueBool:
	}

	return &board.Value{Kind: board.ValueNumber, Number: number}
}

func rowsOf(set *board.RecordSet, reads *Reads) (rows []board.Row, readable int) {
	rows = make([]board.Row, 0, len(set.IDs))
	for _, id := range set.IDs {
		row := board.Row{Resource: set.Resource, ID: id}
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

func assembleRecords(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	set := spec.Records
	block.Measure = spec.Measure
	block.Source = board.Source{
		Kind:     board.SourceRecords,
		Resource: set.Resource,
		Count:    len(set.IDs),
	}
	if !reads.Readable[set.Resource] {
		block.State = board.StateUnreadable
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
	if spec.Order == board.OrderMeasure {
		slices.SortStableFunc(rows, func(a, b board.Row) int {
			return compareRows(&a, &b)
		})
	}
	block.Rows = rows
}

func compareRows(a, b *board.Row) int {
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

func assembleCompare(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	block.Source = board.Source{
		Kind:     board.SourceAnalysis,
		Resource: spec.Records.Resource,
		Count:    len(spec.Records.IDs),
	}
	read := reads.Analyses[spec.Analysis]
	if read == nil || spec.Index >= len(read.Display.Tables) ||
		!reads.Readable[spec.Records.Resource] {
		block.State = board.StateUnreadable
		return
	}
	block.Source.Title = read.Title
	table := &read.Display.Tables[spec.Index]
	key := columnIndex(table, spec.Key)
	columns, readable := rowsOf(spec.Records, reads)
	block.Source.Readable = readable
	if key < 0 || readable == 0 {
		block.State = board.StateUnreadable
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

	compare := &board.FilledCompare{Columns: columns, Wins: make([]int, len(columns))}
	for _, want := range spec.Rows {
		col := columnIndex(table, want.Column)
		if col < 0 {
			continue
		}
		column := &table.Columns[col]
		line := board.CompareLine{
			Label:  cmp.Or(want.Label, column.Name),
			Better: want.Better,
			Cells:  make([]*board.Value, len(columns)),
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

func markWinners(line *board.CompareLine, wins []int) {
	if line.Better == "" {
		return
	}
	var best *decimal.Decimal
	for _, cell := range line.Cells {
		if cell == nil || cell.Kind == board.ValueText {
			continue
		}
		if best == nil ||
			(line.Better == board.BetterHigher && cell.Number.GreaterThan(*best)) ||
			(line.Better == board.BetterLower && cell.Number.LessThan(*best)) {
			number := cell.Number
			best = &number
		}
	}
	if best == nil {
		return
	}
	for idx, cell := range line.Cells {
		if cell != nil && cell.Kind != board.ValueText && cell.Number.Equal(*best) {
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

func assembleTable(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	var (
		table analysis.Table
		title string
	)
	switch {
	case spec.Artifact.IsNotNil():
		block.Source = board.Source{Kind: board.SourceTable}
		read := reads.Tables[spec.Artifact]
		if read == nil {
			block.State = board.StateUnreadable
			return
		}
		table, title = read.Table, read.Title
	default:
		block.Source = board.Source{Kind: board.SourceAnalysis}
		read := reads.Analyses[spec.Analysis]
		if read == nil || spec.Index >= len(read.Display.Tables) {
			block.State = board.StateUnreadable
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
	block.Table = &board.FilledTable{Table: table, Total: total, Artifact: spec.Artifact}
	if spec.Artifact.IsNil() {
		block.Table.Artifact = spec.Analysis
	}
	if total == 0 {
		block.State = board.StateEmpty
	}
}

func assembleChart(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	block.Source = board.Source{Kind: board.SourceAnalysis}
	read := reads.Analyses[spec.Analysis]
	if read == nil || spec.Index >= len(read.Display.Charts) {
		block.State = board.StateUnreadable
		return
	}
	block.Source.Title = read.Title
	chart := read.Display.Charts[spec.Index]
	block.Chart = &chart
	if len(chart.X.Values) == 0 || len(chart.Series) == 0 {
		block.State = board.StateEmpty
	}
}

func assembleTimeline(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	set := spec.Records
	block.Source = board.Source{Kind: board.SourceActivity, Resource: set.Resource, Count: 1}
	rows, readable := rowsOf(set, reads)
	events, read := reads.Events[set.IDs[0]]
	if readable == 0 || !read {
		block.State = board.StateUnreadable
		return
	}
	block.Rows = rows
	block.Source.Readable = readable
	if len(events) > spec.Limit {
		events = events[len(events)-spec.Limit:]
	}
	block.Events = events
	if len(events) == 0 {
		block.State = board.StateEmpty
	}
}

func assembleChecks(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	block.Source = board.Source{Kind: board.SourceChecks}
	kinds := make([]board.SourceKind, 0, len(spec.Checks))
	block.Checks = make([]board.FilledCheck, 0, len(spec.Checks))
	for idx := range spec.Checks {
		source := &spec.Checks[idx]
		check := board.FilledCheck{Title: source.Title, Tone: source.Tone, Detail: source.Detail}
		switch {
		case source.Metric != "":
			kinds = append(kinds, board.SourceMetrics)
			check.Count = reads.Metrics[source.Metric]
		case source.Records != nil:
			kinds = append(kinds, board.SourceRecords)
			block.Source.Resource = source.Records.Resource
			block.Source.Count += len(source.Records.IDs)
			if reads.Readable[source.Records.Resource] {
				rows, readable := rowsOf(source.Records, reads)
				check.Records = slices.DeleteFunc(
					rows,
					func(row board.Row) bool { return !row.Readable },
				)
				check.Hidden = len(source.Records.IDs) - readable
				check.Count = &board.Value{
					Kind:   board.ValueCount,
					Number: decimal.NewFromInt(int64(readable)),
				}
				block.Source.Readable += readable
			}
		}
		if check.Tone.Severity() > block.Tone.Severity() {
			block.Tone = check.Tone
		}
		block.Checks = append(block.Checks, check)
	}
	block.Source.Kind = commonKind(kinds)
	if block.Source.Kind == board.SourceMixed || block.Source.Kind == board.SourceMetrics {
		block.Source.Resource = ""
	}
}

func assembleMeter(block *board.FilledBlock, layout *board.Board, spec *board.Block, reads *Reads) {
	if spec.Metric != "" {
		block.Source = board.Source{Kind: board.SourceMetrics}
		value := reads.Metrics[spec.Metric]
		if value == nil {
			block.State = board.StateUnreadable
			return
		}
		block.Meter = &board.FilledMeter{Percent: value}
		return
	}

	block.Source = board.Source{Kind: board.SourceChecks}
	if spec.Of < 1 || spec.Of > len(layout.Blocks) {
		block.State = board.StateUnreadable
		return
	}
	checks := layout.Blocks[spec.Of-1].Checks
	meter := &board.FilledMeter{Total: len(checks), Tones: make([]board.Tone, 0, len(checks))}
	for idx := range checks {
		meter.Tones = append(meter.Tones, checks[idx].Tone)
		if checks[idx].Tone == board.ToneOK {
			meter.Clear++
		}
	}
	slices.SortStableFunc(meter.Tones, func(a, b board.Tone) int {
		return cmp.Compare(a.Severity(), b.Severity())
	})
	for _, tone := range meter.Tones {
		if tone.Severity() > block.Tone.Severity() {
			block.Tone = tone
		}
	}
	block.Meter = meter
}
