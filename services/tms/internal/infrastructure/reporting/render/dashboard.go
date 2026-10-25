package render

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/xuri/excelize/v2"
)

const (
	excelSheetNameLimit     = 31
	excelSheetNameForbidden = `[]:*?/\`
)

type DashboardSheet struct {
	Title  string
	Result table.Reader
}

type DashboardRenderRequest struct {
	Title       string
	Description string
	GeneratedAt int64
	Timezone    string
	Locale      string
	RequestedBy string
	Filters     []DashboardValue
	Params      []DashboardValue
	Sheets      []DashboardSheet
	Sink        io.Writer
}

type DashboardValue struct {
	Name  string
	Value any
}

func RenderDashboard(ctx context.Context, req *DashboardRenderRequest) error {
	file := excelize.NewFile()
	defer file.Close()

	locale := i18n.FromContext(ctx)
	loc := Location(req.Timezone)
	text := Text(ctx)
	infoSheet := i18n.Translate(locale, "Dashboard")
	if err := writeDashboardInfo(file, req, infoSheet, &text, loc); err != nil {
		return err
	}

	bold, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}

	opts := format.Options{Locale: Language(req.Locale), Location: loc}
	used := make(map[string]int, len(req.Sheets))
	for i := range req.Sheets {
		name := uniqueSheetName(req.Sheets[i].Title, i, used)
		if err = writeDashboardSheet(ctx, file, &dashboardSheet{
			name:   name,
			result: req.Sheets[i].Result,
			bold:   bold,
			opts:   opts,
			total:  text.Total,
			sub:    text.Subtotal,
			notice: text.RowsTruncated,
		}); err != nil {
			return err
		}
	}

	if index, idxErr := file.GetSheetIndex(infoSheet); idxErr == nil && index >= 0 {
		file.SetActiveSheet(index)
	}

	return file.Write(req.Sink)
}

func writeDashboardInfo(
	file *excelize.File,
	req *DashboardRenderRequest,
	sheet string,
	text *export.Text,
	loc *time.Location,
) error {
	if err := file.SetSheetName("Sheet1", sheet); err != nil {
		return err
	}

	rows := [][]any{
		{sheet, req.Title},
		{text.Description, req.Description},
		{text.GeneratedAt, time.Unix(req.GeneratedAt, 0).In(loc).Format("2006-01-02 15:04:05 MST")},
		{text.RequestedBy, req.RequestedBy},
	}
	for _, f := range req.Filters {
		rows = append(rows, []any{f.Name, fmt.Sprint(f.Value)})
	}
	for _, p := range req.Params {
		rows = append(rows, []any{p.Name, fmt.Sprint(p.Value)})
	}

	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return err
		}
		if err = file.SetSheetRow(sheet, cell, &row); err != nil {
			return err
		}
	}
	return nil
}

type dashboardSheet struct {
	name   string
	result table.Reader
	bold   int
	opts   format.Options
	total  string
	sub    string
	notice string
}

type sheetColumn struct {
	col       table.Column
	formatter *format.Formatter
	native    bool
	style     int
	boldStyle int
}

func writeDashboardSheet(ctx context.Context, file *excelize.File, s *dashboardSheet) error {
	if _, err := file.NewSheet(s.name); err != nil {
		return err
	}
	writer, err := file.NewStreamWriter(s.name)
	if err != nil {
		return err
	}

	columns, err := sheetColumns(file, s.result.Columns(), s.opts)
	if err != nil {
		return err
	}

	header := make([]any, len(columns))
	for i := range columns {
		header[i] = excelize.Cell{StyleID: s.bold, Value: columns[i].col.Label}
	}
	if err = writer.SetRow("A1", header); err != nil {
		return err
	}

	rowNum := 2
	for {
		batch, nextErr := s.result.Next(ctx)
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nextErr
		}
		for i := range batch.Len {
			cells := s.row(columns, batch, i)
			cell, cellErr := excelize.CoordinatesToCellName(1, rowNum)
			if cellErr != nil {
				return cellErr
			}
			if err = writer.SetRow(cell, cells); err != nil {
				return err
			}
			rowNum++
		}
	}

	if s.result.Truncated() {
		cell, cellErr := excelize.CoordinatesToCellName(1, rowNum)
		if cellErr != nil {
			return cellErr
		}
		if err = writer.SetRow(cell, []any{s.notice}); err != nil {
			return err
		}
	}

	return writer.Flush()
}

func (s *dashboardSheet) row(columns []sheetColumn, b *table.Batch, i int) []any {
	level := b.Level(i)
	summary := level != table.Detail
	cells := make([]any, len(columns))
	for c := range columns {
		column := &columns[c]
		vector := &b.Columns[c]
		style := column.style
		if summary {
			style = column.boldStyle
		}
		switch {
		case vector.Null(i):
			cells[c] = excelize.Cell{StyleID: style}
		case column.native:
			cells[c] = excelize.Cell{
				StyleID: style,
				Value:   nativeValue(column.col.Type, vector, i, s.opts.Location),
			}
		default:
			text, _ := column.formatter.Append(nil, vector, i)
			cells[c] = excelize.Cell{StyleID: style, Value: string(text)}
		}
	}
	if summary && len(columns) > 0 && b.Columns[0].Null(i) {
		label := s.sub
		if level == table.Total {
			label = s.total
		}
		cells[0] = excelize.Cell{StyleID: s.bold, Value: label}
	}
	return cells
}

func sheetColumns(
	file *excelize.File,
	cols []table.Column,
	opts format.Options,
) ([]sheetColumn, error) {
	out := make([]sheetColumn, len(cols))
	for i := range cols {
		f := format.New(cols[i].Type, cols[i].Display, opts)
		code, native := f.Excel()
		out[i] = sheetColumn{col: cols[i], formatter: f, native: native}
		var numFmt *string
		if native && code != "" {
			numFmt = &code
		}
		style, err := file.NewStyle(&excelize.Style{CustomNumFmt: numFmt})
		if err != nil {
			return nil, err
		}
		boldStyle, err := file.NewStyle(
			&excelize.Style{CustomNumFmt: numFmt, Font: &excelize.Font{Bold: true}},
		)
		if err != nil {
			return nil, err
		}
		out[i].style, out[i].boldStyle = style, boldStyle
	}
	return out, nil
}

func nativeValue(t value.Type, v *value.Vector, i int, loc *time.Location) any {
	switch t {
	case value.Int:
		return v.Int64[i]
	case value.Epoch:
		return time.Unix(v.Int64[i], 0).In(loc)
	case value.Decimal:
		f, err := v.Numeric[i].Float64()
		if err != nil {
			return v.Numeric[i].String()
		}
		return f
	case value.Float:
		return v.Float64[i]
	case value.Bool:
		return v.Bool[i]
	case value.Time:
		return v.Time[i].In(loc)
	case value.String, value.Enum, value.Ref:
		return v.String[i]
	case value.JSON:
		return string(v.JSON[i])
	default:
		return nil
	}
}

func uniqueSheetName(title string, index int, used map[string]int) string {
	base := strings.TrimSpace(title)
	for _, forbidden := range excelSheetNameForbidden {
		base = strings.ReplaceAll(base, string(forbidden), " ")
	}
	base = strings.TrimSpace(base)
	if base == "" {
		base = fmt.Sprintf("Tile %d", index+1)
	}
	base = truncateRunes(base, excelSheetNameLimit)

	key := strings.ToLower(base)
	count, seen := used[key]
	if !seen {
		used[key] = 1
		return base
	}

	for {
		count++
		suffix := fmt.Sprintf(" (%d)", count)
		room := excelSheetNameLimit - utf8.RuneCountInString(suffix)
		candidate := truncateRunes(base, room) + suffix
		if _, taken := used[strings.ToLower(candidate)]; !taken {
			used[key] = count
			used[strings.ToLower(candidate)] = 1
			return candidate
		}
	}
}

func truncateRunes(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:limit]))
}
