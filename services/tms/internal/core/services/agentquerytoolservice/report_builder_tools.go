package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

// maxPreviewRows bounds what preview_report hands the model. A preview exists
// to check the shape of a report — the columns, a few rows, whether the filter
// bites — and twenty rows show that. The whole result is what run_report is for.
const maxPreviewRows = 20

// reportSource is a report the tools can act on, whichever way it was named:
// a built-in one by key or a saved one by definition id. Everything downstream
// reads this rather than telling the two apart again.
type reportSource struct {
	Name          string
	Key           string
	DefinitionID  pulid.ID
	Definition    *report.Definition
	DefaultFormat report.Format
	Description   string
	Category      string
	Kind          string
	Visibility    report.Visibility
	Status        report.DefinitionStatus
	OwnerID       pulid.ID
}

// Editable says whether the actor may change the report. A built-in report is
// nobody's to edit — it is forked instead — and a saved one belongs to its owner.
func (s reportSource) Editable(actor pulid.ID) bool {
	return !s.DefinitionID.IsNil() && s.OwnerID == actor
}

var errNoReportNamed = errors.New(
	"name the report: reportKey for a built-in report or definitionId for a saved one, " +
		"both from list_reports",
)

// resolveReportSource reads the report a call names. A definitionId wins
// when both are given, because it is the more specific of the two: a saved
// report may carry the key it was forked from, and the caller means the fork.
func resolveReportSource(
	ctx context.Context,
	reports reportRunner,
	params serviceports.QueryToolParams,
) (reportSource, error) {
	if raw := optionalString(params.Params, "definitionId"); raw != "" {
		id, err := pulid.Parse(raw)
		if err != nil {
			return reportSource{}, fmt.Errorf("parameter \"definitionId\" is not a valid id: %w", err)
		}

		definition, err := reports.GetDefinition(ctx, &reporting.GetDefinitionRequest{
			Request:      reportingRequestFor(params),
			DefinitionID: id,
		})
		if err != nil {
			if errortypes.IsNotFoundError(err) {
				return reportSource{}, fmt.Errorf(
					"there is no saved report with the id %q that you can see; call "+
						"list_reports for the ones that exist",
					raw,
				)
			}

			return reportSource{}, err
		}

		return sourceFromDefinition(definition), nil
	}

	key, err := reportKeyOf(params.Params)
	if err != nil {
		return reportSource{}, errNoReportNamed
	}

	entry, err := reports.GetCanned(key)
	if err != nil {
		return reportSource{}, fmt.Errorf(
			"there is no report with the key %q; call list_reports for the keys that exist",
			key,
		)
	}

	return reportSource{
		Name:          entry.Name,
		Key:           entry.Key,
		Definition:    entry.Definition,
		DefaultFormat: entry.DefaultFormat,
		Description:   entry.Description,
		Category:      entry.Category,
		Kind:          "canned",
	}, nil
}

func sourceFromDefinition(definition *report.ReportDefinition) reportSource {
	return reportSource{
		Name:          definition.Name,
		Key:           definition.CannedKey,
		DefinitionID:  definition.ID,
		Definition:    definition.Definition,
		DefaultFormat: definition.DefaultFormat,
		Description:   definition.Description,
		Category:      definition.Category,
		Kind:          string(definition.Kind),
		Visibility:    definition.Visibility,
		Status:        definition.Status,
		OwnerID:       definition.OwnerID,
	}
}

// catalogAccess answers which datasets and fields the actor may read, the
// same way the report builder's own catalog does, so a model sees exactly
// what the person driving it would see in the builder.
type catalogAccess struct {
	permissions serviceports.PermissionEngine
	sensitivity *permission.Registry
}

func newCatalogAccess(permissions serviceports.PermissionEngine) catalogAccess {
	return catalogAccess{permissions: permissions, sensitivity: permission.NewRegistry()}
}

// entity resolves one dataset for the actor: the catalog entry and what they
// may read of it. A dataset the actor cannot read at all is reported as absent,
// because a model told "you may not read invoices" will keep trying to.
func (c catalogAccess) entity(
	ctx context.Context,
	params serviceports.QueryToolParams,
	entity *reportcatalog.Entity,
) (*serviceports.ResourcePermissionDetail, bool, error) {
	if c.permissions == nil {
		return nil, false, errors.New("datasets cannot be described because authorization is unavailable")
	}

	detail, err := c.permissions.GetResourcePermissions(
		ctx, params.Actor.UserID, params.OrganizationID, entity.Resource.String(),
	)
	if err != nil {
		return nil, false, fmt.Errorf("resolve permissions for %q: %w", entity.Key, err)
	}

	for _, op := range detail.Operations {
		if op == permission.OpRead {
			return detail, true, nil
		}
	}

	return nil, false, nil
}

