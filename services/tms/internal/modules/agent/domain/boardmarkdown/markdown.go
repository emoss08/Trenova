package boardmarkdown

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
)

type MarkdownMeta struct {
	Question   string
	Author     string
	AnsweredAt int64
	Location   *time.Location
}

func Markdown(f *board.Filled, meta MarkdownMeta) string {
	var sb strings.Builder
	sb.WriteString("# ")
	sb.WriteString(oneLine(f.Title))
	sb.WriteString("\n")
	if question := oneLine(meta.Question); question != "" {
		sb.WriteString("\n> ")
		sb.WriteString(question)
		sb.WriteString("\n")
	}
	if verdict := wordsText(f, f.Verdict); verdict != "" {
		sb.WriteString("\n")
		sb.WriteString(verdict)
		sb.WriteString("\n")
	}
	for idx := range f.Blocks {
		writeBlock(&sb, f, &f.Blocks[idx], meta.Location)
	}
	sb.WriteString("\n---\n")
	sb.WriteString("Read live at ")
	sb.WriteString(board.FormatMoment(f.ReadAt, meta.Location))
	sb.WriteString(".")
	if meta.AnsweredAt > 0 {
		sb.WriteString(" Answered")
		if author := oneLine(meta.Author); author != "" {
			sb.WriteString(" by ")
			sb.WriteString(author)
		}
		sb.WriteString(" at ")
		sb.WriteString(board.FormatMoment(meta.AnsweredAt, meta.Location))
		sb.WriteString(".")
	}
	sb.WriteString("\n")

	return sb.String()
}

func writeBlock(
	sb *strings.Builder,
	f *board.Filled,
	block *board.FilledBlock,
	location *time.Location,
) {
	switch block.Type {
	case board.BlockProse:
		sb.WriteString("\n")
		sb.WriteString(wordsText(f, block.Text))
		sb.WriteString("\n")
		return
	case board.BlockNote:
		writeNote(sb, f, block)
		return
	case board.BlockActions:
		writeActions(sb, block.Actions)
		return
	case board.BlockFigures,
		board.BlockRecords,
		board.BlockCompare,
		board.BlockTable,
		board.BlockChart,
		board.BlockTimeline,
		board.BlockChecks,
		board.BlockMeter,
		board.BlockWork,
		board.BlockMetric:
	}

	sb.WriteString("\n## ")
	sb.WriteString(strconv.Itoa(block.Numeral))
	sb.WriteString(". ")
	sb.WriteString(blockHeading(block))
	sb.WriteString("\n\n")
	if caption := wordsText(f, block.Caption); caption != "" {
		sb.WriteString("_")
		sb.WriteString(caption)
		sb.WriteString("_\n\n")
	}
	if block.State == board.StateUnreadable {
		sb.WriteString("_Could not be read._\n")
		return
	}
	if block.State == board.StateEmpty {
		sb.WriteString("_Nothing to show._\n")
		return
	}

	switch block.Type {
	case board.BlockFigures:
		writeFigures(sb, block.Figures)
	case board.BlockRecords:
		writeRows(sb, block, location)
	case board.BlockCompare:
		writeCompare(sb, block.Compare)
	case board.BlockTable:
		if block.Table != nil {
			sb.WriteString(strings.TrimLeft(block.Table.Table.Markdown(), "\n"))
		}
	case board.BlockChart, board.BlockMetric:
		writeMetric(sb, block.Metric, location)
	case board.BlockTimeline:
		writeEvents(sb, block.Events, location)
	case board.BlockChecks:
		writeChecks(sb, block.Checks)
	case board.BlockMeter:
		writeMeter(sb, block.Meter)
	case board.BlockWork:
		writeWork(sb, block.Work)
	case board.BlockProse, board.BlockNote, board.BlockActions:
	}
}

func writeWork(sb *strings.Builder, rows []board.WorkRow) {
	for idx := range rows {
		row := &rows[idx]
		label := oneLine(row.Label)
		if !row.Readable || label == "" {
			label = "A record the reader may not open"
		}
		sb.WriteString("- ")
		sb.WriteString(label)
		if len(row.Signals) == 0 {
			sb.WriteString(": nothing open\n")
			continue
		}
		sb.WriteString("\n")
		for _, signal := range row.Signals {
			sb.WriteString("  - ")
			sb.WriteString(signal.Lens)
			sb.WriteString(": ")
			sb.WriteString(oneLine(signal.Title))
			sb.WriteString("\n")
		}
	}
}

func writeActions(sb *strings.Builder, actions []board.FilledAction) {
	if len(actions) == 0 {
		return
	}
	sb.WriteString("\nOffered as buttons that send the person's own request:\n")
	for idx := range actions {
		action := &actions[idx]
		sb.WriteString("- ")
		sb.WriteString(oneLine(action.Label))
		sb.WriteString(": ")
		sb.WriteString(oneLine(action.Ask))
		if labels := rowLabels(action.Records); labels != "" {
			sb.WriteString(" (")
			sb.WriteString(labels)
			sb.WriteString(")")
		}
		sb.WriteString("\n")
	}
}

