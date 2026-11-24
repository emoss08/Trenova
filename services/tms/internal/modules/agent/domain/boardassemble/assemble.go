package boardassemble

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardbar"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardregistry"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type AnalysisRead struct {
	Title   string
	Display analysis.Display
}

type TableRead struct {
	Title string
	Table analysis.Table
}

type ListRead struct {
	Title string
	Kind  recordkind.Kind
	Total *int
}

type Reads struct {
	Now       int64
	Readable  map[recordkind.Kind]bool
	Summaries map[recordkind.Kind]map[pulid.ID]boardreading.Summary
	Readings  map[board.MeasureKey]map[pulid.ID]boardreading.Reading
	Withheld  map[board.MeasureKey]bool
	Metrics   map[board.MetricAsk]*boardcatalog.MetricRead
	Analyses  map[pulid.ID]*AnalysisRead
	Tables    map[pulid.ID]*TableRead
	Lists     map[pulid.ID]*ListRead
	Events    map[pulid.ID][]board.Event
	Work      map[pulid.ID][]board.WorkSignal
}

func (r *Reads) list(id pulid.ID) (*boardcatalog.Value, board.Source) {
	read := r.Lists[id]
	if read == nil {
		return nil, board.Source{Kind: board.SourceTable}
	}
	source := board.Source{Kind: board.SourceTable, Title: read.Title, RecordKind: read.Kind}
	if read.Total == nil {
		return nil, source
	}
	value := boardcatalog.Count(int64(*read.Total))

	return &value, source
}

func (r *Reads) summary(kind recordkind.Kind, id pulid.ID) (boardreading.Summary, bool) {
	summary, ok := r.Summaries[kind][id]

	return summary, ok && r.Readable[kind]
}

func (r *Reads) reading(
	kind recordkind.Kind,
	measure board.Measure,
	id pulid.ID,
) (boardreading.Reading, bool) {
	reading, ok := r.Readings[board.MeasureKey{Kind: kind, Measure: measure}][id]

	return reading, ok
}

func (r *Reads) withheld(kind recordkind.Kind, measure board.Measure) bool {
	return r.Withheld[board.MeasureKey{Kind: kind, Measure: measure}]
}

func (r *Reads) figure(ask board.MetricAsk, name string) *boardcatalog.Value {
	read := r.Metrics[ask]
	if read == nil {
		return nil
	}
	if name == "" {
		spec, ok := boardregistry.Catalog().MetricNamed(ask.Metric)
		if ok && len(spec.Figures) == 1 {
			name = spec.Figures[0].Name
		}
	}

	return read.Figures[name]
}

func Assemble(layout *board.Board, reads *Reads) *board.Filled {
	numerals := layout.Numerals()
	filled := &board.Filled{
		Title:    layout.Title,
		Place:    layout.Placed(),
		Readings: assembleReadings(layout.Readings, reads),
		Blocks:   make([]board.FilledBlock, 0, len(layout.Blocks)),
		ReadAt:   reads.Now,
	}
	for idx := range layout.Blocks {
		block := assembleBlock(layout, idx, reads)
		block.Numeral = numerals[idx]
		filled.Blocks = append(filled.Blocks, block)
	}
	filled.Verdict = segmentsOf(filled, layout.Verdict)
	for idx := range layout.Blocks {
		switch layout.Blocks[idx].Type {
		case board.BlockProse:
			filled.Blocks[idx].Text = segmentsOf(filled, layout.Blocks[idx].Text)
		case board.BlockNote:
			filled.Blocks[idx].Text = segmentsOf(filled, board.PlainText(layout.Blocks[idx].Text))
		case board.BlockFigures, board.BlockRecords, board.BlockCompare, board.BlockTable,
			board.BlockChart, board.BlockTimeline, board.BlockChecks, board.BlockMeter,
			board.BlockWork, board.BlockActions, board.BlockMetric:
		}
		if caption := layout.Blocks[idx].Caption; caption != "" {
			filled.Blocks[idx].Caption = segmentsOf(filled, board.PlainText(caption))
		}
	}
	filled.WriteReadings()

	return filled
}