func (c catalogAccess) fieldAccessible(
	entity *reportcatalog.Entity,
	field *reportcatalog.Field,
	detail *serviceports.ResourcePermissionDetail,
) (permission.FieldSensitivity, bool) {
	sensitivity := c.sensitivity.GetFieldSensitivity(entity.Resource.String(), field.Key)

	accessible := detail.MaxSensitivity.CanAccess(sensitivity)
	if accessible && len(detail.AccessibleFields) > 0 {
		accessible = false
		for _, key := range detail.AccessibleFields {
			if key == field.Key {
				accessible = true
				break
			}
		}
	}

	return sensitivity, accessible
}

type datasetEdgeRow struct {
	Name        string `json:"name"`
	Label       string `json:"label,omitempty"`
	Target      string `json:"target"`
	Cardinality string `json:"cardinality"`
	// TargetFields are the keys on the dataset the edge leads to, given by
	// describe_report_dataset so a report can reach one edge out without a
	// second call. A model that had only the edge's name guessed the field
	// on the far side, and guessed wrong.
	TargetFields []string `json:"targetFields,omitempty"`
}

type datasetRow struct {
	Dataset     string           `json:"dataset"`
	Label       string           `json:"label"`
	PluralLabel string           `json:"pluralLabel,omitempty"`
	Description string           `json:"description,omitempty"`
	Category    string           `json:"category,omitempty"`
	FieldCount  int              `json:"fieldCount"`
	Edges       []datasetEdgeRow `json:"edges,omitempty"`
}

type listReportDatasetsTool struct {
	access catalogAccess
}

func newListReportDatasetsTool(
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listReportDatasetsTool{access: newCatalogAccess(permissions)}
}

func (t *listReportDatasetsTool) Name() string { return "list_report_datasets" }

func (t *listReportDatasetsTool) Description() string {
	return "List the datasets a report can be built on — shipments, invoices, workers, " +
		"tractors and so on — with the related datasets each one can reach. Call this " +
		"first when you are going to create_report or change a report's dataset, then " +
		"describe_report_dataset for the fields of the one you pick. Only datasets the " +
		"person may read are listed."
}

func (t *listReportDatasetsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Optional text matched against the dataset key, label and description.",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Optional area to narrow to, such as Operations, Billing or Fleet.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *listReportDatasetsTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

func (t *listReportDatasetsTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	query := strings.ToLower(optionalString(params.Params, "query"))
	category := optionalString(params.Params, "category")

	criteria := newSearchCriteria("datasets").at(clockFor(params))
	criteria.text(query)
	criteria.field("category", category)

	catalog := &reportcatalog.Default
	rows := make([]datasetRow, 0, len(catalog.Entities))
	for i := range catalog.Entities {
		entity := &catalog.Entities[i]
		if category != "" && !strings.EqualFold(entity.Category, category) {
			continue
		}
		if query != "" && !matchesDataset(entity, query) {
			continue
		}

		_, readable, err := t.access.entity(ctx, params, entity)
		if err != nil {
			return nil, err
		}
		if !readable {
			continue
		}

		rows = append(rows, datasetRow{
			Dataset:     entity.Key,
			Label:       entity.Label,
			PluralLabel: entity.PluralLabel,
			Description: entity.Description,
			Category:    entity.Category,
			FieldCount:  len(entity.Fields),
			Edges:       edgeRows(entity, false),
		})
	}

	return criteria.result(rows, len(rows)), nil
}

func matchesDataset(entity *reportcatalog.Entity, needle string) bool {
	return strings.Contains(strings.ToLower(entity.Key), needle) ||
		strings.Contains(strings.ToLower(entity.Label), needle) ||
		strings.Contains(strings.ToLower(entity.PluralLabel), needle) ||
		strings.Contains(strings.ToLower(entity.Description), needle)
}

func edgeRows(entity *reportcatalog.Entity, withTargetFields bool) []datasetEdgeRow {
	rows := make([]datasetEdgeRow, 0, len(entity.Edges))
	for i := range entity.Edges {
		edge := &entity.Edges[i]
		if !edge.Traversable {
			continue
		}
		row := datasetEdgeRow{
			Name:        edge.Name,
			Label:       edge.Label,
			Target:      edge.Target,
			Cardinality: string(edge.Cardinality),
		}
		if withTargetFields {
			if target, ok := reportcatalog.Default.Entity(edge.Target); ok {
				row.TargetFields = fieldKeys(target)
			}
		}
		rows = append(rows, row)
	}

	return rows
}

type datasetFieldRow struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Description  string   `json:"description,omitempty"`
	Type         string   `json:"type"`
	Format       string   `json:"format,omitempty"`
	Nullable     bool     `json:"nullable,omitempty"`
	EnumValues   []string `json:"enumValues,omitempty"`
	Aggregations []string `json:"aggregations,omitempty"`
	Filterable   bool     `json:"filterable"`
	Groupable    bool     `json:"groupable"`
	Sensitivity  string   `json:"sensitivity"`
	// Accessible is false for a field the person's role may not read. The
	// compiler refuses a report that names one, so a model that knows this
	// up front builds a report that compiles rather than one that is refused.
	Accessible bool `json:"accessible"`
}

