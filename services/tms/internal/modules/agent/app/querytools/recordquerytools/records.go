package recordquerytools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const recordsQueryDescription = "Find, filter, count or group any entity's records in one call. " +
	"where conditions all hold; each or group holds when any of its conditions does. Fields may " +
	"follow relations (customer.name) or be entity filters (stops.city, driverId, unassigned); " +
	"describe_entity lists them. countOnly returns the total only; groupBy counts by a field; " +
	"include adds related sections. Dates are YYYY-MM-DD."

const entityHint = "shipment, invoice, worker, tractor, trailer, customer, carrier, location, " +
	"order, driver_settlement, carrier_settlement, billing_queue_item, customer_payment, " +
	"journal_entry, fuel_purchase, edi_transfer, worker_credential; search_tools names the rest."

type QueryTool struct{ linked }

func NewQueryTool() agenttoolports.AgentQueryTool { return &QueryTool{} }

func (t *QueryTool) Name() string { return agentcapability.RecordsQuery }

func (t *QueryTool) Description() string { return recordsQueryDescription }

func (t *QueryTool) SearchTerms() []string {
	return []string{"list", "find", "filter", "count", "how many", "records"}
}

func (t *QueryTool) Policy() agenttoolports.ToolPolicy {
	return readPolicy(t.Name(), "Reads one entity's records through that entity's own read, "+
		"which checks the entity's permission, scope and plan for every call.")
}

func conditionSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"field":  map[string]any{toolschema.KeyType: toolschema.TypeString},
			"op":     map[string]any{toolschema.KeyType: toolschema.TypeString, "enum": operatorSource.Names(), "x-enumOf": operatorSource.Name},
			"value":  map[string]any{},
			"values": map[string]any{toolschema.KeyType: "array"},
			"days":   map[string]any{toolschema.KeyType: "integer"},
		},
		toolschema.KeyRequired:             []string{"field", "op"},
		toolschema.KeyAdditionalProperties: false,
	}
}

var (
	operatorSource  = agenttoolschema.Source("records_query.operator", operators)
	directionSource = agenttoolschema.Source("records_query.direction", []string{"asc", "desc"})
)

func sortSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"field": map[string]any{toolschema.KeyType: toolschema.TypeString},
			"direction": map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				"enum":             directionSource.Names(),
				"x-enumOf":         directionSource.Name,
			},
		},
		toolschema.KeyRequired:             []string{"field"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func text(description string) map[string]any {
	property := map[string]any{toolschema.KeyType: toolschema.TypeString}
	if description != "" {
		property[toolschema.KeyDescription] = description
	}

	return property
}

func list(description string, items map[string]any) map[string]any {
	property := map[string]any{toolschema.KeyType: "array", "items": items}
	if description != "" {
		property[toolschema.KeyDescription] = description
	}

	return property
}

func (t *QueryTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramEntity:          text(entityHint),
			paramWhere:           list("", conditionSchema()),
			paramOr:              list("Groups of conditions.", map[string]any{toolschema.KeyType: "array", "items": conditionSchema()}),
			querytool.ParamQuery: text("Free text: names, numbers, words."),
			paramSort:            list("", sortSchema()),
			"fields":             list("Columns to keep.", text("")),
			paramInclude:         list("Related sections, or all.", text("")),
			paramCountOnly:       map[string]any{toolschema.KeyType: "boolean"},
			paramGroupBy:         text(""),
			querytool.ParamLimit: map[string]any{toolschema.KeyType: "integer", "maximum": maxLimit},
			querytool.ParamAfter: text("Previous page's nextCursor."),
		},
		toolschema.KeyRequired:             []string{paramEntity},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *QueryTool) Query(ctx context.Context, params *agenttoolports.QueryToolParams) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	backing, err := Backing(t.Name(), params.Params)
	if err != nil {
		return nil, err
	}
	entity, _, _ := agentcapability.EntityBacking(backing)
	reader, err := t.reader(backing)
	if err != nil {
		return nil, err
	}
	and, err := conditions(params.Params[paramWhere])
	if err != nil {
		return nil, err
	}
	var or [][]condition
	if groups, ok := params.Params[paramOr].([]any); ok {
		for _, group := range groups {
			parsed, groupErr := conditions(group)
			if groupErr != nil {
				return nil, groupErr
			}
			or = append(or, parsed)
		}
	}
	sorting, err := sortFields(params.Params[paramSort])
	if err != nil {
		return nil, err
	}
	includes, err := includeNames(params.Params[paramInclude], &entity)
	if err != nil {
		return nil, err
	}
	fields := fieldsOf(reader, &entity)
	plan, err := compile(fields, and, or, sorting, zoneOf(params))
	if err != nil {
		return nil, err
	}

	values := map[string]any{}
	for _, key := range []string{querytool.ParamQuery, querytool.ParamAfter, querytool.ParamLimit, "fields"} {
		if value, ok := params.Params[key]; ok {
			values[key] = value
		}
	}
	if _, ok := values[querytool.ParamLimit]; !ok {
		values[querytool.ParamLimit] = defaultLimit
	}
	values = querytool.WithWhere(values, &plan.where)

	envelope := &Envelope{Entity: entity.Name, AppliedFilters: plan.applied}
	countOnly, _ := params.Params[paramCountOnly].(bool)
	groupBy := strings.TrimSpace(querytool.OptionalString(params.Params, paramGroupBy))
	switch {
	case countOnly:
		return t.count(ctx, reader, params, values, envelope)
	case groupBy != "":
		return t.group(ctx, reader, params, values, groupBy, envelope)
	}

	inner, err := queryInner(ctx, reader, params, values)
	if err != nil {
		return nil, err
	}
	fill(envelope, inner)
	envelope.AppliedFilters = plan.applied
	if len(includes) > 0 {
		attachIncludes(ctx, &t.linked, params, &entity, includes, rowIDs(envelope.Items), envelope)
	}

	return envelope, nil
}