func assembleReadings(readings []board.Reading, reads *Reads) []board.FilledReading {
	out := make([]board.FilledReading, 0, len(readings))
	for idx := range readings {
		source := &readings[idx].FigureSource
		filled := board.FilledReading{
			Key:       readings[idx].Key,
			Label:     source.Label,
			Measure:   source.Measure,
			Aggregate: source.Aggregate,
		}
		switch {
		case source.Metric != "":
			filled.Value = reads.figure(source.Ask(), source.Figure)
			filled.Source = board.Source{Kind: board.SourceMetrics}
			if read := reads.Metrics[source.Ask()]; read != nil {
				filled.Source.Title = read.Title
			}
		case source.Records != nil:
			filled.Value, _ = aggregate(source, reads)
			filled.Source = board.Source{
				Kind:       board.SourceRecords,
				RecordKind: source.Records.Kind,
				Count:      len(source.Records.IDs),
			}
			if reads.Readable[source.Records.Kind] {
				_, filled.Source.Readable = measured(source, reads)
			}
		case source.Analysis.IsNotNil():
			filled.Value = analysisFigure(reads.Analyses[source.Analysis], source.Index)
			filled.Source = board.Source{Kind: board.SourceAnalysis}
			if read := reads.Analyses[source.Analysis]; read != nil {
				filled.Source.Title = read.Title
			}
		case source.List.IsNotNil():
			filled.Value, filled.Source = reads.list(source.List)
		}
		out = append(out, filled)
	}

	return out
}