type datasetDescription struct {
	Dataset     string            `json:"dataset"`
	Label       string            `json:"label"`
	Description string            `json:"description,omitempty"`
	Category    string            `json:"category,omitempty"`
	Fields      []datasetFieldRow `json:"fields"`
	FieldCount  int               `json:"fieldCount"`
	Edges       []datasetEdgeRow  `json:"edges,omitempty"`
	Note        string            `json:"note"`
}

type describeReportDatasetTool struct {
	access catalogAccess
}

func newDescribeReportDatasetTool(
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &describeReportDatasetTool{access: newCatalogAccess(permissions)}
}

func (t *describeReportDatasetTool) Name() string { return "describe_report_dataset" }

func (t *describeReportDatasetTool) Description() string {
	return "Describe one dataset from list_report_datasets: every field with its type, " +
		"the aggregations it supports, whether it can filter or group, and whether the " +
		"person may read it, plus the edges to related datasets. Use the field keys " +
		"and edge names exactly as given when you write a report definition. Narrow " +
		"with query when you know what you are looking for, such as \"revenue\" or " +
		"\"delivered\"."
}

func (t *describeReportDatasetTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"dataset": map[string]any{
				"type":        "string",
				"description": "The dataset key from list_report_datasets, such as shipment.",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Optional text matched against field keys, labels and descriptions.",
			},
		},
		"required":             []string{"dataset"},
		"additionalProperties": false,
	}
}

