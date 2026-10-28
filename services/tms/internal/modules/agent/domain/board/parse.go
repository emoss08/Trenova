package board

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	fieldRecords  = "records"
	fieldAnalysis = "analysis"
)

var blockFields = map[BlockType][]string{
	BlockFigures:  {"figures"},
	BlockRecords:  {fieldRecords, "measure", "order"},
	BlockCompare:  {fieldRecords, fieldAnalysis, "table", "key", "rows"},
	BlockTable:    {fieldAnalysis, "table", "artifact", "limit"},
	BlockChart:    {fieldAnalysis, "chart"},
	BlockTimeline: {"record", "limit"},
	BlockChecks:   {"checks"},
	BlockMeter:    {fieldChecksBlock, "metric"},
	BlockProse:    {"text"},
	BlockNote:     {"tone", "text", fieldRecords},
}

var commonFields = []string{"type", "title", "half"}

// fieldChecksBlock is the checks block a meter fills from. It was "of", which
// a model read as "of how many" and sent beside a metric, so every meter it
// drew was refused once.
const fieldChecksBlock = "checksBlock"

type parser struct {
	problems []string
}

func (p *parser) addf(format string, args ...any) {
	p.problems = append(p.problems, fmt.Sprintf(format, args...))
}

func Parse(arguments map[string]any) (board *Board, problems []string) {
	p := &parser{}
	board = &Board{
		Title:   p.words(arguments, "title", "title", MaxTitleRunes, true),
		Verdict: p.words(arguments, "verdict", "verdict", MaxVerdictRunes, true),
	}

	raw, isList := arguments["blocks"].([]any)
	switch {
	case !isList || len(raw) == 0:
		p.addf("blocks needs at least one block")
	case len(raw) > MaxBlocks:
		p.addf("a board holds at most %d blocks", MaxBlocks)
	default:
		board.Blocks = make([]Block, 0, len(raw))
		for idx, entry := range raw {
			fields, isObject := entry.(map[string]any)
			if !isObject {
				p.addf("blocks[%d] is not an object", idx+1)
				continue
			}
			board.Blocks = append(board.Blocks, p.block(fields, idx+1))
		}
	}
	if len(p.problems) == 0 {
		p.whole(board)
	}

	return board, p.problems
}

func (p *parser) whole(board *Board) {
	notes, data := 0, 0
	for idx := range board.Blocks {
		switch {
		case board.Blocks[idx].Type == BlockNote:
			notes++
		case !board.Blocks[idx].Type.Words():
			data++
		}
	}
	if notes > MaxNotes {
		p.addf("a board holds at most %d notes", MaxNotes)
	}
	if data == 0 {
		p.addf(
			"a board needs at least one block that reads data; prose and notes alone belong in the reply",
		)
	}
	total := 0
	for _, ids := range board.Records() {
		total += len(ids)
	}
	if total > MaxBoardRecords {
		p.addf("the board names %d records; it may name at most %d", total, MaxBoardRecords)
	}

	p.citations("verdict", board.Verdict, board)
	for idx := range board.Blocks {
		block := &board.Blocks[idx]
		if block.Type == BlockProse {
			p.citations(fmt.Sprintf("blocks[%d].text", idx+1), block.Text, board)
		}
		if block.Type == BlockMeter && block.Of > 0 {
			if block.Of > len(board.Blocks) || board.Blocks[block.Of-1].Type != BlockChecks {
				p.addf("blocks[%d].checksBlock names block %d, which is not a checks block",
					idx+1, block.Of)
			}
		}
	}
}

func (p *parser) citations(field, text string, board *Board) {
	for _, segment := range Segments(text) {
		if segment.Cite == 0 {
			continue
		}
		if segment.Cite > len(board.Blocks) {
			p.addf("%s cites block %d, but the board has %d",
				field, segment.Cite, len(board.Blocks))
			continue
		}
		if board.Blocks[segment.Cite-1].Type.Words() {
			p.addf("%s cites block %d, which is words, not data; cite the block that shows it",
				field, segment.Cite)
		}
	}
}

