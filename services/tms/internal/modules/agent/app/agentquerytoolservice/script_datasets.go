package agentquerytoolservice

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emoss08/keel/analytics/export"
	"github.com/emoss08/keel/analytics/format"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/scriptports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	paramScriptDatasets    = "datasets"
	paramDatasetName       = "name"
	paramDatasetRun        = "reportRunId"
	paramDatasetDocument   = "documentId"
	paramDatasetDefinition = "definition"

	extensionCSV  = "csv"
	extensionXLSX = "xlsx"
	extensionText = "txt"
)

var (
	errDatasetsNeedAPerson = errors.New(
		"a script reads report runs and files a person gave you, and this run has no person",
	)
	errDatasetsReportRead = errors.New(
		"you do not have permission to read reports, so their results cannot be read",
	)
)

type datasetRequest struct {
	name       string
	reportRun  pulid.ID
	documentID pulid.ID
	// definition is a report the engine runs now, grouped and totalled over
	// every record, rather than a run already made.
	definition *ir.Definition
}

func datasetsFrom(raw any) ([]datasetRequest, error) {
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a list", paramScriptDatasets)
	}
	if len(items) > analysis.MaxDatasets {
		return nil, fmt.Errorf("give at most %d datasets", analysis.MaxDatasets)
	}

	requests := make([]datasetRequest, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for idx, item := range items {
		fields, isObject := item.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("dataset %d must be an object with a name", idx+1)
		}
		name := strings.TrimSpace(optionalString(fields, paramDatasetName))
		if err := analysis.CheckDatasetName(name); err != nil {
			return nil, err
		}
		if _, dup := seen[name]; dup {
			return nil, fmt.Errorf("two datasets are named %s; give each its own name", name)
		}
		seen[name] = struct{}{}
		run, hasRun, err := optionalPulid(fields, paramDatasetRun)
		if err != nil {
			return nil, err
		}
		document, hasDocument, err := optionalPulid(fields, paramDatasetDocument)
		if err != nil {
			return nil, err
		}
		var definition *ir.Definition
		if raw, given := fields[paramDatasetDefinition]; given && raw != nil {
			if definition, err = reporting.DecodeDefinition(raw); err != nil {
				return nil, fmt.Errorf("dataset %s: %w", name, err)
			}
		}
		if sources := countTrue(hasRun, hasDocument, definition != nil); sources != 1 {
			return nil, fmt.Errorf(
				"dataset %s needs exactly one of reportRunId, documentId or definition", name)
		}
		requests = append(requests, datasetRequest{
			name: name, reportRun: run, documentID: document, definition: definition,
		})
	}

	return requests, nil
}

// scriptDatasets reads each dataset a script asked for as the caller: a
// report run only if this person ran it and may read reports, a file only if
// they attached it to this conversation. Nothing is read for anyone else.
type scriptDatasets struct {
	reports     reportRunner
	files       documentports.ConversationFileReader
	permissions permissionports.PermissionEngine
}

type resolvedDatasets struct {
	datasets    []scriptports.Dataset
	attachments []agent.RecordRef
}

func (d *scriptDatasets) resolve(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	requests []datasetRequest,
) (*resolvedDatasets, error) {
	resolved := &resolvedDatasets{
		datasets:    make([]scriptports.Dataset, 0, len(requests)),
		attachments: make([]agent.RecordRef, 0),
	}
	if len(requests) == 0 {
		return resolved, nil
	}

	person := personOf(params.Actor)
	if person.IsNil() {
		return nil, errDatasetsNeedAPerson
	}

	if slices.ContainsFunc(requests, readsReport) {
		if err := d.authorizeReports(ctx, params.Actor); err != nil {
			return nil, err
		}
	}

	total := 0
	for idx := range requests {
		request := &requests[idx]
		var (
			dataset *scriptports.Dataset
			size    int
			err     error
		)
		switch {
		case request.definition != nil:
			dataset, size, err = d.definitionDataset(ctx, params, request)
		case readsReport(*request):
			dataset, size, err = d.reportDataset(ctx, params, person, request)
		default:
			dataset, size, err = d.fileDataset(ctx, params, person, request)
			if err == nil {
				resolved.attachments = append(resolved.attachments, agent.RecordRef{
					EntityType: agent.TaintEntityDocument,
					ID:         request.documentID.String(),
				})
			}
		}
		if err != nil {
			return nil, err
		}
		total += size
		if total > analysis.MaxInputBytes {
			return nil, errors.New("the datasets together are larger than 40 MB; " +
				"narrow them, or read them one at a time")
		}
		resolved.datasets = append(resolved.datasets, *dataset)
	}

	return resolved, nil
}