func (t *describeReportDatasetTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

func (t *describeReportDatasetTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	key, err := requireString(params.Params, "dataset")
	if err != nil {
		return nil, err
	}

	entity, ok := reportcatalog.Default.Entity(key)
	if !ok {
		return nil, fmt.Errorf(
			"there is no dataset with the key %q; call list_report_datasets for the ones that exist",
			key,
		)
	}

	detail, readable, err := t.access.entity(ctx, params, entity)
	if err != nil {
		return nil, err
	}
	if !readable {
		return nil, fmt.Errorf(
			"there is no dataset with the key %q that you can read; call "+
				"list_report_datasets for the ones you can",
			key,
		)
	}

	query := strings.ToLower(optionalString(params.Params, "query"))
	fields := make([]datasetFieldRow, 0, len(entity.Fields))
	for i := range entity.Fields {
		field := &entity.Fields[i]
		if query != "" && !matchesField(field, query) {
			continue
		}

		sensitivity, accessible := t.access.fieldAccessible(entity, field, detail)
		fields = append(fields, datasetFieldRow{
			Key:          field.Key,
			Label:        field.Label,
			Description:  field.Description,
			Type:         string(field.Type),
			Format:       string(field.Format),
			Nullable:     field.Nullable,
			EnumValues:   enumValueKeys(field),
			Aggregations: aggregationNames(field),
			Filterable:   field.Filterable,
			Groupable:    field.Groupable,
			Sensitivity:  sensitivity.String(),
			Accessible:   accessible,
		})
	}

	return datasetDescription{
		Dataset:     entity.Key,
		Label:       entity.Label,
		Description: entity.Description,
		Category:    entity.Category,
		Fields:      fields,
		FieldCount:  len(entity.Fields),
		Edges:       edgeRows(entity, true),
		Note: "Refer to a field of this dataset as {\"field\": \"<key>\"} and to a field " +
			"of a related dataset as {\"path\": [\"<edge>\"], \"field\": \"<key>\"}, " +
			"using only the keys each edge's targetFields lists; an edge two steps " +
			"away needs describe_report_dataset on the first target. A measure " +
			"column needs an agg the field lists; a dimension column groups the " +
			"rows. A field marked accessible: false cannot be used by this person.",
	}, nil
}

func matchesField(field *reportcatalog.Field, needle string) bool {
	return strings.Contains(strings.ToLower(field.Key), needle) ||
		strings.Contains(strings.ToLower(field.Label), needle) ||
		strings.Contains(strings.ToLower(field.Description), needle)
}

func enumValueKeys(field *reportcatalog.Field) []string {
	if len(field.EnumValues) == 0 {
		return nil
	}

	values := make([]string, 0, len(field.EnumValues))
	for _, value := range field.EnumValues {
		values = append(values, value.Value)
	}

	return values
}

func aggregationNames(field *reportcatalog.Field) []string {
	if len(field.Aggregations) == 0 {
		return nil
	}

	names := make([]string, 0, len(field.Aggregations))
	for _, agg := range field.Aggregations {
		names = append(names, string(agg))
	}

	return names
}

type reportColumnRow struct {
	ID     string `json:"id"`
	Label  string `json:"label,omitempty"`
	Field  string `json:"field"`
	Kind   string `json:"kind"`
	Agg    string `json:"agg,omitempty"`
	Bucket string `json:"bucket,omitempty"`
}

type reportDescription struct {
	ReportKey    string               `json:"reportKey,omitempty"`
	DefinitionID string               `json:"definitionId,omitempty"`
	Name         string               `json:"name"`
	Description  string               `json:"description,omitempty"`
	Category     string               `json:"category,omitempty"`
	Kind         string               `json:"kind"`
	Visibility   string               `json:"visibility,omitempty"`
	Status       string               `json:"status,omitempty"`
	Editable     bool                 `json:"editable"`
	Format       string               `json:"defaultFormat,omitempty"`
	Dataset      string               `json:"dataset"`
	Columns      []reportColumnRow    `json:"columns"`
	Filters      []string             `json:"filters,omitempty"`
	Sort         []string             `json:"sort,omitempty"`
	Limit        int                  `json:"limit,omitempty"`
	Parameters   []reportParameterRow `json:"parameters,omitempty"`
	// Definition is the report exactly as the builder saves it, so a change
	// can be made by editing it and sending it back through update_report or
	// preview_report rather than by rebuilding it from the summary above.
	Definition *report.Definition `json:"definition"`
	Note       string             `json:"note"`
}

type describeReportTool struct {
	reports reportRunner
}

func newDescribeReportTool(reports reportRunner) serviceports.AgentQueryTool {
	return &describeReportTool{reports: reports}
}

func (t *describeReportTool) Name() string { return "describe_report" }

func (t *describeReportTool) Description() string {
	return "Show what a report contains: its dataset, columns, filters, sort, " +
		"parameters and the full definition as the report builder saves it. Call " +
		"this to answer \"what is in this report\", and before update_report or " +
		"preview_report when you are going to adjust one, so you change the " +
		"definition that exists rather than one you imagined. Name a built-in " +
		"report by reportKey or a saved one by definitionId, from list_reports."
}

func (t *describeReportTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"reportKey": map[string]any{
				"type":        "string",
				"description": "The reportKey of a built-in report from list_reports.",
			},
			"definitionId": map[string]any{
				"type":        "string",
				"description": "The definitionId of a saved report from list_reports.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *describeReportTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

func (t *describeReportTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	source, err := resolveReportSource(ctx, t.reports, params)
	if err != nil {
		return nil, err
	}

	return describeReport(source, params.Actor.UserID), nil
}

func describeReport(source reportSource, actor pulid.ID) reportDescription {
	description := reportDescription{
		ReportKey:   source.Key,
		Name:        source.Name,
		Description: source.Description,
		Category:    source.Category,
		Kind:        source.Kind,
		Visibility:  string(source.Visibility),
		Status:      string(source.Status),
		Editable:    source.Editable(actor),
		Format:      string(source.DefaultFormat),
		Definition:  source.Definition,
		Parameters:  parameterRows(source.Definition),
	}
	if !source.DefinitionID.IsNil() {
		description.DefinitionID = source.DefinitionID.String()
	}

	definition := source.Definition
	if definition == nil {
		description.Note = "This report has no definition to show."

		return description
	}

	description.Dataset = definition.Entity
	description.Columns = make([]reportColumnRow, 0, len(definition.Columns))
	for _, column := range definition.Columns {
		description.Columns = append(description.Columns, reportColumnRow{
			ID:     column.ID,
			Label:  column.Label,
			Field:  fieldRefText(column.Ref),
			Kind:   string(column.Kind),
			Agg:    string(column.Agg),
			Bucket: string(column.Bucket),
		})
	}
	description.Filters = filterGroupText(definition.Filters, 0)
	description.Limit = definition.Limit
	for _, spec := range definition.Sort {
		description.Sort = append(
			description.Sort,
			fmt.Sprintf("%s %s", spec.ColumnID, spec.Direction),
		)
	}

	switch {
	case description.Editable:
		description.Note = "You may change this report with update_report: send the " +
			"definition back with your edits, after checking them with " +
			"preview_report. Describe the columns and filters to the person in " +
			"plain words rather than pasting the definition."
	case source.DefinitionID.IsNil():
		description.Note = "This is a built-in report and cannot be edited in place. To " +
			"adjust it, fork_report makes the person their own copy to change, or " +
			"create_report builds a new one from the same definition. Describe the " +
			"columns and filters in plain words rather than pasting the definition."
	default:
		description.Note = "This report belongs to someone else, so only they can " +
			"change it. create_report can build the person their own version from " +
			"this definition. Describe the columns and filters in plain words " +
			"rather than pasting the definition."
	}

	return description
}

func fieldRefText(ref report.FieldRef) string {
	if len(ref.Path) == 0 {
		return ref.Field
	}

	return strings.Join(ref.Path, ".") + "." + ref.Field
}

// filterGroupText reads a filter tree back as sentences, one per leaf, with
// the group's operator carried on nested lines. The model narrates these to
// the person; the definition itself is what it edits.
func filterGroupText(group *report.FilterGroup, depth int) []string {
	if group == nil || depth > 8 {
		return nil
	}

	lines := make([]string, 0, len(group.Filters)+len(group.Groups))
	for _, filter := range group.Filters {
		lines = append(lines, filterLeafText(filter))
	}
	for i := range group.Groups {
		nested := filterGroupText(&group.Groups[i], depth+1)
		if len(nested) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf(
			"(%s)", strings.Join(nested, " "+strings.ToUpper(string(group.Groups[i].Op))+" "),
		))
	}

	if len(lines) > 1 && depth == 0 {
		return []string{strings.Join(lines, " "+strings.ToUpper(string(group.Op))+" ")}
	}

	return lines
}

func filterLeafText(filter report.FieldFilter) string {
	target := fieldRefText(filter.Ref)
	if filter.Agg != "" {
		target = fmt.Sprintf("%s(%s)", filter.Agg, target)
	}

	switch {
	case filter.Param != "":
		return fmt.Sprintf("%s %s :%s", target, filter.Operator, filter.Param)
	case filter.Value == nil:
		return fmt.Sprintf("%s %s", target, filter.Operator)
	default:
		return fmt.Sprintf("%s %s %v", target, filter.Operator, filter.Value)
	}
}

type previewColumnRow struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Type   string `json:"type"`
	Format string `json:"format,omitempty"`
}

type reportPreview struct {
	Name     string             `json:"name,omitempty"`
	Dataset  string             `json:"dataset"`
	Columns  []previewColumnRow `json:"columns"`
	RowCount int                `json:"rowCount"`
	// Rows are keyed by column label so the model reads them as a table
	// rather than as positional tuples it has to line up with the columns.
	Rows      []map[string]any `json:"rows"`
	Totals    map[string]any   `json:"totals,omitempty"`
	Truncated bool             `json:"truncated"`
	Note      string           `json:"note"`
}

type previewReportTool struct {
	reports reportRunner
}

func newPreviewReportTool(reports reportRunner) serviceports.AgentQueryTool {
	return &previewReportTool{reports: reports}
}

func (t *previewReportTool) Name() string { return "preview_report" }

func (t *previewReportTool) Description() string {
	return "Run a report definition now and see its first rows, without saving it or " +
		"producing a file. Use it to check a definition before create_report or " +
		"update_report, to answer a quick question a small custom report can answer, " +
		"or to look at what a saved or built-in report produces. Give a definition " +
		"to preview (the shape describe_report returns and create_report takes), or " +
		"name an existing report by definitionId or reportKey. The compiler checks " +
		"every field against the person's access and says what is wrong, in " +
		"which case fix the definition and try again rather than reporting the " +
		"error to the person. At most " + fmt.Sprint(maxPreviewRows) + " rows come " +
		"back; for the full result, run_report."
}

func (t *previewReportTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"definition": report.DefinitionJSONSchema(),
			"definitionId": map[string]any{
				"type":        "string",
				"description": "Preview a saved report from list_reports instead of a definition.",
			},
			"reportKey": map[string]any{
				"type":        "string",
				"description": "Preview a built-in report from list_reports instead of a definition.",
			},
			"parameters": map[string]any{
				"type": "object",
				"description": "Values for the report's parameters, keyed by name, in the " +
					"shape each declares. A list parameter takes a plain JSON array.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *previewReportTool) PermissionResource() permission.Resource {
	return permission.ResourceReport
}

func (t *previewReportTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	source, err := t.source(ctx, params)
	if err != nil {
		return nil, err
	}

	values := normalizeReportParameters(
		source.Definition,
		optionalObject(params.Params, "parameters"),
	)
	if err = requireReportParameters(source.Name, source.Definition, values); err != nil {
		return nil, err
	}

	result, err := t.reports.Preview(ctx, &reporting.PreviewRequest{
		Request:    reportingRequestFor(params),
		Definition: source.Definition,
		Params:     values,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"the report could not be previewed: %s",
			compileErrorHint(&reportcatalog.Default, err.Error()),
		)
	}

	return previewOf(source, result), nil
}

// source reads what to preview: an inline definition first, because that is
// the case the tool exists for, and a named report otherwise.
func (t *previewReportTool) source(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (reportSource, error) {
	raw, given := params.Params["definition"]
	if given && raw != nil {
		definition, err := report.DecodeDefinition(raw)
		if err != nil {
			return reportSource{}, err
		}

		return reportSource{Name: "the definition", Definition: definition}, nil
	}

	source, err := resolveReportSource(ctx, t.reports, params)
	if err != nil {
		if errors.Is(err, errNoReportNamed) {
			return reportSource{}, errors.New(
				"give a definition to preview, or name a report by definitionId or " +
					"reportKey from list_reports",
			)
		}

		return reportSource{}, err
	}

	return source, nil
}

func previewOf(source reportSource, result *reporting.PreviewResult) reportPreview {
	preview := reportPreview{
		Dataset:   source.Definition.Entity,
		Columns:   make([]previewColumnRow, 0, len(result.Columns)),
		RowCount:  len(result.Rows),
		Truncated: result.Truncated,
	}
	if !source.DefinitionID.IsNil() || source.Key != "" {
		preview.Name = source.Name
	}

	labels := make([]string, 0, len(result.Columns))
	for _, column := range result.Columns {
		label := column.Label
		if label == "" {
			label = column.ID
		}
		labels = append(labels, label)
		preview.Columns = append(preview.Columns, previewColumnRow{
			ID:     column.ID,
			Label:  label,
			Type:   string(column.Type),
			Format: string(column.Format),
		})
	}

	shown := len(result.Rows)
	if shown > maxPreviewRows {
		shown = maxPreviewRows
	}
	preview.Rows = make([]map[string]any, 0, shown)
	for _, row := range result.Rows[:shown] {
		preview.Rows = append(preview.Rows, rowByLabel(labels, row))
	}
	if len(result.Totals) > 0 {
		preview.Totals = rowByLabel(labels, result.Totals)
	}

	switch {
	case len(result.Rows) == 0:
		preview.Note = "The definition compiled and returned no rows. The filters may be " +
			"too narrow, or nothing matches; say which filters were applied rather " +
			"than that there is no data."
	case len(result.Rows) > shown:
		preview.Note = fmt.Sprintf(
			"The definition compiled. %d rows came back and the first %d are shown; "+
				"for the whole result save it with create_report and run_report it, "+
				"or run_report an existing report.",
			len(result.Rows), shown,
		)
	default:
		preview.Note = fmt.Sprintf(
			"The definition compiled and produced %d rows, all shown.", len(result.Rows),
		)
	}
	if result.Truncated {
		preview.Note += " The preview hit its row cap, so the count is a floor, not the total."
	}

	return preview
}

func rowByLabel(labels []string, row serviceports.ReportRow) map[string]any {
	out := make(map[string]any, len(labels))
	for i, label := range labels {
		if i < len(row) {
			out[label] = row[i]
		}
	}

	return out
}