func (p *parser) block(fields map[string]any, at int) Block {
	path := fmt.Sprintf("blocks[%d]", at)
	blockType := BlockType(strings.TrimSpace(stringOf(fields["type"])))
	if !blockType.IsValid() {
		p.addf("%s.type must be one of %s", path, joinTypes())
		return Block{Type: blockType}
	}
	allowed := slices.Concat(commonFields, blockFields[blockType])
	for key := range fields {
		if !slices.Contains(allowed, key) {
			p.addf("%s is a %s block, which takes no %q", path, blockType, key)
		}
	}

	titleMost := MaxBlockTitleRunes
	if blockType == BlockNote {
		titleMost = MaxNoteTitleRunes
	}
	block := Block{
		Type:  blockType,
		Title: p.words(fields, "title", path+".title", titleMost, false),
		Half:  fields["half"] == true,
	}
	switch blockType {
	case BlockFigures:
		block.Figures = p.figures(fields["figures"], path)
	case BlockRecords:
		block.Records = p.records(fields[fieldRecords], path+".records", 1, MaxListRecords)
		block.Measure = p.measure(fields, path, block.Records)
		block.Order = p.order(fields, path, block.Measure)
	case BlockCompare:
		block.Records = p.records(
			fields[fieldRecords], path+".records", MinCompareRecords, MaxCompareRecords,
		)
		block.Analysis = p.id(fields, fieldAnalysis, path, true)
		block.Index = p.index(fields, "table", path)
		block.Key = p.text(fields, "key", path+".key", MaxColumnRunes, true)
		block.Rows = p.compareRows(fields["rows"], path)
	case BlockTable:
		block.Artifact = p.id(fields, "artifact", path, false)
		if block.Artifact.IsNil() {
			block.Analysis = p.id(fields, fieldAnalysis, path, true)
			block.Index = p.index(fields, "table", path)
		}
		block.Limit = p.limit(fields, path, MaxTableRows, DefaultTableRows)
	case BlockChart:
		block.Analysis = p.id(fields, fieldAnalysis, path, true)
		block.Index = p.index(fields, "chart", path)
	case BlockTimeline:
		p.timeline(fields, path, &block)
	case BlockChecks:
		block.Checks = p.checks(fields["checks"], path)
	case BlockMeter:
		p.meter(fields, path, &block)
	case BlockProse:
		block.Text = p.words(fields, "text", path+".text", MaxProseRunes, true)
	case BlockNote:
		p.note(fields, path, &block)
	}

	return block
}

func (p *parser) timeline(fields map[string]any, path string, block *Block) {
	if fields["record"] == nil {
		p.addf("%s.record is required: the id of the record whose history to show", path)
		return
	}
	block.Records = p.records([]any{fields["record"]}, path+".record", 1, 1)
	block.Limit = p.limit(fields, path, MaxTimelineEntries, DefaultTimeline)
}

func (p *parser) note(fields map[string]any, path string, block *Block) {
	block.Tone = p.tone(fields, path)
	block.Text = p.words(fields, "text", path+".text", MaxNoteTextRunes, true)
	if block.Title == "" {
		p.addf("%s.title is required: what the note says in a line", path)
	}
	if _, named := fields[fieldRecords]; named {
		block.Records = p.records(fields[fieldRecords], path+".records", 1, MaxNoteRecords)
	}
}

