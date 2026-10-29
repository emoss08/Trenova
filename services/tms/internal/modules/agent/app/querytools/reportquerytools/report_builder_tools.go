package reportquerytools

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/keel/analytics/table"
	"github.com/emoss08/keel/analytics/value"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const maxPreviewRows = 20

type reportSource struct {
	Name          string
	Key           string
	DefinitionID  pulid.ID
	Definition    *ir.Definition
	DefaultFormat report.Format
	Description   string
	Category      string
	Kind          string
	Visibility    report.Visibility
	Status        report.DefinitionStatus
	OwnerID       pulid.ID
}

func (s reportSource) Editable(actor pulid.ID) bool {
	return !s.DefinitionID.IsNil() && s.OwnerID == actor
}

var errNoReportNamed = errors.New(
	"name the report: reportKey for a built-in report or definitionId for a saved one, " +
		"both from report_list",
)

func resolveReportSource(
	ctx context.Context,
	reports Runner,
	params *agenttoolports.QueryToolParams,
) (reportSource, error) {
	if raw := querytool.OptionalString(params.Params, "definitionId"); raw != "" {
		id, err := pulid.Parse(raw)
		if err != nil {
			return reportSource{}, fmt.Errorf(
				"parameter \"definitionId\" is not a valid id: %w",
				err,
			)
		}

		definition, err := reports.GetDefinition(ctx, &reporting.GetDefinitionRequest{
			Request:      RequestFor(params),
			DefinitionID: id,
		})
		if err != nil {
			if errortypes.IsNotFoundError(err) {
				return reportSource{}, fmt.Errorf(
					"there is no saved report with the id %q that you can see; call "+
						"report_list for the ones that exist",
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
			"there is no report with the key %q; call report_list for the keys that exist",
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

type actorCatalog struct {
	*reporting.Catalog

	byKey map[string]*reporting.CatalogEntity
}

func catalogFor(
	ctx context.Context,
	reports Runner,
	params *agenttoolports.QueryToolParams,
) (*actorCatalog, error) {
	request := RequestFor(params)
	catalog, err := reports.Catalog(ctx, &request)
	if err != nil {
		return nil, err
	}
	out := &actorCatalog{
		Catalog: catalog,
		byKey:   make(map[string]*reporting.CatalogEntity, len(catalog.Entities)),
	}
	for i := range catalog.Entities {
		out.byKey[catalog.Entities[i].View.Key] = &catalog.Entities[i]
	}
	return out, nil
}

func (c *actorCatalog) entity(key string) (*reporting.CatalogEntity, bool) {
	entity, ok := c.byKey[key]
	return entity, ok
}

func (c *actorCatalog) need(use reportaccess.Use) string {
	if c.Kinds == nil {
		return ""
	}
	return c.Kinds.Need(use)
}

type datasetEdgeRow struct {
	Name          string   `json:"name"`
	Label         string   `json:"label,omitempty"`
	Target        string   `json:"target"`
	Cardinality   string   `json:"cardinality"`
	Key           string   `json:"key,omitempty"`
	TargetFields  []string `json:"targetFields,omitempty"`
	ReferenceOnly bool     `json:"referenceOnly,omitempty"`
	Needs         string   `json:"needs,omitempty"`
}

type datasetRow struct {
	Dataset     string `json:"dataset"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	FieldCount  int    `json:"fieldCount"`
}

const (
	defaultDatasetPage = 20
	maxDatasetPage     = 90
	defaultFieldPage   = 40
	maxFieldPage       = 80
	maxEnumValuesShown = 12
)

type listReportDatasetsTool struct {
	reports Runner
}

func NewListReportDatasetsTool(reports Runner) agenttoolports.AgentQueryTool {
	return &listReportDatasetsTool{reports: reports}
}

func (t *listReportDatasetsTool) Name() string { return "report_list_datasets" }

func (t *listReportDatasetsTool) Description() string {
	return "List the datasets a new report can be built on, such as shipment, invoice or " +
		"worker. Use it before report_create to pick the dataset, searching with query " +
		"(\"shipment\", \"invoice\") rather than paging through all of them, then call " +
		"report_get_dataset for its fields. It lists kinds of data, not records: to " +
		"find a customer or a shipment use customer_list or shipment_list."
}

func (t *listReportDatasetsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": querytool.WithPaging(map[string]any{
			"query": map[string]any{
				"type": "string",
				"description": "Words matched against the dataset key, label and description; " +
					"each word must begin a word there.",
			},
			"category": map[string]any{
				"type":        "string",
				"description": "Optional area to narrow to, such as Operations, Billing or Fleet.",
			},
		}, defaultDatasetPage, maxDatasetPage),
		"additionalProperties": false,
	}
}

func (t *listReportDatasetsTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource: permission.ResourceReport,
	})
}

func (t *listReportDatasetsTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	query := strings.TrimSpace(querytool.OptionalString(params.Params, "query"))
	category := querytool.OptionalString(params.Params, "category")

	criteria := filtercatalog.NewCriteria("datasets").At(querytool.ClockFor(params))
	criteria.Text(query)
	criteria.Field("category", category)

	catalog, err := catalogFor(ctx, t.reports, params)
	if err != nil {
		return nil, err
	}

	words := stringutils.SearchWords(query)
	rows := make([]datasetRow, 0, len(catalog.Entities))
	for i := range catalog.Entities {
		entity := &catalog.Entities[i]
		view := &entity.View
		if !entity.Readable {
			continue
		}
		if category != "" && !strings.EqualFold(view.Category, category) {
			continue
		}
		if !matchesDataset(view, words) {
			continue
		}

		rows = append(rows, datasetRow{
			Dataset:     view.Key,
			Label:       view.Label,
			Description: stringutils.FirstSentence(view.Doc),
			Category:    view.Category,
			FieldCount:  len(view.Fields),
		})
	}

	window := querytool.ReadPage(params.Params, defaultDatasetPage, maxDatasetPage)
	shown, more := querytool.SlicePage(window, rows)
	if shown == nil {
		shown = []datasetRow{}
	}

	outcome := querytool.SearchResult(criteria, shown, len(shown)).Paged(window, more)
	if len(rows) == 0 && query != "" {
		outcome.Note = fmt.Sprintf(
			"No dataset matched %s. Every word of query has to begin a word of a dataset's "+
				"key, label or description, so use fewer or broader words (\"shipment\" "+
				"rather than \"shipment delivery\"), drop category, or call this with no query "+
				"to list every dataset before concluding the data is not reportable.",
			strings.Join(criteria.Terms(), " and "),
		)
	}

	return outcome, nil
}

func matchesDataset(view *analytics.EntityView, words []string) bool {
	return stringutils.MatchesWordPrefixes(
		words,
		stringutils.SearchableKey(view.Key),
		view.Label,
		view.Plural,
		view.Doc,
	)
}

func edgeCardinality(edge *analytics.EdgeView) string {
	switch {
	case edge.Pick:
		return "pick"
	case edge.ToMany:
		return "many"
	default:
		return "one"
	}
}

func edgeRows(catalog *actorCatalog, decided *reporting.CatalogEntity) []datasetEdgeRow {
	view := &decided.View
	rows := make([]datasetEdgeRow, 0, len(view.Edges))
	for i := range view.Edges {
		edge := &view.Edges[i]
		use := decided.Edges[i]
		reached, ok := catalog.entity(edge.Target)
		if !ok {
			continue
		}
		if use.Available() && !reached.Usable() {
			continue
		}

		row := datasetEdgeRow{
			Name:        edge.Name,
			Label:       edge.Label,
			Target:      edge.Target,
			Cardinality: edgeCardinality(edge),
			Needs:       catalog.need(use),
		}
		if !edge.ToMany && !edge.Pick && hasUsableField(reached, "id") {
			row.Key = edge.Name + ".id"
		}
		if use.Available() {
			row.TargetFields = usableFieldKeys(reached)
			row.ReferenceOnly = !reached.Readable
		}
		rows = append(rows, row)
	}

	return rows
}

func hasUsableField(decided *reporting.CatalogEntity, key string) bool {
	for i := range decided.View.Fields {
		if decided.View.Fields[i].Key == key {
			return decided.Fields[i].Available()
		}
	}
	return false
}

func listedFieldCount(decided *reporting.CatalogEntity) int {
	if decided.Readable {
		return len(decided.View.Fields)
	}
	count := 0
	for i := range decided.View.Fields {
		if decided.Fields[i].Decision != reportaccess.NeedsKindRead {
			count++
		}
	}
	return count
}

func usableFieldKeys(decided *reporting.CatalogEntity) []string {
	keys := make([]string, 0, len(decided.Fields))
	for i := range decided.View.Fields {
		if decided.Fields[i].Available() {
			keys = append(keys, decided.View.Fields[i].Key)
		}
	}
	return keys
}

type datasetFieldRow struct {
	Key          string   `json:"key"`
	Label        string   `json:"label,omitempty"`
	Description  string   `json:"description,omitempty"`
	Type         string   `json:"type"`
	Measure      bool     `json:"measure,omitempty"`
	EnumValues   []string `json:"enumValues,omitempty"`
	Aggregations []string `json:"aggs,omitempty"`
	NoFilter     bool     `json:"noFilter,omitempty"`
	NoGroup      bool     `json:"noGroup,omitempty"`
	Closed       bool     `json:"closed,omitempty"`
	Needs        string   `json:"needs,omitempty"`
}

type datasetDescription struct {
	Dataset       string            `json:"dataset"`
	Label         string            `json:"label"`
	Description   string            `json:"description,omitempty"`
	Category      string            `json:"category,omitempty"`
	Fields        []datasetFieldRow `json:"fields"`
	FieldCount    int               `json:"fieldCount"`
	Shown         int               `json:"shownFieldCount"`
	NextOffset    *int              `json:"nextOffset,omitempty"`
	Edges         []datasetEdgeRow  `json:"edges,omitempty"`
	RelatedFields []string          `json:"relatedFields,omitempty"`
	Note          string            `json:"note"`
}

type describeReportDatasetTool struct {
	reports Runner
}

func NewDescribeReportDatasetTool(reports Runner) agenttoolports.AgentQueryTool {
	return &describeReportDatasetTool{reports: reports}
}

func (t *describeReportDatasetTool) Name() string { return "report_get_dataset" }

func (t *describeReportDatasetTool) Description() string {
	return "Describe one report dataset's fields, and the related datasets it can reach. " +
		"Use it after report_list_datasets and before report_preview or report_create, and " +
		"write field keys and edge names exactly as given. Narrow with query (\"revenue\", " +
		"\"delivered\") when you know what you want; a wide dataset pages with offset."
}

func (t *describeReportDatasetTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": querytool.WithPaging(map[string]any{
			"dataset": map[string]any{
				"type":        "string",
				"description": "The dataset key from report_list_datasets, such as shipment.",
			},
			"query": map[string]any{
				"type": "string",
				"description": "Optional words matched against field keys, labels and " +
					"descriptions, here and on the datasets the edges reach.",
			},
		}, defaultFieldPage, maxFieldPage),
		"required":             []string{"dataset"},
		"additionalProperties": false,
	}
}

func (t *describeReportDatasetTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource: permission.ResourceReport,
	})
}

const datasetNote = "Refer to a field by its dotted path: \"<key>\" for a field of this " +
	"dataset and \"<edge>.<key>\" for a field of a related dataset, using only the keys " +
	"each edge's targetFields lists; an edge two steps away needs report_get_dataset on " +
	"the first target. A measure column needs an agg the field lists; a dimension column " +
	"groups the rows, and a report of dimension columns alone lists rows. When you " +
	"already know a related record's id, filter on the path the edge's key names with " +
	"that id — customer.id eq \"cus_…\" — not on the record's name, which two records " +
	"can share and a rename changes. A field marked closed cannot be used by this " +
	"person; its needs says what it lacks. An edge marked referenceOnly leads to a " +
	"dataset this person cannot read, and only its targetFields can be used through it."

func (t *describeReportDatasetTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	key, err := querytool.RequireString(params.Params, "dataset")
	if err != nil {
		return nil, err
	}

	catalog, err := catalogFor(ctx, t.reports, params)
	if err != nil {
		return nil, err
	}
	decided, ok := catalog.entity(key)
	if !ok {
		return nil, fmt.Errorf(
			"there is no dataset with the key %q; call report_list_datasets for the ones that exist",
			key,
		)
	}
	view := &decided.View
	if !decided.Usable() {
		return nil, fmt.Errorf(
			"there is no dataset with the key %q that you can use: it needs %s read, "+
				"which your role does not grant; call report_list_datasets for the ones you can",
			key, decided.Resource,
		)
	}

	query := strings.TrimSpace(querytool.OptionalString(params.Params, "query"))
	words := stringutils.SearchWords(query)
	fields := make([]datasetFieldRow, 0, len(view.Fields))
	for i := range view.Fields {
		field := &view.Fields[i]
		if !matchesField(field, words) {
			continue
		}

		use := decided.Fields[i]
		if use.Decision == reportaccess.NeedsKindRead {
			continue
		}
		fields = append(fields, datasetFieldRow{
			Key:          field.Key,
			Label:        labelIfNotKey(field),
			Description:  stringutils.FirstSentence(field.Doc),
			Type:         string(field.Type),
			Measure:      field.Measure,
			EnumValues:   shortEnum(field),
			Aggregations: aggregationNames(field),
			NoFilter:     !field.Filterable,
			NoGroup:      !field.Groupable,
			Closed:       !use.Available(),
			Needs:        catalog.need(use),
		})
	}

	matched := len(fields)
	window := querytool.ReadPage(params.Params, defaultFieldPage, maxFieldPage)
	fields, more := querytool.SlicePage(window, fields)
	if fields == nil {
		fields = []datasetFieldRow{}
	}

	description := datasetDescription{
		Dataset:     view.Key,
		Label:       view.Label,
		Description: view.Doc,
		Category:    view.Category,
		Fields:      fields,
		FieldCount:  listedFieldCount(decided),
		Shown:       len(fields),
		Edges:       edgeRows(catalog, decided),
		Note:        datasetNote,
	}
	if !decided.Readable {
		description.Note = fmt.Sprintf(
			"You cannot read %s (it needs %s read), so a report cannot start from this "+
				"dataset. A report on a dataset you can read reaches it through an edge "+
				"marked referenceOnly and can use only the fields listed here. ",
			view.Plural, decided.Resource,
		) + description.Note
	}
	if more {
		next := window.Offset + window.Limit
		description.NextOffset = &next
		description.Note += fmt.Sprintf(
			" Showing %d of %d matching fields; call this again with offset %d, or with "+
				"query, for the rest rather than assuming they do not exist.",
			len(fields), matched, next,
		)
	}
	if len(words) > 0 {
		description.RelatedFields = relatedFieldMatches(catalog, decided, words)
		description.Note += queryNote(view.Key, query, matched, description.RelatedFields)
	}

	return description, nil
}

const maxRelatedFieldMatches = 20

func relatedFieldMatches(
	catalog *actorCatalog,
	decided *reporting.CatalogEntity,
	words []string,
) []string {
	matches := make([]string, 0, 4)
	for i := range decided.View.Edges {
		edge := &decided.View.Edges[i]
		if !decided.Edges[i].Available() {
			continue
		}
		target, ok := catalog.entity(edge.Target)
		if !ok {
			continue
		}
		for j := range target.View.Fields {
			field := &target.View.Fields[j]
			if !target.Fields[j].Available() || !matchesField(field, words) {
				continue
			}
			matches = append(matches, edge.Name+"."+field.Key)
			if len(matches) == maxRelatedFieldMatches {
				return matches
			}
		}
	}

	return matches
}

func queryNote(dataset, query string, matched int, related []string) string {
	switch {
	case matched > 0 && len(related) > 0:
		return " relatedFields are fields of related datasets the query also matches, " +
			"written as the dotted path a definition uses."
	case matched > 0:
		return ""
	case len(related) > 0:
		edge, field, _ := strings.Cut(related[0], ".")
		return fmt.Sprintf(
			" No field of %s itself matches %q, but fields of related datasets do: "+
				"relatedFields lists them as dotted paths, so %s is the field %s on the edge "+
				"%s, and a definition writes it exactly so.",
			dataset, query, related[0], field, edge,
		)
	default:
		return fmt.Sprintf(
			" No field of %s, or of the datasets its edges reach, matches %q. Every word "+
				"of query has to begin a word of a field's key, label or description, so "+
				"use fewer or broader words, or call this with no query to page through "+
				"every field, before concluding the dataset cannot report it.",
			dataset, query,
		)
	}
}

func labelIfNotKey(field *analytics.FieldView) string {
	squashed := strings.ReplaceAll(strings.ToLower(field.Label), " ", "")
	if squashed == strings.ToLower(field.Key) {
		return ""
	}

	return field.Label
}

func shortEnum(field *analytics.FieldView) []string {
	if len(field.Enum) == 0 || len(field.Enum) > maxEnumValuesShown {
		return nil
	}

	values := make([]string, 0, len(field.Enum))
	for _, value := range field.Enum {
		values = append(values, value.Value)
	}

	return values
}

func matchesField(field *analytics.FieldView, words []string) bool {
	return stringutils.MatchesWordPrefixes(
		words,
		stringutils.SearchableKey(field.Key),
		field.Label,
		field.Doc,
	)
}

func aggregationNames(field *analytics.FieldView) []string {
	if len(field.Aggs) == 0 {
		return nil
	}

	names := make([]string, 0, len(field.Aggs))
	for _, agg := range field.Aggs {
		names = append(names, string(agg))
	}

	return names
}

type reportColumnRow struct {
	ID     string `json:"id"`
	Label  string `json:"label,omitempty"`
	Field  string `json:"field,omitempty"`
	Kind   string `json:"kind"`
	Agg    string `json:"agg,omitempty"`
	Bucket string `json:"bucket,omitempty"`
	Calc   string `json:"calc,omitempty"`
	Hidden bool   `json:"hidden,omitempty"`
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
	Having       []string             `json:"having,omitempty"`
	Sort         []string             `json:"sort,omitempty"`
	Limit        int                  `json:"limit,omitempty"`
	Parameters   []reportParameterRow `json:"parameters,omitempty"`
	Definition   *ir.Definition       `json:"definition"`
	Note         string               `json:"note"`
}

type describeReportTool struct {
	reports Runner
}

func NewDescribeReportTool(reports Runner) agenttoolports.AgentQueryTool {
	return &describeReportTool{reports: reports}
}

func (t *describeReportTool) Name() string { return querytool.ToolReportGet }

func (t *describeReportTool) Description() string {
	return "Show what a report contains: its dataset, columns, filters, sort, " +
		"parameters and the full definition as the report builder saves it. Call " +
		"this to answer \"what is in this report\", and before report_update or " +
		"report_preview when you are going to adjust one, so you change the " +
		"definition that exists rather than one you imagined. Name a built-in " +
		"report by reportKey or a saved one by definitionId, from report_list."
}

func (t *describeReportTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"reportKey": map[string]any{
				"type":        "string",
				"description": "The reportKey of a built-in report from report_list.",
			},
			"definitionId": agenttoolschema.OfResource(map[string]any{
				"type":        "string",
				"description": "The definitionId of a saved report from report_list.",
			}, permission.ResourceReport),
		},
		"additionalProperties": false,
	}
}

func (t *describeReportTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource: permission.ResourceReport,
	})
}

func (t *describeReportTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
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
	for i := range definition.Columns {
		description.Columns = append(
			description.Columns,
			definitionColumnRow(&definition.Columns[i]),
		)
	}
	description.Filters = definitionFilterText(definition.Filters, 0)
	description.Having = definitionFilterText(definition.Having, 0)
	description.Limit = definition.Limit.Value
	for _, spec := range definition.Sort {
		direction := "asc"
		if spec.Desc {
			direction = "desc"
		}
		description.Sort = append(description.Sort, spec.Column+" "+direction)
	}

	switch {
	case description.Editable:
		description.Note = "You may change this report with report_update: send the " +
			"definition back with your edits, after checking them with " +
			"report_preview. Describe the columns and filters to the person in " +
			"plain words rather than pasting the definition."
	case source.DefinitionID.IsNil():
		description.Note = "This is a built-in report and cannot be edited in place. To " +
			"adjust it, report_fork makes the person their own copy to change, or " +
			"report_create builds a new one from the same definition. Describe the " +
			"columns and filters in plain words rather than pasting the definition."
	default:
		description.Note = "This report belongs to someone else, so only they can " +
			"change it. report_create can build the person their own version from " +
			"this definition. Describe the columns and filters in plain words " +
			"rather than pasting the definition."
	}

	return description
}

func definitionColumnRow(column *ir.Column) reportColumnRow {
	row := reportColumnRow{ID: column.ID, Label: column.Label, Hidden: column.Hidden}
	switch {
	case column.Dimension != nil:
		row.Kind = "dimension"
		row.Field = column.Dimension.Field
		if column.Dimension.Bucket.Unit != "" {
			row.Bucket = string(column.Dimension.Bucket.Unit)
		} else if column.Dimension.Bucket.Param != "" {
			row.Bucket = "@" + column.Dimension.Bucket.Param
		}
	case column.Measure != nil:
		row.Kind = "measure"
		row.Field = column.Measure.Field
		row.Agg = string(column.Measure.Agg)
	case column.Calc != nil:
		row.Kind = "calc"
		row.Calc = column.Calc.String()
	case column.Window != nil:
		row.Kind = "window"
		row.Field = column.Window.Of
		row.Agg = string(column.Window.Fn)
	}
	return row
}

func definitionFilterText(filter *ir.Filter, depth int) []string {
	if filter == nil || depth > 8 {
		return nil
	}

	switch {
	case filter.And != nil, filter.Or != nil:
		children, op := filter.And, "AND"
		if filter.Or != nil {
			children, op = filter.Or, "OR"
		}
		lines := make([]string, 0, len(children))
		for i := range children {
			nested := definitionFilterText(&children[i], depth+1)
			if len(nested) == 0 {
				continue
			}
			line := strings.Join(nested, " ")
			if children[i].And != nil || children[i].Or != nil {
				line = "(" + line + ")"
			}
			lines = append(lines, line)
		}
		if len(lines) > 1 {
			return []string{strings.Join(lines, " "+op+" ")}
		}
		return lines
	default:
		return []string{filterLeafText(filter)}
	}
}

func filterLeafText(filter *ir.Filter) string {
	target := filter.Field
	switch {
	case filter.Column != "":
		target = "[" + filter.Column + "]"
	case filter.Calc != nil:
		target = filter.Calc.String()
	}

	switch {
	case filter.Param != "":
		return fmt.Sprintf("%s %s @%s", target, filter.Op, filter.Param)
	case filter.Value == nil:
		return fmt.Sprintf("%s %s", target, filter.Op)
	default:
		return fmt.Sprintf("%s %s %v", target, filter.Op, filter.Value)
	}
}

type previewColumnRow struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Type   string `json:"type"`
	Format string `json:"format,omitempty"`
}

type reportPreview struct {
	Name      string             `json:"name,omitempty"`
	Dataset   string             `json:"dataset"`
	Columns   []previewColumnRow `json:"columns"`
	RowCount  int                `json:"rowCount"`
	Rows      []map[string]any   `json:"rows"`
	Totals    map[string]any     `json:"totals,omitempty"`
	Truncated bool               `json:"truncated"`
	Note      string             `json:"note"`
}

type previewReportTool struct {
	reports Runner
}

func NewPreviewReportTool(reports Runner) agenttoolports.AgentQueryTool {
	return &previewReportTool{reports: reports}
}

func (t *previewReportTool) Name() string { return "report_preview" }

func (t *previewReportTool) Description() string {
	return "Run a report definition now and see its first rows, without saving it or " +
		"producing a file. Use it to check a definition before report_create or " +
		"report_update, to answer a quick question a small custom report can answer, " +
		"or to look at what a saved or built-in report produces. Give a definition " +
		"to preview (the shape report_get returns and report_create takes), or " +
		"name an existing report by definitionId or reportKey. The engine checks " +
		"every field against the person's access and says what is wrong, naming the " +
		"part of the definition at fault and the closest names it knows, in which case " +
		"fix the definition and try again rather than reporting the error to the " +
		"person. At most " + strconv.Itoa(maxPreviewRows) + " rows come back; for the " +
		"full result, report_run."
}

func (t *previewReportTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"definition": report.DefinitionToolSchema(),
			"definitionId": agenttoolschema.OfResource(map[string]any{
				"type":        "string",
				"description": "Preview a saved report from report_list instead of a definition.",
			}, permission.ResourceReport),
			"reportKey": map[string]any{
				"type":        "string",
				"description": "Preview a built-in report from report_list instead of a definition.",
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

func (t *previewReportTool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource: permission.ResourceReport,
	})
}

func (t *previewReportTool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	source, err := t.source(ctx, params)
	if err != nil {
		return nil, err
	}

	values := NormalizeParameters(
		source.Definition,
		querytool.OptionalObject(params.Params, "parameters"),
	)
	if err = RequireParameters(source.Name, source.Definition, values); err != nil {
		return nil, err
	}

	result, err := t.reports.Preview(ctx, &reporting.PreviewRequest{
		Request:    RequestFor(params),
		Definition: source.Definition,
		Params:     values,
	})
	if err != nil {
		return nil, fmt.Errorf("the report could not be previewed: %w", err)
	}

	return previewOf(source, result, querytool.ClockFor(params)), nil
}

func (t *previewReportTool) source(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (reportSource, error) {
	raw, given := params.Params["definition"]
	if given && raw != nil {
		definition, err := reporting.DecodeDefinition(raw)
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
					"reportKey from report_list",
			)
		}

		return reportSource{}, err
	}

	return source, nil
}

func previewOf(
	source reportSource,
	result *reporting.PreviewResult,
	clk filtercatalog.Clock,
) reportPreview {
	preview := reportPreview{
		Dataset:   source.Definition.Entity,
		Columns:   make([]previewColumnRow, 0, len(result.Columns)),
		RowCount:  len(result.Rows),
		Truncated: result.Truncated,
	}
	if !source.DefinitionID.IsNil() || source.Key != "" {
		preview.Name = source.Name
	}

	shape := newSampleShape(len(result.Columns), clk)
	for i := range result.Columns {
		preview.Columns = append(preview.Columns, shape.add(&result.Columns[i]))
	}

	preview.Rows = sampleRows(shape, result.Rows, maxPreviewRows)
	shown := len(preview.Rows)
	if len(result.Totals) > 0 {
		preview.Totals = shape.row(result.Totals)
	}

	switch {
	case len(result.Rows) == 0:
		preview.Note = "The definition compiled and returned no rows. The filters may be " +
			"too narrow, or nothing matches; say which filters were applied rather " +
			"than that there is no data."
	case len(result.Rows) > shown:
		preview.Note = fmt.Sprintf(
			"The definition compiled. %d rows came back and the first %d are shown; "+
				"for the whole result save it with report_create and report_run it, "+
				"or report_run an existing report.",
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
	if hasDated(shape.dated) {
		preview.Note += " Date columns are shown here as dates in the organization's timezone, " +
			"and the download renders them the same way; the stored value is epoch seconds."
	}

	return preview
}

type sampleShape struct {
	labels []string
	dated  []bool
	clk    filtercatalog.Clock
}

func newSampleShape(width int, clk filtercatalog.Clock) sampleShape {
	return sampleShape{
		labels: make([]string, 0, width),
		dated:  make([]bool, 0, width),
		clk:    clk,
	}
}

func (s *sampleShape) add(column *table.Column) previewColumnRow {
	label := column.Label
	if label == "" {
		label = column.ID
	}
	s.labels = append(s.labels, label)
	s.dated = append(s.dated, column.Type == value.Epoch)

	return previewColumnRow{
		ID:     column.ID,
		Label:  label,
		Type:   column.Type.String(),
		Format: string(column.Display.Style),
	}
}

func (s *sampleShape) row(row []any) map[string]any {
	return rowByLabel(s.labels, renderDates(row, s.dated, s.clk))
}

func sampleRows(shape sampleShape, rows [][]any, limit int) []map[string]any {
	shown := min(len(rows), limit)
	out := make([]map[string]any, 0, shown)
	for _, row := range rows[:shown] {
		out = append(out, shape.row(row))
	}

	return out
}

func hasDated(dated []bool) bool {
	for _, is := range dated {
		if is {
			return true
		}
	}

	return false
}

func renderDates(row []any, dated []bool, clk filtercatalog.Clock) []any {
	out := make([]any, len(row))
	for idx, cell := range row {
		if idx < len(dated) && dated[idx] {
			if seconds, ok := epochSeconds(cell); ok {
				out[idx] = time.Unix(seconds, 0).In(clk.Location()).Format("2006-01-02 15:04 MST")

				continue
			}
		}
		out[idx] = cell
	}

	return out
}

func epochSeconds(cell any) (int64, bool) {
	switch typed := cell.(type) {
	case int64:
		return typed, typed > 0
	case int:
		return int64(typed), typed > 0
	case float64:
		return int64(typed), typed > 0
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)

		return parsed, err == nil && parsed > 0
	default:
		return 0, false
	}
}

func rowByLabel(labels []string, row []any) map[string]any {
	out := make(map[string]any, len(labels))
	for i, label := range labels {
		if i < len(row) {
			out[label] = row[i]
		}
	}

	return out
}