func segmentsOf(f *board.Filled, text string) []board.FilledSegment {
	raw := board.Segments(text)
	out := make([]board.FilledSegment, 0, len(raw))
	for _, segment := range raw {
		shown := board.FilledSegment{
			Text:     segment.Text,
			Reading:  segment.Reading,
			Artifact: segment.Artifact,
		}
		if segment.Cite > 0 && segment.Cite <= len(f.Blocks) {
			cited := &f.Blocks[segment.Cite-1]
			shown.Numeral = cited.Numeral
			shown.Struck = cited.State == board.StateUnreadable
			shown.Tone = boardcatalog.ToneInfo
			if cited.Tone.Severity() >= boardcatalog.ToneWarn.Severity() {
				shown.Tone = boardcatalog.ToneWarn
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
		if !spec.Records.Empty() {
			block.Chips, _ = rowsOf(spec.Records, reads)
		}
	case board.BlockWork:
		assembleWork(&block, spec, reads)
	case board.BlockActions:
		assembleActions(&block, spec, reads)
	case board.BlockMetric:
		assembleMetric(&block, spec, reads)
	}

	return block
}

func assembleWork(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	set := spec.Records
	block.Source = board.Source{
		Kind:       board.SourceRecords,
		RecordKind: set.Kind,
		Count:      len(set.IDs),
	}
	if !reads.Readable[set.Kind] {
		block.State = board.StateUnreadable
		return
	}
	rows, readable := rowsOf(set, reads)
	block.Source.Readable = readable
	block.Work = make([]board.WorkRow, 0, len(rows))
	for idx := range rows {
		row := board.WorkRow{Row: rows[idx], Signals: []board.WorkSignal{}}
		if rows[idx].Readable {
			row.Signals = append(row.Signals, reads.Work[rows[idx].ID]...)
		}
		for _, signal := range row.Signals {
			if signal.Tone.Severity() > row.Tone.Severity() {
				row.Tone = signal.Tone
			}
			if signal.DueAt != nil && (row.DueAt == nil || *signal.DueAt < *row.DueAt) {
				due := *signal.DueAt
				row.DueAt = &due
			}
		}
		if row.Tone.Severity() > block.Tone.Severity() {
			block.Tone = row.Tone
		}
		block.Work = append(block.Work, row)
	}
	if readable == 0 {
		block.State = board.StateUnreadable
	} else if len(block.Work) > 0 && !slices.ContainsFunc(block.Work, func(row board.WorkRow) bool {
		return len(row.Signals) > 0
	}) {
		block.State = board.StateEmpty
	}
}

func assembleActions(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	block.Actions = make([]board.FilledAction, 0, len(spec.Actions))
	for idx := range spec.Actions {
		action := &spec.Actions[idx]
		filled := board.FilledAction{Label: action.Label, Ask: board.PlainText(action.Ask)}
		if !action.Records.Empty() && reads.Readable[action.Records.Kind] {
			filled.Records, _ = rowsOf(action.Records, reads)
		}
		block.Actions = append(block.Actions, filled)
	}
}

func assembleFigures(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	block.Figures = make([]board.FilledFigure, 0, len(spec.Figures))
	kinds := make([]board.SourceKind, 0, len(spec.Figures))
	for idx := range spec.Figures {
		source := &spec.Figures[idx]
		figure := board.FilledFigure{Label: source.Label}
		switch {
		case source.Metric != "":
			figure.Value = reads.figure(source.Ask(), source.Figure)
			kinds = append(kinds, board.SourceMetrics)
		case source.Records != nil:
			figure.Value, figure.Tone = aggregate(source, reads)
			kinds = append(kinds, board.SourceRecords)
			block.Source.RecordKind = source.Records.Kind
			block.Source.Count = max(block.Source.Count, len(source.Records.IDs))
		case source.List.IsNotNil():
			figure.Value, _ = reads.list(source.List)
			kinds = append(kinds, board.SourceTable)
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

func aggregate(
	source *board.FigureSource,
	reads *Reads,
) (value *boardcatalog.Value, tone boardcatalog.Tone) {
	if !reads.Readable[source.Records.Kind] {
		return nil, ""
	}
	if source.Measure != "" && reads.withheld(source.Records.Kind, source.Measure) {
		return nil, ""
	}
	values, readable := measured(source, reads)
	if readable < len(source.Records.IDs) {
		return nil, ""
	}
	spec, _ := source.Measure.Spec(source.Records.Kind)
	if source.Aggregate == board.AggregateSum || source.Aggregate == board.AggregateMax {
		value = combined(source, &spec, values)
	} else {
		value = counted(source, values, readable)
	}
	if value == nil {
		return nil, ""
	}
	if spec.Adverse && value.Number.IsPositive() {
		return value, boardcatalog.ToneWarn
	}

	return value, ""
}

func measured(
	source *board.FigureSource,
	reads *Reads,
) (values []boardcatalog.Value, readable int) {
	set := source.Records
	values = make([]boardcatalog.Value, 0, len(set.IDs))
	for _, id := range set.IDs {
		if _, ok := reads.summary(set.Kind, id); !ok {
			continue
		}
		readable++
		if source.Measure == "" {
			continue
		}
		if reading, has := reads.reading(set.Kind, source.Measure, id); has &&
			!reading.Estimated() {
			values = append(values, reading.Value)
		}
	}

	return values, readable
}

func combined(
	source *board.FigureSource,
	spec *boardcatalog.Measure,
	values []boardcatalog.Value,
) *boardcatalog.Value {
	if len(values) == 0 {
		return &boardcatalog.Value{Kind: spec.Value, Number: decimal.Zero}
	}
	currency := values[0].Currency
	total := values[0].Number
	for i := 1; i < len(values); i++ {
		value := &values[i]
		if value.Currency != currency {
			return nil
		}
		switch {
		case source.Aggregate == board.AggregateSum:
			total = total.Add(value.Number)
		case spec.Ascending && value.Number.LessThan(total):
			total = value.Number
		case !spec.Ascending && value.Number.GreaterThan(total):
			total = value.Number
		}
	}

	return &boardcatalog.Value{Kind: spec.Value, Number: total, Currency: currency}
}

func counted(
	source *board.FigureSource,
	values []boardcatalog.Value,
	readable int,
) *boardcatalog.Value {
	count := readable
	if source.Measure != "" {
		count = 0
		for i := range values {
			if values[i].Number.IsPositive() {
				count++
			}
		}
	}

	return &boardcatalog.Value{
		Kind:   boardcatalog.ValueCount,
		Number: decimal.NewFromInt(int64(count)),
	}
}

func analysisFigure(read *AnalysisRead, index int) *boardcatalog.Value {
	if read == nil || index < 0 || index >= len(read.Display.Figures) {
		return nil
	}
	figure := &read.Display.Figures[index]

	return valueOf(figure.Value, figure.Kind, figure.Currency)
}

func valueOf(raw any, kind analysis.ValueKind, currency string) *boardcatalog.Value {
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
		return &boardcatalog.Value{Kind: boardcatalog.ValueText, Text: fmt.Sprint(typed)}
	}
	switch kind {
	case analysis.ValueMoney:
		return &boardcatalog.Value{
			Kind:     boardcatalog.ValueMoney,
			Number:   number,
			Currency: currency,
		}
	case analysis.ValueCount:
		return &boardcatalog.Value{Kind: boardcatalog.ValueCount, Number: number}
	case analysis.ValuePercent:
		return &boardcatalog.Value{Kind: boardcatalog.ValuePercent, Number: number}
	case analysis.ValueText, analysis.ValueNumber, analysis.ValueDate, analysis.ValueDateTime,
		analysis.ValueBool:
	}

	return &boardcatalog.Value{Kind: boardcatalog.ValueNumber, Number: number}
}

func rowsOf(set *board.RecordSet, reads *Reads) (rows []board.Row, readable int) {
	rows = make([]board.Row, 0, len(set.IDs))
	for _, id := range set.IDs {
		row := board.Row{Kind: set.Kind, ID: id}
		if summary, ok := reads.summary(set.Kind, id); ok {
			readable++
			row.Readable = true
			row.Label = summary.Label
			row.Subtitle = summary.Subtitle
			row.Status = summary.Status
			row.StatusTone = summary.StatusTone
			row.Opens = summary.Opens
		}
		rows = append(rows, row)
	}

	return rows, readable
}

func assembleRecords(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	set := spec.Records
	block.Measure = spec.Measure
	block.Source = board.Source{
		Kind:       board.SourceRecords,
		RecordKind: set.Kind,
		Count:      len(set.IDs),
	}
	if !reads.Readable[set.Kind] {
		block.State = board.StateUnreadable
		return
	}
	rows, readable := rowsOf(set, reads)
	block.Source.Readable = readable
	measure, measured := spec.Measure.Spec(set.Kind)
	if measured {
		measureRows(rows, &measure, spec.Measure, reads)
		for idx := range rows {
			if rows[idx].Measure != nil &&
				rows[idx].Measure.Tone.Severity() > block.Tone.Severity() {
				block.Tone = rows[idx].Measure.Tone
			}
		}
	}
	if spec.Order == board.OrderMeasure {
		slices.SortStableFunc(rows, func(a, b board.Row) int {
			return compareRows(&a, &b, measure.Ascending)
		})
	}
	block.Rows = rows
}

func measureRows(
	rows []board.Row,
	measure *boardcatalog.Measure,
	name board.Measure,
	reads *Reads,
) {
	withheld := reads.withheld(measure.Kind, name)
	placed := make([]int, 0, len(rows))
	readings := make([]*boardreading.Reading, 0, len(rows))
	for idx := range rows {
		if !rows[idx].Readable {
			continue
		}
		if withheld {
			rows[idx].Measure = &board.RowMeasure{Measure: name}
			continue
		}
		reading, has := reads.reading(measure.Kind, name, rows[idx].ID)
		if !has {
			continue
		}
		value := reading.Value
		rows[idx].Measure = &board.RowMeasure{
			Measure:  name,
			Value:    &value,
			Standing: reading.Standing,
			Fact:     reading.Fact,
			Tone:     reading.Tone,
		}
		placed = append(placed, idx)
		readings = append(readings, &reading)
	}
	bars := boardbar.Layout(measure.Shape, readings)
	for at, idx := range placed {
		rows[idx].Measure.Bar = bars[at]
	}
}

func compareRows(a, b *board.Row, ascending bool) int {
	aValue, bValue := measuredValue(a), measuredValue(b)
	switch {
	case a.Readable != b.Readable:
		if a.Readable {
			return -1
		}
		return 1
	case aValue == nil || bValue == nil:
		if aValue != nil {
			return -1
		}
		if bValue != nil {
			return 1
		}
		return 0
	case ascending:
		return aValue.Number.Cmp(bValue.Number)
	default:
		return bValue.Number.Cmp(aValue.Number)
	}
}

func measuredValue(row *board.Row) *boardcatalog.Value {
	if row.Measure == nil {
		return nil
	}

	return row.Measure.Value
}

func assembleCompare(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	block.Source = board.Source{
		Kind:       board.SourceAnalysis,
		RecordKind: spec.Records.Kind,
		Count:      len(spec.Records.IDs),
	}
	read := reads.Analyses[spec.Analysis]
	if read == nil || spec.Index >= len(read.Display.Tables) ||
		!reads.Readable[spec.Records.Kind] {
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
			Cells:  make([]*boardcatalog.Value, len(columns)),
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
		if cell == nil || cell.Kind == boardcatalog.ValueText {
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
		if cell != nil && cell.Kind != boardcatalog.ValueText && cell.Number.Equal(*best) {
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
	chart, form := boardbar.FromAnalysis(&read.Display.Charts[spec.Index])
	block.Metric = &board.FilledMetric{Form: form, Read: chart}
	if !boardbar.Fits(form, chart) {
		block.State = board.StateEmpty
	}
}

func assembleMetric(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	block.Source = board.Source{Kind: board.SourceMetrics}
	metric, known := boardregistry.Catalog().MetricNamed(spec.Metric)
	read := reads.Metrics[spec.Ask()]
	if !known || read == nil {
		block.State = board.StateUnreadable
		return
	}
	block.Source.Title = read.Title
	block.Metric = &board.FilledMetric{
		Name: metric.Name,
		Form: metric.Form,
		Read: read,
		Rows: metricRows(read, reads),
	}
	for _, check := range read.Checks {
		if check.Tone.Severity() > block.Tone.Severity() {
			block.Tone = check.Tone
		}
	}
	if !boardbar.Fits(metric.Form, read) {
		block.State = board.StateEmpty
	}
}

func metricRows(read *boardcatalog.MetricRead, reads *Reads) []board.Row {
	refs := MetricRefs(read)
	rows := make([]board.Row, 0, len(refs))
	for _, ref := range refs {
		row := board.Row{Kind: ref.Kind, ID: ref.ID}
		if summary, ok := reads.summary(ref.Kind, ref.ID); ok {
			row.Readable = true
			row.Label = summary.Label
			row.Subtitle = summary.Subtitle
			row.Status = summary.Status
			row.StatusTone = summary.StatusTone
			row.Opens = summary.Opens
		}
		rows = append(rows, row)
	}

	return rows
}

func MetricRefs(read *boardcatalog.MetricRead) []boardcatalog.Ref {
	refs := make([]boardcatalog.Ref, 0, 4)
	add := func(ref boardcatalog.Ref) {
		if ref.ID.IsNotNil() && !slices.Contains(refs, ref) {
			refs = append(refs, ref)
		}
	}
	if read == nil {
		return refs
	}
	for idx := range read.Log {
		for _, ref := range read.Log[idx].Refs {
			add(ref)
		}
	}
	for idx := range read.Checks {
		for _, ref := range read.Checks[idx].Refs {
			add(ref)
		}
	}
	if read.Chart != nil {
		for idx := range read.Chart.Rows {
			if ref := read.Chart.Rows[idx].Ref; ref != nil {
				add(*ref)
			}
		}
	}

	return refs
}

func assembleTimeline(block *board.FilledBlock, spec *board.Block, reads *Reads) {
	set := spec.Records
	block.Source = board.Source{Kind: board.SourceActivity, RecordKind: set.Kind, Count: 1}
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
			check.Count = reads.figure(source.Ask(), source.Figure)
		case source.Records != nil:
			kinds = append(kinds, board.SourceRecords)
			block.Source.RecordKind = source.Records.Kind
			block.Source.Count += len(source.Records.IDs)
			if reads.Readable[source.Records.Kind] {
				rows, readable := rowsOf(source.Records, reads)
				check.Records = slices.DeleteFunc(
					rows,
					func(row board.Row) bool { return !row.Readable },
				)
				check.Hidden = len(source.Records.IDs) - readable
				check.Count = &boardcatalog.Value{
					Kind:   boardcatalog.ValueCount,
					Number: decimal.NewFromInt(int64(readable)),
				}
				block.Source.Readable += readable
			} else {
				check.Hidden = len(source.Records.IDs)
			}
		}
		if check.Tone.Severity() > block.Tone.Severity() {
			block.Tone = check.Tone
		}
		block.Checks = append(block.Checks, check)
	}
	block.Source.Kind = commonKind(kinds)
	if block.Source.Kind == board.SourceMixed || block.Source.Kind == board.SourceMetrics {
		block.Source.RecordKind = ""
	}
}

func assembleMeter(block *board.FilledBlock, layout *board.Board, spec *board.Block, reads *Reads) {
	if spec.Metric != "" {
		block.Source = board.Source{Kind: board.SourceMetrics}
		value := reads.figure(spec.Ask(), spec.Figure)
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
	meter := &board.FilledMeter{
		Total: len(checks),
		Tones: make([]boardcatalog.Tone, 0, len(checks)),
	}
	for idx := range checks {
		meter.Tones = append(meter.Tones, checks[idx].Tone)
		if checks[idx].Tone == boardcatalog.ToneOK {
			meter.Clear++
		}
	}
	slices.SortStableFunc(meter.Tones, func(a, b boardcatalog.Tone) int {
		return cmp.Compare(a.Severity(), b.Severity())
	})
	for _, tone := range meter.Tones {
		if tone.Severity() > block.Tone.Severity() {
			block.Tone = tone
		}
	}
	block.Meter = meter
}