func blockHeading(block *board.FilledBlock) string {
	if title := oneLine(block.Title); title != "" {
		return title
	}
	if phrase := board.SourcePhrase(&block.Source); phrase != "" {
		return strings.ToUpper(phrase[:1]) + phrase[1:]
	}

	return string(block.Type)
}

func writeFigures(sb *strings.Builder, figures []board.FilledFigure) {
	for idx := range figures {
		figure := &figures[idx]
		fmt.Fprintf(sb, "- %s: ", oneLine(figure.Label))
		if figure.Value == nil {
			sb.WriteString("could not be read\n")
			continue
		}
		sb.WriteString(board.FormatValue(figure.Value))
		if figure.Was != nil {
			fmt.Fprintf(sb, " (%s when answered)", board.FormatValue(figure.Was))
		}
		sb.WriteString("\n")
	}
}

func writeRows(sb *strings.Builder, block *board.FilledBlock, location *time.Location) {
	if block.Source.RecordKind == "" {
		return
	}
	noun := block.Source.RecordKind.Noun()
	header := []string{strings.ToUpper(noun[:1]) + noun[1:], "Detail", "Status"}
	if block.Measure != "" {
		header = append(header, measureHeading(block.Source.RecordKind, block.Measure))
	}
	writeRow(sb, header)
	writeRule(sb, len(header))
	hidden := 0
	for idx := range block.Rows {
		row := &block.Rows[idx]
		if !row.Readable {
			hidden++
			continue
		}
		cells := []string{row.Label, row.Subtitle, row.Status}
		if block.Measure != "" {
			cells = append(cells, MeasureText(row.Measure, location))
		}
		writeRow(sb, cells)
	}
	if hidden > 0 {
		fmt.Fprintf(
			sb,
			"\n%s could not be read.\n",
			board.RecordsPhrase(block.Source.RecordKind, hidden),
		)
	}
}

func measureHeading(kind recordkind.Kind, measure board.Measure) string {
	if spec, ok := measure.Spec(kind); ok && spec.Heading != "" {
		return spec.Heading
	}

	return string(measure)
}

func MeasureText(measure *board.RowMeasure, location *time.Location) string {
	if measure == nil {
		return ""
	}
	if measure.Value == nil {
		return "withheld"
	}
	text := board.FormatWords(&measure.Standing, location)
	if text == "" {
		text = board.FormatValue(measure.Value)
	}
	if fact := board.FormatWords(measure.Fact, location); fact != "" {
		text += " (" + fact + ")"
	}

	return text
}

func writeCompare(sb *strings.Builder, compare *board.FilledCompare) {
	if compare == nil {
		return
	}
	header := make([]string, 0, len(compare.Columns)+1)
	header = append(header, "")
	for idx := range compare.Columns {
		column := &compare.Columns[idx]
		if !column.Readable {
			header = append(header, "(could not be read)")
			continue
		}
		header = append(header, column.Label)
	}
	writeRow(sb, header)
	writeRule(sb, len(header))
	for idx := range compare.Lines {
		line := &compare.Lines[idx]
		cells := make([]string, 0, len(line.Cells)+1)
		cells = append(cells, line.Label)
		for cell, value := range line.Cells {
			text := board.FormatValue(value)
			if cell < len(line.Wins) && line.Wins[cell] {
				text = "**" + text + "**"
			}
			cells = append(cells, text)
		}
		writeRow(sb, cells)
	}
	if len(compare.Wins) > 0 {
		cells := make([]string, 0, len(compare.Wins)+1)
		cells = append(cells, "Rows won")
		for _, wins := range compare.Wins {
			cells = append(cells, strconv.Itoa(wins))
		}
		writeRow(sb, cells)
	}
}

func writeEvents(sb *strings.Builder, events []board.Event, location *time.Location) {
	for idx := range events {
		event := &events[idx]
		fmt.Fprintf(sb, "- %s: %s", board.FormatMoment(event.At, location), EventSentence(event))
		if event.Actor != "" {
			fmt.Fprintf(sb, " (%s)", oneLine(event.Actor))
		}
		sb.WriteString("\n")
	}
}

func writeChecks(sb *strings.Builder, checks []board.FilledCheck) {
	for _, tone := range []boardcatalog.Tone{boardcatalog.ToneBad, boardcatalog.ToneWarn, boardcatalog.ToneInfo, boardcatalog.ToneOK} {
		heading := false
		for idx := range checks {
			check := &checks[idx]
			if check.Tone != tone {
				continue
			}
			if !heading {
				fmt.Fprintf(sb, "**%s**\n\n", checkGroup(tone))
				heading = true
			}
			fmt.Fprintf(sb, "- %s", oneLine(check.Title))
			if check.Count != nil {
				fmt.Fprintf(sb, ": %s", board.FormatValue(check.Count))
				if check.Was != nil {
					fmt.Fprintf(sb, " (%s when answered)", board.FormatValue(check.Was))
				}
			}
			if detail := oneLine(check.Detail); detail != "" {
				fmt.Fprintf(sb, ". %s", detail)
			}
			if labels := rowLabels(check.Records); labels != "" {
				fmt.Fprintf(sb, " (%s)", labels)
			}
			sb.WriteString("\n")
		}
		if heading {
			sb.WriteString("\n")
		}
	}
}