func (t *QueryTool) count(
	ctx context.Context,
	reader agenttoolports.AgentQueryTool,
	params *agenttoolports.QueryToolParams,
	values map[string]any,
	envelope *Envelope,
) (any, error) {
	counted := *params
	counted.IncludeTotal = true
	values[querytool.ParamLimit] = 1
	inner, err := queryInner(ctx, reader, &counted, values)
	if err != nil {
		return nil, err
	}
	applied := envelope.AppliedFilters
	fill(envelope, inner)
	envelope.AppliedFilters = applied
	envelope.Items = []any{}
	envelope.HasMore = false
	envelope.NextCursor = ""
	if envelope.Total == nil {
		return t.group(ctx, reader, params, values, "", envelope)
	}

	return envelope, nil
}

func (t *QueryTool) group(
	ctx context.Context,
	reader agenttoolports.AgentQueryTool,
	params *agenttoolports.QueryToolParams,
	values map[string]any,
	field string,
	envelope *Envelope,
) (any, error) {
	counts := map[string]int{}
	rows := 0
	values[querytool.ParamLimit] = groupPageSize
	delete(values, querytool.ParamAfter)
	more := false
	for rows < maxGroupRows {
		inner, err := queryInner(ctx, reader, params, values)
		if err != nil {
			return nil, err
		}
		page := &Envelope{}
		fill(page, inner)
		envelope.extra = append(envelope.extra, inner)
		items, _ := page.Items.([]any)
		rows += len(items)
		if field != "" {
			groupCounts(page.Items, field, counts)
		}
		envelope.OmittedForAccess = append(envelope.OmittedForAccess, page.OmittedForAccess...)
		more = page.HasMore && page.NextCursor != ""
		if !more {
			break
		}
		values[querytool.ParamAfter] = page.NextCursor
	}
	total := rows
	envelope.Total = &total
	envelope.Items = []any{}
	envelope.HasMore = false
	if more {
		envelope.TotalEstimated = true
		envelope.Truncated = &Truncation{Reason: fmt.Sprintf(
			"counted the first %d rows; narrow the conditions for an exact figure", maxGroupRows)}
	}
	if field != "" {
		envelope.Groups = sortedGroups(counts)
	}

	return envelope, nil
}

func attachIncludes(
	ctx context.Context,
	l *linked,
	params *agenttoolports.QueryToolParams,
	entity *agentcapability.Entity,
	includes []string,
	ids []string,
	envelope *Envelope,
) {
	if len(ids) == 0 {
		return
	}
	if len(ids) > maxIncludedRows {
		ids = ids[:maxIncludedRows]
		envelope.Truncated = &Truncation{Reason: fmt.Sprintf(
			"related sections were read for the first %d rows", maxIncludedRows)}
	}
	envelope.Includes = map[string]any{}
	envelope.OmittedIncludes = map[string]string{}
	summary := false
	for _, name := range includes {
		include, _ := entity.Include(name)
		if include.FromSummary {
			summary = true
			continue
		}
		reader, err := l.reader(include.Reader)
		if err != nil {
			envelope.OmittedIncludes[name] = err.Error()
			continue
		}
		section, err := readKeyed(ctx, reader, params, include.Key, ids)
		if err != nil {
			envelope.OmittedIncludes[name] = err.Error()
			continue
		}
		envelope.Includes[name] = section
		envelope.extra = append(envelope.extra, section)
	}
	if summary && entity.Get != "" {
		reader, err := l.reader(entity.Get)
		if err == nil {
			key, _ := idParam(reader.ParamSchema())
			if section, readErr := readKeyed(ctx, reader, params, key, ids); readErr == nil {
				envelope.Includes["record"] = section
				envelope.extra = append(envelope.extra, section)
			} else {
				envelope.OmittedIncludes["record"] = readErr.Error()
			}
		}
	}
	if len(envelope.OmittedIncludes) == 0 {
		envelope.OmittedIncludes = nil
	}
}

func readKeyed(
	ctx context.Context,
	reader agenttoolports.AgentQueryTool,
	params *agenttoolports.QueryToolParams,
	key string,
	ids []string,
) (any, error) {
	if key == "" {
		return nil, fmt.Errorf("%s takes no record id", reader.Name())
	}
	if strings.HasSuffix(key, "Ids") {
		return queryInner(ctx, reader, params, map[string]any{key: anyList(ids)})
	}
	sections := make(map[string]any, len(ids))
	for _, id := range ids {
		section, err := queryInner(ctx, reader, params, map[string]any{key: id})
		if err != nil {
			sections[id] = map[string]any{"error": err.Error()}
			continue
		}
		sections[id] = section
	}

	return sections, nil
}

func anyList(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}

	return out
}