func readsReport(request datasetRequest) bool {
	return request.reportRun.IsNotNil() || request.definition != nil
}

func countTrue(flags ...bool) int {
	count := 0
	for _, flag := range flags {
		if flag {
			count++
		}
	}

	return count
}

// definitionDataset is a report the engine runs now from its definition, with
// the person's report access, every row of the grouped result typed by its
// columns. The engine groups and totals over every record and returns at most
// its preview bound of rows, so a definition that groups (by month, by fuel
// type) reads tens of thousands of records as a few dozen rows; one that lists
// records is refused when it has more rows than that, as a script should then
// group it.
func (d *scriptDatasets) definitionDataset(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	request *datasetRequest,
) (*scriptports.Dataset, int, error) {
	label := "a report on " + request.definition.Entity
	values := normalizeReportParameters(request.definition, nil)
	if err := requireReportParameters(label, request.definition, values); err != nil {
		return nil, 0, err
	}
	result, err := d.reports.Preview(ctx, &reporting.PreviewRequest{
		Request:    reportingRequestFor(params),
		Definition: request.definition,
		Params:     values,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("dataset %s: the report could not be run: %w",
			request.name, err)
	}
	if result.Truncated {
		return nil, 0, fmt.Errorf("dataset %s has more rows than a report from a definition "+
			"returns (%d); group it (a bucket on a date, a measure such as sum) so the engine "+
			"totals the records, or narrow its filters", request.name, len(result.Rows))
	}

	location := timezoneOf(params.Timezone)
	columns := make([]scriptports.DatasetColumn, 0, len(result.Columns))
	for idx := range result.Columns {
		column := &result.Columns[idx]
		kind := analysis.KindOf(
			column.Type.String(), reportFormat(column), string(column.Display.DateStyle),
		)
		columns = append(columns, scriptports.DatasetColumn{
			Name:     column.Label,
			Kind:     kind,
			Currency: analysis.Currency(kind, column.Display.Currency),
		})
	}
	size := 0
	rows := make([][]any, 0, len(result.Rows))
	for _, cells := range result.Rows {
		row := make([]any, len(result.Columns))
		for idx := range result.Columns {
			if idx < len(cells) {
				text := previewCellText(cells[idx])
				size += len(text)
				row[idx] = rawValue(text, &result.Columns[idx], location)
			}
		}
		rows = append(rows, row)
	}
	if size > analysis.MaxDatasetBytes {
		return nil, 0, fmt.Errorf("dataset %s is larger than 20 MB; group it or narrow its "+
			"filters", request.name)
	}

	return &scriptports.Dataset{
		Name:    request.name,
		Source:  analysis.DatasetReportDefinition,
		Label:   label,
		Columns: columns,
		Rows:    rows,
	}, size, nil
}

// previewCellText is a preview cell as the raw export writes it, so it is
// typed by its column as a run's rows are.
func previewCellText(cell any) string {
	switch value := cell.(type) {
	case nil:
		return ""
	case string:
		return value
	case int64:
		return strconv.FormatInt(value, 10)
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(value)
	default:
		return fmt.Sprint(value)
	}
}

func (d *scriptDatasets) authorizeReports(
	ctx context.Context,
	actor *permissionports.RequestActor,
) error {
	result, err := d.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceReport.String(),
		Operation:      permission.OpRead,
	})
	if err != nil {
		return fmt.Errorf("reading reports could not be authorized: %w", err)
	}
	if !result.Allowed {
		return errDatasetsReportRead
	}

	return nil
}