func checkGroup(tone boardcatalog.Tone) string {
	switch tone {
	case boardcatalog.ToneBad:
		return "Blocks it"
	case boardcatalog.ToneWarn:
		return "Needs a look"
	case boardcatalog.ToneInfo:
		return "Worth knowing"
	case boardcatalog.ToneOK:
	}

	return "All clear"
}

func writeMeter(sb *strings.Builder, meter *board.FilledMeter) {
	if meter == nil {
		return
	}
	if meter.Percent != nil {
		sb.WriteString(board.FormatValue(meter.Percent))
		sb.WriteString("\n")
		return
	}
	fmt.Fprintf(sb, "%d of %d clear\n", meter.Clear, meter.Total)
}

func writeNote(sb *strings.Builder, f *board.Filled, block *board.FilledBlock) {
	sb.WriteString("\n**")
	sb.WriteString(NoteLabel(block.NoteTone))
	sb.WriteString(": ")
	sb.WriteString(oneLine(block.Title))
	sb.WriteString(".** ")
	sb.WriteString(wordsText(f, block.Text))
	if labels := rowLabels(block.Chips); labels != "" {
		fmt.Fprintf(sb, " (%s)", labels)
	}
	sb.WriteString("\n")
}

func NoteLabel(tone boardcatalog.Tone) string {
	switch tone {
	case boardcatalog.ToneBad:
		return "Blocks it"
	case boardcatalog.ToneWarn:
		return "Worth acting on"
	case boardcatalog.ToneInfo, boardcatalog.ToneOK:
	}

	return "What I'd do"
}

func EventSentence(event *board.Event) string {
	if comment := oneLine(event.Comment); comment != "" {
		return comment
	}
	operation := strings.ToLower(event.Operation)
	if operation == "update" && len(event.Changes) > 0 {
		fields := make([]string, 0, len(event.Changes))
		for field := range event.Changes {
			fields = append(fields, field)
		}
		slices.Sort(fields)
		return "Changed " + strings.Join(fields, ", ")
	}
	if operation == "" {
		return "Changed"
	}

	return strings.ToUpper(operation[:1]) + operation[1:]
}

func rowLabels(rows []board.Row) string {
	labels := make([]string, 0, len(rows))
	for idx := range rows {
		if rows[idx].Readable && rows[idx].Label != "" {
			labels = append(labels, oneLine(rows[idx].Label))
		}
	}

	return strings.Join(labels, ", ")
}

func wordsText(f *board.Filled, segments []board.FilledSegment) string {
	var sb strings.Builder
	for _, segment := range segments {
		if segment.Reading != "" {
			sb.WriteString(readingText(f.Reading(segment.Reading)))
			continue
		}
		if segment.Artifact != "" {
			sb.WriteString(linked(&segment))
			continue
		}
		sb.WriteString(segment.Text)
		if segment.Numeral > 0 {
			fmt.Fprintf(&sb, " [%d]", segment.Numeral)
		}
	}

	return strings.TrimSpace(sb.String())
}

const unreadReading = "—"

func readingText(reading *board.FilledReading) string {
	if reading == nil || reading.Answered() == nil {
		return unreadReading
	}
	text := board.FormatValue(reading.Answered())
	if reading.Was != nil && reading.Value != nil {
		text += " (now " + board.FormatValue(reading.Value) + ")"
	}

	return text
}

func Answer(f *board.Filled) string {
	var sb strings.Builder
	for _, segment := range f.Verdict {
		if segment.Artifact != "" {
			sb.WriteString(linked(&segment))
			continue
		}
		if segment.Reading == "" {
			sb.WriteString(segment.Text)
			continue
		}
		if reading := f.Reading(segment.Reading); reading != nil && reading.Answered() != nil {
			sb.WriteString(board.FormatValue(reading.Answered()))
			continue
		}
		sb.WriteString(unreadReading)
	}

	return oneLine(sb.String())
}

func linked(segment *board.FilledSegment) string {
	return "[" + segment.Text + "](artifact:" + segment.Artifact + ")"
}

func writeRow(sb *strings.Builder, cells []string) {
	sb.WriteString("|")
	for _, value := range cells {
		sb.WriteString(" ")
		sb.WriteString(strings.ReplaceAll(oneLine(value), "|", `\|`))
		sb.WriteString(" |")
	}
	sb.WriteString("\n")
}

func writeRule(sb *strings.Builder, columns int) {
	sb.WriteString("|")
	for range columns {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")
}

func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}
