package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"github.com/emoss08/trenova/shared/stringutils"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
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
			return reportSource{}, fmt.Errorf(
				"parameter \"definitionId\" is not a valid id: %w",
				err,
			)
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
		return nil, false, errors.New(
			"datasets cannot be described because authorization is unavailable",
		)
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
	// Key is the field on this dataset holding the id of the record the edge
	// leads to, such as customerId for the customer edge. A model that had
	// resolved a customer to its id still filtered on customer.name, because
	// nothing said the id had a field of its own here.
	Key string `json:"key,omitempty"`
	// TargetFields are the keys on the dataset the edge leads to, given by
	// describe_report_dataset so a report can reach one edge out without a
	// second call. A model that had only the edge's name guessed the field
	// on the far side, and guessed wrong.
	TargetFields []string `json:"targetFields,omitempty"`
}

// datasetRow is one dataset in the list. Edges are left to
// describe_report_dataset: with them, the list of seventy ran past one tool
// result and reached the model cut off.
type datasetRow struct {
	Dataset     string `json:"dataset"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	FieldCount  int    `json:"fieldCount"`
}

const (
	defaultDatasetPage = 20
	maxDatasetPage     = 70
	defaultFieldPage   = 40
	maxFieldPage       = 80
	// maxEnumValuesShown keeps an enum's values in a field row while they
	// are a short list to choose from, not a code table.
	maxEnumValuesShown = 12
)

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
	return "List the datasets a new report can be built on, such as shipment, invoice or " +
		"worker. Use it before create_report to pick the dataset, searching with query " +
		"(\"shipment\", \"invoice\") rather than paging through all of them, then call " +
		"describe_report_dataset for its fields. It lists kinds of data, not records: to " +
		"find a customer or a shipment use list_customers or list_shipments."
}

func (t *listReportDatasetsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": withPaging(map[string]any{
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

	query := strings.TrimSpace(optionalString(params.Params, "query"))
	category := optionalString(params.Params, "category")

	criteria := filtercatalog.NewCriteria("datasets").At(clockFor(params))
	criteria.Text(query)
	criteria.Field("category", category)

	words := stringutils.SearchWords(query)
	catalog := &reportcatalog.Default
	rows := make([]datasetRow, 0, len(catalog.Entities))
	for i := range catalog.Entities {
		entity := &catalog.Entities[i]
		if category != "" && !strings.EqualFold(entity.Category, category) {
			continue
		}
		if !matchesDataset(entity, words) {
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
			Description: stringutils.FirstSentence(entity.Description),
			Category:    entity.Category,
			FieldCount:  describedFieldCount(entity),
		})
	}

	window := readPage(params.Params, defaultDatasetPage, maxDatasetPage)
	shown, more := slicePage(window, rows)
	if shown == nil {
		shown = []datasetRow{}
	}

	outcome := searchResult(criteria, shown, len(shown)).paged(window, more)
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

// matchesDataset reports whether every word of the query begins a word of the
// dataset's key, label, plural label or description. It used to look for the
// query as one literal substring, so "shipment delivery" found nothing although
// the shipment dataset is described in both words.
func matchesDataset(entity *reportcatalog.Entity, words []string) bool {
	return stringutils.MatchesWordPrefixes(
		words,
		stringutils.SearchableKey(entity.Key),
		entity.Label,
		entity.PluralLabel,
		entity.Description,
	)
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
		if key, ok := referenceKeyField(&reportcatalog.Default, entity, edge); ok {
			row.Key = key
		}
		if withTargetFields {
			if target, ok := reportcatalog.Default.Entity(edge.Target); ok {
				row.TargetFields = describedFieldKeys(target)
			}
		}
		rows = append(rows, row)
	}

	return rows
}

// datasetFieldRow is one field, in as few characters as say what a report
// needs: its key and type, what it aggregates by, and only the exceptions —
// a field that cannot filter or group, or that this person may not read. The
// widest datasets carry eighty fields, and a row that spelt out every flag and
// its sensitivity pushed the shipment dataset past one tool result.
type datasetFieldRow struct {
	Key          string   `json:"key"`
	Label        string   `json:"label,omitempty"`
	Description  string   `json:"description,omitempty"`
	Type         string   `json:"type"`
	EnumValues   []string `json:"enumValues,omitempty"`
	Aggregations []string `json:"aggs,omitempty"`
	NoFilter     bool     `json:"noFilter,omitempty"`
	NoGroup      bool     `json:"noGroup,omitempty"`
	// Closed marks a field the person's role may not read. The compiler
	// refuses a report that names one, so a model that knows this up front
	// builds a report that compiles rather than one that is refused.
	Closed bool `json:"closed,omitempty"`
}

type datasetDescription struct {
	Dataset     string            `json:"dataset"`
	Label       string            `json:"label"`
	Description string            `json:"description,omitempty"`
	Category    string            `json:"category,omitempty"`
	Fields      []datasetFieldRow `json:"fields"`
	FieldCount  int               `json:"fieldCount"`
	// Shown is how many of them this result carries. It differs from
	// FieldCount when the dataset is wider than one page, and NextOffset
	// then says where the rest continue.
	Shown      int              `json:"shownFieldCount"`
	NextOffset *int             `json:"nextOffset,omitempty"`
	Edges      []datasetEdgeRow `json:"edges,omitempty"`
	// RelatedFields are the fields of related datasets a query matched, as
	// edge.field, so a search names a field one edge out as well as its own.
	RelatedFields []string `json:"relatedFields,omitempty"`
	Note          string   `json:"note"`
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
	return "Describe one report dataset's fields, and the related datasets it can reach. " +
		"Use it after list_report_datasets and before preview_report or create_report, and " +
		"write field keys and edge names exactly as given. Narrow with query (\"revenue\", " +
		"\"delivered\") when you know what you want; a wide dataset pages with offset."
}

func (t *describeReportDatasetTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": withPaging(map[string]any{
			"dataset": map[string]any{
				"type":        "string",
				"description": "The dataset key from list_report_datasets, such as shipment.",
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

	query := strings.TrimSpace(optionalString(params.Params, "query"))
	words := stringutils.SearchWords(query)
	fields := make([]datasetFieldRow, 0, len(entity.Fields))
	for i := range entity.Fields {
		field := &entity.Fields[i]
		if !describedField(field) {
			continue
		}
		if !matchesField(field, words) {
			continue
		}

		_, accessible := t.access.fieldAccessible(entity, field, detail)
		fields = append(fields, datasetFieldRow{
			Key:          field.Key,
			Label:        labelIfNotKey(field),
			Description:  stringutils.FirstSentence(field.Description),
			Type:         string(field.Type),
			EnumValues:   shortEnum(field),
			Aggregations: aggregationNames(field),
			NoFilter:     !field.Filterable,
			NoGroup:      !field.Groupable,
			Closed:       !accessible,
		})
	}

	matched := len(fields)
	window := readPage(params.Params, defaultFieldPage, maxFieldPage)
	fields, more := slicePage(window, fields)
	if fields == nil {
		fields = []datasetFieldRow{}
	}

	note := "Refer to a field of this dataset as {\"field\": \"<key>\"} and to a field " +
		"of a related dataset as {\"path\": [\"<edge>\"], \"field\": \"<key>\"}, " +
		"using only the keys each edge's targetFields lists; an edge two steps " +
		"away needs describe_report_dataset on the first target. A measure " +
		"column needs an agg the field lists; a dimension column groups the " +
		"rows, and a report of dimension columns alone lists rows. When you " +
		"already know a related record's id, filter on the field the edge's key " +
		"names with that id — customerId eq \"cus_…\" — not on the record's " +
		"name, which two records can share and a rename changes. A field " +
		"marked closed cannot be used by this person."

	description := datasetDescription{
		Dataset:     entity.Key,
		Label:       entity.Label,
		Description: entity.Description,
		Category:    entity.Category,
		Fields:      fields,
		FieldCount:  describedFieldCount(entity),
		Shown:       len(fields),
		Edges:       edgeRows(entity, true),
		Note:        note,
	}
	if more {
		next := window.offset + window.limit
		description.NextOffset = &next
		description.Note += fmt.Sprintf(
			" Showing %d of %d matching fields; call this again with offset %d, or with "+
				"query, for the rest rather than assuming they do not exist.",
			len(fields), matched, next,
		)
	}
	if len(words) > 0 {
		description.RelatedFields = relatedFieldMatches(entity, words)
		description.Note += queryNote(entity.Key, query, matched, description.RelatedFields)
	}

	return description, nil
}

// maxRelatedFieldMatches bounds the related fields one query names. A query
// broad enough to pass it matches too much to be a search.
const maxRelatedFieldMatches = 20

// relatedFieldMatches are the fields of the datasets this one's edges reach
// that a query matches, each as edge.field: "arrival" on shipment finds
// destinationStop.actualArrival, which a search of shipment's own fields
// never would.
func relatedFieldMatches(entity *reportcatalog.Entity, words []string) []string {
	matches := make([]string, 0, 4)
	for i := range entity.Edges {
		edge := &entity.Edges[i]
		if !edge.Traversable {
			continue
		}
		target, ok := reportcatalog.Default.Entity(edge.Target)
		if !ok {
			continue
		}
		for j := range target.Fields {
			field := &target.Fields[j]
			if !describedField(field) || !matchesField(field, words) {
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

// queryNote says what a query found when it was narrowed: nothing at all, and
// how to widen; or only fields of related datasets, and how to reach them.
func queryNote(dataset, query string, matched int, related []string) string {
	switch {
	case matched > 0 && len(related) > 0:
		return " relatedFields are fields of related datasets the query also matches, " +
			"written edge.field."
	case matched > 0:
		return ""
	case len(related) > 0:
		edge, field, _ := strings.Cut(related[0], ".")
		return fmt.Sprintf(
			" No field of %s itself matches %q, but fields of related datasets do: "+
				"relatedFields lists them as edge.field, so %s is the field %s on the edge "+
				"%s. Refer to it as {\"path\": [%q], \"field\": %q}.",
			dataset, query, related[0], field, edge, edge, field,
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

// labelIfNotKey keeps a field's label only when it says something its key
// does not: "Pro number" beside proNumber is noise, "Linehaul revenue" beside
// freightChargeAmount is not.
func labelIfNotKey(field *reportcatalog.Field) string {
	squashed := strings.ReplaceAll(strings.ToLower(field.Label), " ", "")
	if squashed == strings.ToLower(field.Key) {
		return ""
	}

	return field.Label
}

// shortEnum keeps an enum's values while they are a short list to choose
// from. A longer one is filtered by value anyway, and the model can ask
// preview_report for the values it sees.
func shortEnum(field *reportcatalog.Field) []string {
	values := enumValueKeys(field)
	if len(values) > maxEnumValuesShown {
		return nil
	}

	return values
}

// matchesField reports whether every word of the query begins a word of the
// field's key, label or description, with a camelCase key read as its words:
// scheduledWindowEnd is found by "window" and by "scheduled end".
func matchesField(field *reportcatalog.Field, words []string) bool {
	return stringutils.MatchesWordPrefixes(
		words,
		stringutils.SearchableKey(field.Key),
		field.Label,
		field.Description,
	)
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
	// Ref is the reference as a definition writes it. Field below is the
	// same thing in one readable string, which is what a model copied into
	// its own definitions until the decoder learned to read it.
	Ref    report.FieldRef `json:"ref"`
	ID     string          `json:"id"`
	Label  string          `json:"label,omitempty"`
	Field  string          `json:"field"`
	Kind   string          `json:"kind"`
	Agg    string          `json:"agg,omitempty"`
	Bucket string          `json:"bucket,omitempty"`
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
			Ref:    column.Ref,
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

	preview := previewOf(source, result, clockFor(params))
	preview.Note += referenceKeyHint(&reportcatalog.Default, source.Definition)

	return preview, nil
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

func previewOf(source reportSource, result *reporting.PreviewResult, clk clock) reportPreview {
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
	for _, column := range result.Columns {
		preview.Columns = append(preview.Columns, shape.add(
			column.ID, column.Label, column.Type, column.Format,
		))
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
	if hasDated(shape.dated) {
		preview.Note += " Date columns are shown here as dates in the organization's timezone, " +
			"and the download renders them the same way; the stored value is epoch seconds."
	}

	return preview
}

// sampleShape is how a report's rows are handed to a model, by preview_report
// and by a finished run alike: keyed by column label, so the model reads a
// table rather than positional tuples, and with date columns written as dates.
// A date comes back from the dataset as epoch seconds, which the download
// renders and a model reading the rows does not.
type sampleShape struct {
	labels []string
	dated  []bool
	clk    clock
}

func newSampleShape(width int, clk clock) sampleShape {
	return sampleShape{
		labels: make([]string, 0, width),
		dated:  make([]bool, 0, width),
		clk:    clk,
	}
}

// add records one column and returns the row that describes it.
func (s *sampleShape) add(
	id, label string,
	fieldType reportcatalog.FieldType,
	format reportcatalog.FormatHint,
) previewColumnRow {
	if label == "" {
		label = id
	}
	s.labels = append(s.labels, label)
	s.dated = append(s.dated, fieldType == reportcatalog.FieldEpoch)

	return previewColumnRow{
		ID:     id,
		Label:  label,
		Type:   string(fieldType),
		Format: string(format),
	}
}

func (s *sampleShape) row(row serviceports.ReportRow) map[string]any {
	return rowByLabel(s.labels, renderDates(row, s.dated, s.clk))
}

// sampleRows renders at most limit rows from the top of a result.
func sampleRows[R ~[]any](shape sampleShape, rows []R, limit int) []map[string]any {
	shown := min(len(rows), limit)
	out := make([]map[string]any, 0, shown)
	for _, row := range rows[:shown] {
		out = append(out, shape.row(serviceports.ReportRow(row)))
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

// renderDates writes each epoch cell as a readable date.
func renderDates(row serviceports.ReportRow, dated []bool, clk clock) serviceports.ReportRow {
	out := make(serviceports.ReportRow, len(row))
	for idx, value := range row {
		if idx < len(dated) && dated[idx] {
			if seconds, ok := epochSeconds(value); ok {
				out[idx] = time.Unix(seconds, 0).In(clk.Location()).Format("2006-01-02 15:04 MST")

				continue
			}
		}
		out[idx] = value
	}

	return out
}

func epochSeconds(value any) (int64, bool) {
	switch typed := value.(type) {
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

func rowByLabel(labels []string, row serviceports.ReportRow) map[string]any {
	out := make(map[string]any, len(labels))
	for i, label := range labels {
		if i < len(row) {
			out[label] = row[i]
		}
	}

	return out
}
