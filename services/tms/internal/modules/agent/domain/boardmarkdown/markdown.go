package boardmarkdown

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
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
	if verdict := segmentsText(f.Verdict); verdict != "" {
		sb.WriteString("\n")
		sb.WriteString(verdict)
		sb.WriteString("\n")
	}
	for idx := range f.Blocks {
		writeBlock(&sb, &f.Blocks[idx], meta.Location)
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

func writeBlock(sb *strings.Builder, block *board.FilledBlock, location *time.Location) {
	switch block.Type {
	case board.BlockProse:
		sb.WriteString("\n")
		sb.WriteString(segmentsText(block.Text))
		sb.WriteString("\n")
		return
	case board.BlockNote:
		writeNote(sb, block)
		return
	case board.BlockFigures,
		board.BlockRecords,
		board.BlockCompare,
		board.BlockTable,
		board.BlockChart,
		board.BlockTimeline,
		board.BlockChecks,
		board.BlockMeter:
	}

	sb.WriteString("\n## ")
	sb.WriteString(strconv.Itoa(block.Numeral))
	sb.WriteString(". ")
	sb.WriteString(blockHeading(block))
	sb.WriteString("\n\n")
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
		writeRows(sb, block)
	case board.BlockCompare:
		writeCompare(sb, block.Compare)
	case board.BlockTable:
		if block.Table != nil {
			sb.WriteString(strings.TrimLeft(block.Table.Table.Markdown(), "\n"))
		}
	case board.BlockChart:
		if block.Chart != nil {
			sb.WriteString(strings.TrimLeft(block.Chart.Markdown(), "\n"))
		}
	case board.BlockTimeline:
		writeEvents(sb, block.Events, location)
	case board.BlockChecks:
		writeChecks(sb, block.Checks)
	case board.BlockMeter:
		writeMeter(sb, block.Meter)
	case board.BlockProse, board.BlockNote:
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

func writeRows(sb *strings.Builder, block *board.FilledBlock) {
	if block.Source.Resource == "" {
		return
	}
	noun := block.Source.Resource.Noun()
	header := []string{strings.ToUpper(noun[:1]) + noun[1:], "Detail", "Status"}
	if block.Measure != "" {
		header = append(header, measureHeading(block.Measure))
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
			cells = append(cells, measureText(row.Measure))
		}
		writeRow(sb, cells)
	}
	if hidden > 0 {
		fmt.Fprintf(
			sb,
			"\n%s could not be read.\n",
			board.RecordsPhrase(block.Source.Resource, hidden),
		)
	}
}

func measureHeading(measure board.Measure) string {
	switch measure {
	case board.MeasureLateness:
		return "Behind"
	case board.MeasureRevenue:
		return "Revenue"
	case board.MeasureAmountDue:
		return "Due"
	case board.MeasureTotal:
		return "Total"
	case board.MeasureAge:
		return "Age"
	case board.MeasureAmount:
		return "Amount"
	case board.MeasurePendingFor:
		return "Waiting"
	default:
		return string(measure)
	}
}

func measureText(measure *board.RowMeasure) string {
	if measure == nil {
		return ""
	}
	if measure.Measure != board.MeasureLateness {
		return board.FormatValue(&measure.Value)
	}
	text := board.FormatValue(&measure.Value)
	switch {
	case measure.State == board.MeasureDue:
		text = "not estimated"
	case measure.State == board.MeasureProjected:
		text = "estimated " + text + " late"
	case measure.Value.Number.IsZero():
		text = "on time"
	}
	if measure.Place != "" {
		text += " (" + measure.Place + ")"
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
	for _, tone := range []board.Tone{board.ToneBad, board.ToneWarn, board.ToneInfo, board.ToneOK} {
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

func checkGroup(tone board.Tone) string {
	switch tone {
	case board.ToneBad:
		return "Blocks it"
	case board.ToneWarn:
		return "Needs a look"
	case board.ToneInfo:
		return "Worth knowing"
	case board.ToneOK:
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

func writeNote(sb *strings.Builder, block *board.FilledBlock) {
	sb.WriteString("\n**")
	sb.WriteString(NoteLabel(block.NoteTone))
	sb.WriteString(": ")
	sb.WriteString(oneLine(block.Title))
	sb.WriteString(".** ")
	sb.WriteString(segmentsText(block.Text))
	if labels := rowLabels(block.Chips); labels != "" {
		fmt.Fprintf(sb, " (%s)", labels)
	}
	sb.WriteString("\n")
}

func NoteLabel(tone board.Tone) string {
	switch tone {
	case board.ToneBad:
		return "Blocks it"
	case board.ToneWarn:
		return "Worth acting on"
	case board.ToneInfo, board.ToneOK:
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

func segmentsText(segments []board.FilledSegment) string {
	var sb strings.Builder
	for _, segment := range segments {
		sb.WriteString(segment.Text)
		if segment.Numeral > 0 {
			fmt.Fprintf(&sb, " [%d]", segment.Numeral)
		}
	}

	return strings.TrimSpace(sb.String())
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