func (p *parser) figures(raw any, path string) []FigureSource {
	entries, isList := raw.([]any)
	if !isList || len(entries) == 0 || len(entries) > MaxFigures {
		p.addf("%s.figures needs 1 to %d figures", path, MaxFigures)
		return nil
	}
	figures := make([]FigureSource, 0, len(entries))
	for idx, entry := range entries {
		fields, isObject := entry.(map[string]any)
		at := fmt.Sprintf("%s.figures[%d]", path, idx+1)
		if !isObject {
			p.addf("%s is not an object", at)
			continue
		}
		figure := FigureSource{Label: p.words(fields, "label", at+".label", MaxLabelRunes, true)}
		sources := 0
		if metric := strings.TrimSpace(stringOf(fields["metric"])); metric != "" {
			sources++
			figure.Metric = p.metric(metric, at)
		}
		if _, named := fields[fieldRecords]; named {
			sources++
			figure.Records = p.records(fields[fieldRecords], at+".records", 1, MaxListRecords)
			figure.Measure = p.measure(fields, at, figure.Records)
			figure.Aggregate = p.aggregate(fields, at, figure.Measure)
		}
		if _, named := fields[fieldAnalysis]; named {
			sources++
			figure.Analysis = p.id(fields, fieldAnalysis, at, true)
			figure.Index = p.index(fields, "figure", at)
		}
		if sources != 1 {
			p.addf(
				"%s needs exactly one source: a metric, records, or an analysis with its figure",
				at,
			)
		}
		figures = append(figures, figure)
	}

	return figures
}

func (p *parser) checks(raw any, path string) []Check {
	entries, isList := raw.([]any)
	if !isList || len(entries) == 0 || len(entries) > MaxChecks {
		p.addf("%s.checks needs 1 to %d checks", path, MaxChecks)
		return nil
	}
	checks := make([]Check, 0, len(entries))
	for idx, entry := range entries {
		fields, isObject := entry.(map[string]any)
		at := fmt.Sprintf("%s.checks[%d]", path, idx+1)
		if !isObject {
			p.addf("%s is not an object", at)
			continue
		}
		check := Check{
			Title:  p.words(fields, "title", at+".title", MaxCheckTitleRunes, true),
			Tone:   p.tone(fields, at),
			Detail: p.words(fields, "detail", at+".detail", MaxCheckDetailRunes, false),
		}
		_, named := fields[fieldRecords]
		metric := strings.TrimSpace(stringOf(fields["metric"]))
		switch {
		case named && metric != "":
			p.addf("%s counts records or a metric, not both", at)
		case named:
			check.Records = p.records(fields[fieldRecords], at+".records", 1, MaxCheckRecords)
		case metric != "":
			check.Metric = p.metric(metric, at)
		default:
			p.addf("%s needs the records it counts or a metric", at)
		}
		checks = append(checks, check)
	}

	return checks
}

func (p *parser) compareRows(raw any, path string) []CompareRow {
	entries, isList := raw.([]any)
	if !isList || len(entries) == 0 || len(entries) > MaxCompareRows {
		p.addf("%s.rows needs 1 to %d rows", path, MaxCompareRows)
		return nil
	}
	rows := make([]CompareRow, 0, len(entries))
	for idx, entry := range entries {
		fields, isObject := entry.(map[string]any)
		at := fmt.Sprintf("%s.rows[%d]", path, idx+1)
		if !isObject {
			p.addf("%s is not an object", at)
			continue
		}
		row := CompareRow{
			Column: p.text(fields, "column", at+".column", MaxColumnRunes, true),
			Label:  p.words(fields, "label", at+".label", MaxLabelRunes, false),
		}
		switch better := Better(strings.TrimSpace(stringOf(fields["better"]))); better {
		case "", BetterHigher, BetterLower:
			row.Better = better
		default:
			p.addf("%s.better is higher or lower", at)
		}
		rows = append(rows, row)
	}

	return rows
}

func (p *parser) meter(fields map[string]any, path string, block *Block) {
	_, ofChecks := fields[fieldChecksBlock]
	metric := strings.TrimSpace(stringOf(fields["metric"]))
	switch {
	case ofChecks && metric != "":
		p.addf("%s fills from a checks block or a percentage metric, not both: keep metric "+
			"for a percentage such as onTimePercent and leave out checksBlock, or keep "+
			"checksBlock and leave out metric", path)
	case ofChecks:
		position, ok := wholeOf(fields[fieldChecksBlock])
		if !ok || position < 1 {
			p.addf("%s.checksBlock is the position of a checks block on this board, from 1",
				path)
			return
		}
		block.Of = position
	case metric != "":
		block.Metric = p.metric(metric, path)
		if kind, _ := MetricKind(block.Metric); block.Metric != "" && kind != ValuePercent {
			p.addf("%s.metric must be a percentage to draw as a meter", path)
		}
	default:
		p.addf("%s needs checksBlock (the position of a checks block) or a percentage metric",
			path)
	}
}