// reportDataset is a report run's every row, typed by its columns and headed
// by their labels, as report_run showed them. Totals are left out.
func (d *scriptDatasets) reportDataset(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	person pulid.ID,
	request *datasetRequest,
) (*scriptports.Dataset, int, error) {
	runRequest := &reporting.GetRunRequest{
		Request: reportingRequestFor(params),
		RunID:   request.reportRun,
	}
	notYours := fmt.Errorf(
		"report run %s is not one you ran for this person; run the report with report_run and use its runId",
		request.reportRun,
	)

	run, err := d.reports.GetRun(ctx, runRequest)
	if err != nil {
		if isNotFound(err) {
			return nil, 0, notYours
		}
		return nil, 0, err
	}
	if run.RequestedByID != person {
		return nil, 0, notYours
	}

	envelope, err := d.reports.ReadRunRows(ctx, runRequest)
	if err != nil {
		return nil, 0, err
	}
	var content bytes.Buffer
	if _, err = export.CSV(
		ctx, &content, envelope.Table.Reader(), export.CSVOptions{Raw: true},
	); err != nil {
		return nil, 0, fmt.Errorf("the rows of report run %s could not be read: %w",
			request.reportRun, err)
	}
	if content.Len() > analysis.MaxDatasetBytes {
		return nil, 0, fmt.Errorf("report run %s is larger than 20 MB; narrow the report's "+
			"filters or columns and run it again", request.reportRun)
	}

	location := timezoneOf(params.Timezone)
	columns := make([]scriptports.DatasetColumn, 0, len(envelope.Table.Columns))
	for idx := range envelope.Table.Columns {
		column := &envelope.Table.Columns[idx]
		kind := analysis.KindOf(
			column.Type.String(), reportFormat(column), string(column.Display.DateStyle),
		)
		columns = append(columns, scriptports.DatasetColumn{
			Name:     column.Label,
			Kind:     kind,
			Currency: analysis.Currency(kind, column.Display.Currency),
		})
	}
	rows, err := readRawRows(content.Bytes(), envelope.Table.Columns, location)
	if err != nil {
		return nil, 0, fmt.Errorf("the rows of report run %s could not be read: %w",
			request.reportRun, err)
	}

	return &scriptports.Dataset{
		Name:     request.name,
		Source:   analysis.DatasetReportRun,
		SourceID: request.reportRun,
		Label:    envelope.Meta.Title,
		Columns:  columns,
		Rows:     rows,
	}, content.Len(), nil
}

func reportFormat(column *table.Column) string {
	switch column.Display.Style {
	case format.StyleCurrency:
		return "money"
	case format.StylePercent:
		return fieldPercent
	case format.StyleAuto, format.StyleNumber, format.StyleDuration, format.StyleDate,
		format.StyleBool, format.StyleText:
		return ""
	}

	return ""
}

// readRawRows types a report's raw CSV by its columns: numbers as numbers,
// times as ISO 8601 in the organization's zone, flags as flags.
func readRawRows(
	content []byte,
	columns []table.Column,
	location *time.Location,
) ([][]any, error) {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.FieldsPerRecord = -1
	if _, err := reader.Read(); err != nil {
		if errors.Is(err, io.EOF) {
			return [][]any{}, nil
		}
		return nil, err
	}

	rows := make([][]any, 0)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			return nil, err
		}
		if len(rows) >= analysis.MaxDatasetRows {
			return nil, fmt.Errorf("the run has more than %d rows; narrow the report's filters",
				analysis.MaxDatasetRows)
		}
		row := make([]any, len(columns))
		for idx := range columns {
			if idx < len(record) {
				row[idx] = rawValue(record[idx], &columns[idx], location)
			}
		}
		rows = append(rows, row)
	}
}

func rawValue(cell string, column *table.Column, location *time.Location) any {
	if cell == "" {
		return nil
	}
	switch column.Type.String() {
	case "int", "decimal", "float":
		if number, err := strconv.ParseFloat(cell, 64); err == nil {
			return number
		}
	case "epoch":
		if seconds, err := strconv.ParseInt(cell, 10, 64); err == nil {
			at := time.Unix(seconds, 0).In(location)
			if string(column.Display.DateStyle) == "date" {
				return at.Format(time.DateOnly)
			}
			return at.Format(time.RFC3339)
		}
	case "time":
		if at, err := time.Parse(time.RFC3339Nano, cell); err == nil {
			return at.In(location).Format(time.RFC3339)
		}
	case "bool":
		if flag, err := strconv.ParseBool(cell); err == nil {
			return flag
		}
	}

	return cell
}

