package board

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

type MarkdownMeta struct {
	Question   string
	Author     string
	AnsweredAt int64
	Location   *time.Location
}

func (f *Filled) Markdown(meta MarkdownMeta) string {
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
	sb.WriteString(FormatMoment(f.ReadAt, meta.Location))
	sb.WriteString(".")
	if meta.AnsweredAt > 0 {
		sb.WriteString(" Answered")
		if author := oneLine(meta.Author); author != "" {
			sb.WriteString(" by ")
			sb.WriteString(author)
		}
		sb.WriteString(" at ")
		sb.WriteString(FormatMoment(meta.AnsweredAt, meta.Location))
		sb.WriteString(".")
	}
	sb.WriteString("\n")

	return sb.String()
}

func writeBlock(sb *strings.Builder, block *FilledBlock, location *time.Location) {
	switch block.Type {
	case BlockProse:
		sb.WriteString("\n")
		sb.WriteString(segmentsText(block.Text))
		sb.WriteString("\n")
		return
	case BlockNote:
		writeNote(sb, block)
		return
	default:
	}

	sb.WriteString("\n## ")
	sb.WriteString(strconv.Itoa(block.Numeral))
	sb.WriteString(". ")
	sb.WriteString(blockHeading(block))
	sb.WriteString("\n\n")
	if block.State == StateUnreadable {
		sb.WriteString("_Could not be read._\n")
		return
	}
	if block.State == StateEmpty {
		sb.WriteString("_Nothing to show._\n")
		return
	}

	switch block.Type {
	case BlockFigures:
		writeFigures(sb, block.Figures)
	case BlockRecords:
		writeRows(sb, block)
	case BlockCompare:
		writeCompare(sb, block.Compare)
	case BlockTable:
		if block.Table != nil {
			sb.WriteString(strings.TrimLeft(block.Table.Table.Markdown(), "\n"))
		}
	case BlockChart:
		if block.Chart != nil {
			sb.WriteString(strings.TrimLeft(block.Chart.Markdown(), "\n"))
		}
	case BlockTimeline:
		writeEvents(sb, block.Events, location)
	case BlockChecks:
		writeChecks(sb, block.Checks)
	case BlockMeter:
		writeMeter(sb, block.Meter)
	default:
	}
}

func blockHeading(block *FilledBlock) string {
	if title := oneLine(block.Title); title != "" {
		return title
	}
	if phrase := SourcePhrase(&block.Source); phrase != "" {
		return strings.ToUpper(phrase[:1]) + phrase[1:]
	}

	return string(block.Type)
}

func writeFigures(sb *strings.Builder, figures []FilledFigure) {
	for idx := range figures {
		figure := &figures[idx]
		fmt.Fprintf(sb, "- %s: ", oneLine(figure.Label))
		if figure.Value == nil {
			sb.WriteString("could not be read\n")
			continue
		}
		sb.WriteString(FormatValue(figure.Value))
		if figure.Was != nil {
			fmt.Fprintf(sb, " (%s when answered)", FormatValue(figure.Was))
		}
		sb.WriteString("\n")
	}
}

func writeRows(sb *strings.Builder, block *FilledBlock) {
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
		fmt.Fprintf(sb, "\n%s could not be read.\n", RecordsPhrase(block.Source.Resource, hidden))
	}
}

func measureHeading(measure Measure) string {
	switch measure {
	case MeasureLateness:
		return "Behind"
	case MeasureRevenue:
		return "Revenue"
	case MeasureAmountDue:
		return "Due"
	case MeasureTotal:
		return "Total"
	case MeasureAge:
		return "Age"
	case MeasureAmount:
		return "Amount"
	case MeasurePendingFor:
		return "Waiting"
	default:
		return string(measure)
	}
}

func measureText(measure *RowMeasure) string {
	if measure == nil {
		return ""
	}
	if measure.Measure == MeasureLateness && measure.Value.Number.IsZero() {
		return "on time"
	}

	return FormatValue(&measure.Value)
}

func writeCompare(sb *strings.Builder, compare *FilledCompare) {
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
			text := FormatValue(value)
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

func writeEvents(sb *strings.Builder, events []Event, location *time.Location) {
	for idx := range events {
		event := &events[idx]
		fmt.Fprintf(sb, "- %s: %s", FormatMoment(event.At, location), EventSentence(event))
		if event.Actor != "" {
			fmt.Fprintf(sb, " (%s)", oneLine(event.Actor))
		}
		sb.WriteString("\n")
	}
}

func writeChecks(sb *strings.Builder, checks []FilledCheck) {
	for _, tone := range []Tone{ToneBad, ToneWarn, ToneInfo, ToneOK} {
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
				fmt.Fprintf(sb, ": %s", FormatValue(check.Count))
				if check.Was != nil {
					fmt.Fprintf(sb, " (%s when answered)", FormatValue(check.Was))
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

func checkGroup(tone Tone) string {
	switch tone {
	case ToneBad:
		return "Blocks it"
	case ToneWarn:
		return "Needs a look"
	case ToneInfo:
		return "Worth knowing"
	default:
		return "Clear"
	}
}

func writeMeter(sb *strings.Builder, meter *FilledMeter) {
	if meter == nil {
		return
	}
	if meter.Percent != nil {
		sb.WriteString(FormatValue(meter.Percent))
		sb.WriteString("\n")
		return
	}
	fmt.Fprintf(sb, "%d of %d clear\n", meter.Clear, meter.Total)
}

func writeNote(sb *strings.Builder, block *FilledBlock) {
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

func NoteLabel(tone Tone) string {
	switch tone {
	case ToneBad:
		return "Blocks it"
	case ToneWarn:
		return "Worth acting on"
	default:
		return "What I'd do"
	}
}

func EventSentence(event *Event) string {
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

func rowLabels(rows []Row) string {
	labels := make([]string, 0, len(rows))
	for idx := range rows {
		if rows[idx].Readable && rows[idx].Label != "" {
			labels = append(labels, oneLine(rows[idx].Label))
		}
	}

	return strings.Join(labels, ", ")
}

func segmentsText(segments []FilledSegment) string {
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