func (p *parser) records(raw any, path string, least, most int) *RecordSet {
	entries, isList := raw.([]any)
	if !isList || len(entries) < least || len(entries) > most {
		if least == most {
			p.addf("%s needs one record id", path)
		} else {
			p.addf("%s needs %d to %d record ids", path, least, most)
		}
		return nil
	}
	set := &RecordSet{IDs: make([]pulid.ID, 0, len(entries))}
	for _, entry := range entries {
		id, err := pulid.Parse(strings.TrimSpace(stringOf(entry)))
		if err != nil || id.IsNil() {
			p.addf("%s holds %q, which is not a record id", path, stringOf(entry))
			return nil
		}
		resource, known := resourceOf(id)
		if !known {
			p.addf("%s holds %s, which is not a %s", path, id, joinResources())
			return nil
		}
		if set.Resource == "" {
			set.Resource = resource
		} else if set.Resource != resource {
			p.addf("%s mixes %s and %s ids; one block shows one kind of record",
				path, set.Resource, resource)
			return nil
		}
		if !slices.Contains(set.IDs, id) {
			set.IDs = append(set.IDs, id)
		}
	}

	return set
}

func (p *parser) measure(fields map[string]any, path string, set *RecordSet) Measure {
	measure := Measure(strings.TrimSpace(stringOf(fields["measure"])))
	if measure == "" || set == nil {
		return measure
	}
	if !measure.Of(set.Resource) {
		measures := MeasuresOf(set.Resource)
		if len(measures) == 0 {
			p.addf("%s.measure: a %s has no measure", path, set.Resource)
		} else {
			p.addf("%s.measure for a %s is one of %s", path, set.Resource, joinMeasures(measures))
		}
		return ""
	}

	return measure
}

func (p *parser) order(fields map[string]any, path string, measure Measure) Order {
	switch order := Order(strings.TrimSpace(stringOf(fields["order"]))); order {
	case "":
		if measure != "" {
			return OrderMeasure
		}
		return OrderGiven
	case OrderGiven:
		return order
	case OrderMeasure:
		if measure == "" {
			p.addf("%s.order measure needs a measure", path)
		}
		return order
	default:
		p.addf("%s.order is given or measure", path)
		return OrderGiven
	}
}

func (p *parser) aggregate(fields map[string]any, path string, measure Measure) Aggregate {
	aggregate := Aggregate(strings.TrimSpace(stringOf(fields["aggregate"])))
	switch {
	case aggregate == "":
		return AggregateCount
	case !slices.Contains(AllAggregates(), aggregate):
		p.addf("%s.aggregate is count, sum or max", path)
		return AggregateCount
	case aggregate != AggregateCount && measure == "":
		p.addf("%s.aggregate %s needs a measure to add up", path, aggregate)
	}

	return aggregate
}

func (p *parser) metric(key, path string) string {
	if _, ok := MetricKind(key); !ok {
		p.addf("%s.metric %q is not one of %s", path, key, strings.Join(MetricKeys(), ", "))
		return ""
	}

	return key
}

func (p *parser) tone(fields map[string]any, path string) Tone {
	tone := Tone(strings.TrimSpace(stringOf(fields["tone"])))
	if !tone.IsValid() {
		p.addf("%s.tone is bad, warn, info or ok", path)
		return ToneInfo
	}

	return tone
}

