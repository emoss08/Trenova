package render

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/xuri/excelize/v2"
)

// excelSheetNameLimit is Excel's hard cap on a worksheet name.
const excelSheetNameLimit = 31

// excelSheetNameForbidden are the characters Excel rejects in a sheet name.
// A tile is titled by a human, so any of them can realistically appear.
const excelSheetNameForbidden = `[]:*?/\`

// DashboardSheet is one tile's finished result. Unlike a report render, a
// dashboard's tiles are already in memory — each is bounded by the preview row
// limit, and there are at most MaxDashboardTiles of them.
type DashboardSheet struct {
	Title     string
	Schema    []services.ReportResultColumn
	Rows      []services.ReportRow
	Totals    services.ReportRow
	Truncated bool
}

// DashboardRenderRequest asks for one workbook holding every tile of a
// dashboard, so the export is a single file rather than a folder of reports.
type DashboardRenderRequest struct {
	Title       string
	Description string
	GeneratedAt int64
	Timezone    string
	RequestedBy string
	Filters     map[string]any
	Params      map[string]any
	Sheets      []DashboardSheet
	Sink        io.Writer
}

// RenderDashboard writes one sheet per tile plus a leading info sheet naming
// the dashboard and the filter values it was run with — without those, two
// exports of the same dashboard are indistinguishable.
func (r *XLSXRenderer) RenderDashboard(req *DashboardRenderRequest) error {
	file := excelize.NewFile()
	defer file.Close()

	loc := metaLocation(&services.ReportRunMeta{Timezone: req.Timezone})

	if err := r.writeDashboardInfoSheet(file, req, loc); err != nil {
		return err
	}

	headerStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}

	used := make(map[string]int, len(req.Sheets))
	for i := range req.Sheets {
		sheet := &req.Sheets[i]
		name := uniqueSheetName(sheet.Title, i, used)
		if err = r.writeDashboardSheet(file, sheet, name, headerStyle, loc); err != nil {
			return err
		}
	}

	if index, idxErr := file.GetSheetIndex(dashboardInfoSheet); idxErr == nil && index >= 0 {
		file.SetActiveSheet(index)
	}

	return file.Write(req.Sink)
}

const dashboardInfoSheet = "Dashboard"

func (r *XLSXRenderer) writeDashboardInfoSheet(
	file *excelize.File,
	req *DashboardRenderRequest,
	loc *time.Location,
) error {
	if err := file.SetSheetName("Sheet1", dashboardInfoSheet); err != nil {
		return err
	}

	generatedAt := time.Unix(req.GeneratedAt, 0).In(loc).Format("2006-01-02 15:04:05 MST")
	rows := [][]any{
		{"Dashboard", req.Title},
		{"Description", req.Description},
		{"Generated At", generatedAt},
		{"Requested By", req.RequestedBy},
	}
	// Sorted so two exports of the same dashboard produce identical info
	// sheets; Go map iteration order would otherwise make them differ.
	for _, name := range sortedKeys(req.Filters) {
		rows = append(rows, []any{"Filter: " + name, fmt.Sprint(req.Filters[name])})
	}
	for _, name := range sortedKeys(req.Params) {
		rows = append(rows, []any{"Parameter: " + name, fmt.Sprint(req.Params[name])})
	}

	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return err
		}
		if err = file.SetSheetRow(dashboardInfoSheet, cell, &row); err != nil {
			return err
		}
	}

	return nil
}

func (r *XLSXRenderer) writeDashboardSheet(
	file *excelize.File,
	sheet *DashboardSheet,
	sheetName string,
	headerStyle int,
	loc *time.Location,
) error {
	// Every tile carries its own schema, so styles are registered per sheet
	// rather than once for the workbook — a money column on one tile and a
	// duration on another must not share a number format.
	columnStyles, err := newColumnStyles(file, sheet.Schema, false)
	if err != nil {
		return err
	}
	totalsStyles, err := newColumnStyles(file, sheet.Schema, true)
	if err != nil {
		return err
	}
	toneStyles, err := newToneStyles(file, sheet.Schema)
	if err != nil {
		return err
	}
	styles := &sheetStyles{
		header:  headerStyle,
		columns: columnStyles,
		totals:  totalsStyles,
		tones:   toneStyles,
	}

	writer, err := r.newDataSheet(file, sheetName, sheet.Schema, headerStyle)
	if err != nil {
		return err
	}

	cells := make([]any, len(sheet.Schema))
	rowNum := 2
	for _, row := range sheet.Rows {
		for i := range sheet.Schema {
			var value any
			if i < len(row) {
				value = row[i]
			}
			cells[i] = xlsxCell(
				&sheet.Schema[i], value, loc, styles.styleFor(i, &sheet.Schema[i], value),
			)
		}
		if err = r.writeRow(writer, rowNum, cells); err != nil {
			return err
		}
		rowNum++
	}

	if len(sheet.Totals) > 0 {
		totals := make([]any, len(sheet.Schema))
		for i := range sheet.Schema {
			if i >= len(sheet.Totals) || sheet.Totals[i] == nil {
				continue
			}
			totals[i] = xlsxCell(&sheet.Schema[i], sheet.Totals[i], loc, styles.totals[i])
		}
		if sheet.Totals[0] == nil && len(totals) > 0 {
			totals[0] = excelize.Cell{StyleID: headerStyle, Value: totalsLabel}
		}
		if err = r.writeRow(writer, rowNum, totals); err != nil {
			return err
		}
		rowNum++
	}

	if sheet.Truncated {
		if err = r.writeRow(writer, rowNum, []any{truncationNotice}); err != nil {
			return err
		}
	}

	return writer.Flush()
}

// uniqueSheetName makes a human tile title safe and unique as a worksheet
// name. Excel silently corrupts a workbook with a duplicate or over-long
// sheet name, and dashboards routinely carry two tiles of the same report.
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

	// Suffix collisions as " (2)", trimming the base so the result still fits.
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

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:limit]))
}

func sortedKeys(values map[string]any) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
