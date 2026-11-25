package recordquerytools

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const recordGetDescription = "Read records of any entity in full by ids (pass every id at once) " +
	"or one by its number (a PRO), with related sections through include (a shipment's comments, " +
	"tracking, documents, permits; all for every one)."

type GetTool struct{ linked }

func NewGetTool() agenttoolports.AgentQueryTool { return &GetTool{} }

func (t *GetTool) Name() string { return agentcapability.RecordGet }

func (t *GetTool) Description() string { return recordGetDescription }

func (t *GetTool) SearchTerms() []string {
	return []string{"details", "everything about", "show record", "open"}
}

func (t *GetTool) Policy() agenttoolports.ToolPolicy {
	return readPolicy(t.Name(), "Reads records through each entity's own read, which checks "+
		"the entity's permission, scope and plan for every call.")
}

func (t *GetTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramEntity:  text("As records_query takes it."),
			paramIDs:     list("", text("")),
			paramNumber:  text(""),
			paramInclude: list("", text("")),
		},
		toolschema.KeyRequired:             []string{paramEntity},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *GetTool) Query(ctx context.Context, params *agenttoolports.QueryToolParams) (any, error) {
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
	includes, err := includeNames(params.Params[paramInclude], &entity)
	if err != nil {
		return nil, err
	}
	ids := querytool.OptionalStrings(params.Params, paramIDs)
	number := strings.TrimSpace(querytool.OptionalString(params.Params, paramNumber))
	key, list := idParam(reader.ParamSchema())
	values := map[string]any{}
	switch {
	case len(ids) > 0 && key == "":
		return nil, fmt.Errorf("%w: %s is read by its number, not an id", ErrBadArgument, entity.Name)
	case len(ids) > 0 && list:
		values[key] = anyList(ids)
	case len(ids) == 1:
		values[key] = ids[0]
	case len(ids) > 1:
		return t.each(ctx, reader, params, key, ids, &entity, includes)
	case number != "":
		field := numberParam(reader.ParamSchema())
		if field == "" {
			return nil, fmt.Errorf("%w: %s has no number to read it by; pass ids", ErrBadArgument, entity.Name)
		}
		values[field] = number
	default:
		return nil, fmt.Errorf("%w: pass ids or number", ErrBadArgument)
	}

	inner, err := queryInner(ctx, reader, params, values)
	if err != nil {
		return nil, err
	}
	envelope := &Envelope{Entity: entity.Name, AppliedFilters: []Applied{}}
	fill(envelope, inner)
	if len(includes) > 0 {
		found := ids
		if len(found) == 0 {
			found = recordIDs(envelope.Items)
		}
		withSections(ctx, &t.linked, params, &entity, includes, found, envelope)
	}

	return envelope, nil
}

func (t *GetTool) each(
	ctx context.Context,
	reader agenttoolports.AgentQueryTool,
	params *agenttoolports.QueryToolParams,
	key string,
	ids []string,
	entity *agentcapability.Entity,
	includes []string,
) (any, error) {
	envelope := &Envelope{Entity: entity.Name, AppliedFilters: []Applied{}}
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		inner, err := queryInner(ctx, reader, params, map[string]any{key: id})
		if err != nil {
			items = append(items, map[string]any{"id": id, "error": err.Error()})
			continue
		}
		envelope.extra = append(envelope.extra, inner)
		page := &Envelope{}
		fill(page, inner)
		items = append(items, page.Items)
		envelope.OmittedForAccess = append(envelope.OmittedForAccess, page.OmittedForAccess...)
	}
	envelope.Items = items
	if len(includes) > 0 {
		withSections(ctx, &t.linked, params, entity, includes, ids, envelope)
	}

	return envelope, nil
}

func withSections(
	ctx context.Context,
	l *linked,
	params *agenttoolports.QueryToolParams,
	entity *agentcapability.Entity,
	includes []string,
	ids []string,
	envelope *Envelope,
) {
	readers := slices.DeleteFunc(slices.Clone(includes), func(name string) bool {
		include, _ := entity.Include(name)
		return include.FromSummary
	})
	if len(readers) == 0 {
		return
	}
	attachIncludes(ctx, l, params, entity, readers, ids, envelope)
}

func recordIDs(items any) []string {
	if ids := rowIDs(items); len(ids) > 0 {
		return ids
	}
	object, ok := items.(map[string]any)
	if !ok {
		return nil
	}
	if id, hasID := object["id"].(string); hasID {
		return []string{id}
	}

	return nil
}