func (p *parser) id(fields map[string]any, key, path string, required bool) pulid.ID {
	text := strings.TrimSpace(stringOf(fields[key]))
	if text == "" {
		if required {
			p.addf("%s.%s is required", path, key)
		}
		return pulid.Nil
	}
	id, err := pulid.Parse(text)
	if err != nil || !strings.HasPrefix(text, "art_") {
		p.addf("%s.%s is the artifactId a tool result gave you", path, key)
		return pulid.Nil
	}

	return id
}

func (p *parser) index(fields map[string]any, key, path string) int {
	value, present := fields[key]
	if !present {
		p.addf("%s.%s is required: its position, from 1", path, key)
		return 0
	}
	position, ok := wholeOf(value)
	if !ok || position < 1 {
		p.addf("%s.%s is a position from 1", path, key)
		return 0
	}

	return position - 1
}

func (p *parser) limit(fields map[string]any, path string, most, fallback int) int {
	value, present := fields["limit"]
	if !present {
		return fallback
	}
	limit, ok := wholeOf(value)
	if !ok || limit < 1 || limit > most {
		p.addf("%s.limit is 1 to %d", path, most)
		return fallback
	}

	return limit
}

var emphasis = strings.NewReplacer("**", "", "__", "", "`", "")

func (p *parser) words(fields map[string]any, key, path string, most int, required bool) string {
	text := p.limited(fields, key, path, most, required, emphasis.Replace)
	if figures := Figures(PlainText(text)); len(figures) > 0 {
		p.addf("%s states %s; the board reads every figure live from its blocks, so take it "+
			"out of the words. Where the board should show it, add a block that reads it "+
			"(records or checks for the loads it counts, figures for a metric) and cite that "+
			"block; never point at a block the board does not have", path, figureList(figures))
	}

	return text
}

func figureList(figures []string) string {
	named := figures[:min(len(figures), maxFiguresNamed)]
	quoted := make([]string, 0, len(named))
	for _, figure := range named {
		quoted = append(quoted, strconv.Quote(figure))
	}

	return strings.Join(quoted, ", ")
}

func (p *parser) text(fields map[string]any, key, path string, most int, required bool) string {
	return p.limited(fields, key, path, most, required, nil)
}

func (p *parser) limited(
	fields map[string]any,
	key, path string,
	most int,
	required bool,
	plain func(string) string,
) string {
	value, present := fields[key]
	text, isText := value.(string)
	if present && !isText {
		p.addf("%s is text", path)
		return ""
	}
	if plain != nil {
		text = plain(text)
	}
	text = strings.Join(strings.Fields(text), " ")
	switch {
	case text == "" && required:
		p.addf("%s is required", path)
	case utf8.RuneCountInString(text) > most:
		p.addf("%s is longer than %d characters", path, most)
	}

	return text
}

func resourceOf(id pulid.ID) (permission.Resource, bool) {
	text := id.String()
	cut := strings.IndexByte(text, '_')
	if cut < 0 {
		return "", false
	}
	resource, known := permission.ResourceOfIDPrefix(text[:cut+1])
	if !known || !Supports(resource) {
		return "", false
	}

	return resource, true
}

func wholeOf(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		if typed != math.Trunc(typed) || typed > math.MaxInt32 || typed < math.MinInt32 {
			return 0, false
		}
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	default:
		return 0, false
	}
}

func stringOf(value any) string {
	text, _ := value.(string)

	return text
}

func joinTypes() string {
	names := make([]string, 0, len(AllBlockTypes()))
	for _, blockType := range AllBlockTypes() {
		names = append(names, string(blockType))
	}

	return strings.Join(names, ", ")
}

func joinResources() string {
	names := make([]string, 0, len(Resources()))
	for _, resource := range Resources() {
		names = append(names, resource.Noun())
	}

	return strings.Join(names[:len(names)-1], ", ") + " or " + names[len(names)-1]
}

func joinMeasures(measures []Measure) string {
	names := make([]string, 0, len(measures))
	for _, measure := range measures {
		names = append(names, string(measure))
	}

	return strings.Join(names, ", ")
}