// fileDataset is an attached CSV or XLSX file as rows headed by its header
// row, or a text file as its text.
func (d *scriptDatasets) fileDataset(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
	person pulid.ID,
	request *datasetRequest,
) (*scriptports.Dataset, int, error) {
	notAttached := fmt.Errorf(attachmentOutsideThread, request.documentID)
	if params.ThreadID.IsNil() {
		return nil, 0, notAttached
	}

	file, err := d.files.ReadConversationFile(ctx, documentports.GetDocumentByIDRequest{
		ID:         request.documentID,
		TenantInfo: tenantOf(params),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, 0, notAttached
		}
		return nil, 0, err
	}
	if file.Document == nil || !file.Document.AttachedBy(params.ThreadID, person) {
		return nil, 0, notAttached
	}
	if len(file.Data) > analysis.MaxDatasetBytes {
		return nil, 0, fmt.Errorf("%s is larger than 20 MB, which a script cannot read",
			file.Document.OriginalName)
	}

	dataset := &scriptports.Dataset{
		Name:     request.name,
		Source:   analysis.DatasetAttachment,
		SourceID: request.documentID,
		Label:    file.Document.OriginalName,
	}
	extension := strings.TrimPrefix(strings.ToLower(path.Ext(file.Document.OriginalName)), ".")
	switch extension {
	case extensionText:
		if !utf8.Valid(file.Data) {
			return nil, 0, fmt.Errorf("%s is not text a script can read",
				file.Document.OriginalName)
		}
		text := string(file.Data)
		dataset.Text = &text
		return dataset, len(file.Data), nil
	case extensionCSV, extensionXLSX:
	default:
		return nil, 0, fmt.Errorf("%s is a .%s file, which a script cannot read; use a CSV, "+
			"XLSX or text file", file.Document.OriginalName, extension)
	}

	limits := rateimport.Limits{
		MaxRows:       analysis.MaxDatasetRows,
		MaxColumns:    analysis.MaxDatasetColumns,
		MaxSheetBytes: analysis.MaxDatasetBytes,
	}
	var sheet *rateimport.Sheet
	if extension == extensionXLSX {
		sheet, err = rateimport.ReadXLSX(file.Data, limits)
	} else {
		sheet, err = rateimport.ReadCSV(file.Data, limits)
	}
	switch {
	case errors.Is(err, rateimport.ErrNoRows), errors.Is(err, rateimport.ErrNoTable):
		return nil, 0, fmt.Errorf("%s has no table a script can read: no header row with "+
			"rows under it", file.Document.OriginalName)
	case err != nil:
		return nil, 0, fmt.Errorf("%s could not be read: %w", file.Document.OriginalName, err)
	}
	dataset.Columns, dataset.Rows = typedSheet(sheet)

	return dataset, len(file.Data), nil
}

// typedSheet heads a sheet's columns by its header row and reads each column
// as numbers when every value in it is one, as ISO dates when every value is
// one, and as text otherwise.
func typedSheet(
	sheet *rateimport.Sheet,
) (columns []scriptports.DatasetColumn, rows [][]any) {
	columns = make([]scriptports.DatasetColumn, len(sheet.Headers))
	for idx, header := range sheet.Headers {
		name := strings.TrimSpace(header)
		if name == "" {
			name = "Column " + strconv.Itoa(idx+1)
		}
		columns[idx] = scriptports.DatasetColumn{Name: name, Kind: columnKind(sheet.Rows, idx)}
	}

	rows = make([][]any, 0, len(sheet.Rows))
	for _, record := range sheet.Rows {
		row := make([]any, len(columns))
		for idx := range columns {
			if idx >= len(record) {
				continue
			}
			row[idx] = sheetValue(strings.TrimSpace(record[idx]), columns[idx].Kind)
		}
		rows = append(rows, row)
	}

	return columns, rows
}

func columnKind(rows [][]string, idx int) analysis.ValueKind {
	numbers, dates, filled := true, true, false
	for _, record := range rows {
		if idx >= len(record) {
			continue
		}
		cell := strings.TrimSpace(record[idx])
		if cell == "" {
			continue
		}
		filled = true
		if _, err := strconv.ParseFloat(numericText(cell), 64); err != nil {
			numbers = false
		}
		if _, err := time.Parse(time.DateOnly, cell); err != nil {
			dates = false
		}
		if !numbers && !dates {
			return analysis.ValueText
		}
	}
	switch {
	case !filled:
		return analysis.ValueText
	case numbers:
		return analysis.ValueNumber
	case dates:
		return analysis.ValueDate
	default:
		return analysis.ValueText
	}
}

func sheetValue(cell string, kind analysis.ValueKind) any {
	if cell == "" {
		return nil
	}
	if kind == analysis.ValueNumber {
		if number, err := strconv.ParseFloat(numericText(cell), 64); err == nil {
			return number
		}
	}

	return cell
}

// numericText drops what a spreadsheet writes around a number: thousands
// separators and a dollar sign.
func numericText(cell string) string {
	return strings.NewReplacer(",", "", "$", "").Replace(cell)
}

func timezoneOf(name string) *time.Location {
	if name == "" {
		return time.UTC
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}

	return location
}
