package recordquerytools

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const describeEntityDescription = "List an entity's fields records_query can filter and sort by, " +
	"its entity filters, and the sections record_get and records_query can include. Use it when " +
	"a field name is unknown, rather than guessing."

type DescribeTool struct{ linked }

func NewDescribeTool() agenttoolports.AgentQueryTool { return &DescribeTool{} }

func (t *DescribeTool) Name() string { return agentcapability.DescribeEntity }

func (t *DescribeTool) Description() string { return describeEntityDescription }

func (t *DescribeTool) Policy() agenttoolports.ToolPolicy {
	return describePolicy(t.Name(), "Describes an entity's fields from the registry; it reads no record.")
}

func (t *DescribeTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			paramEntity: map[string]any{
				toolschema.KeyType:        toolschema.TypeString,
				toolschema.KeyDescription: "An entity name, as records_query takes it.",
			},
		},
		toolschema.KeyRequired:             []string{paramEntity},
		toolschema.KeyAdditionalProperties: false,
	}
}

type Description struct {
	Entity    string            `json:"entity"`
	Title     string            `json:"title"`
	Area      string            `json:"area"`
	Query     bool              `json:"query"`
	Get       bool              `json:"get"`
	Operators []string          `json:"operators"`
	Fields    []string          `json:"fields"`
	Sortable  []string          `json:"sortable,omitempty"`
	Filters   []querytool.Extra `json:"entityFilters,omitempty"`
	Aliases   map[string]string `json:"aliases,omitempty"`
	Include   []string          `json:"include,omitempty"`
	Truncated *Truncation       `json:"truncated,omitempty"`
}

func (t *DescribeTool) Query(_ context.Context, params *agenttoolports.QueryToolParams) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}
	backing, err := Backing(t.Name(), params.Params)
	if err != nil {
		return nil, err
	}
	entity, _, _ := agentcapability.EntityBacking(backing)
	out := &Description{
		Entity:    entity.Name,
		Title:     entity.Title,
		Area:      entity.Area,
		Query:     entity.List != "",
		Get:       entity.Get != "",
		Operators: operators,
		Aliases:   entity.Aliases,
		Include:   entity.IncludeNames(),
	}
	if entity.List == "" {
		return out, nil
	}
	reader, err := t.reader(entity.List)
	if err != nil {
		return nil, err
	}
	fields := fieldsOf(reader, &entity)
	out.Filters = fields.extras
	out.Fields = fields.publicFields()
	if len(out.Fields) > maxDescribedKeys {
		out.Fields = out.Fields[:maxDescribedKeys]
		out.Truncated = &Truncation{Reason: "the related records' fields past the first are left out"}
	}
	if fields.config != nil {
		for field, on := range fields.config.SortableFields {
			if on && public(field) {
				out.Sortable = append(out.Sortable, field)
			}
		}
		slices.Sort(out.Sortable)
	}

	return out, nil
}
