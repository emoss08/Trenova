package gettool

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type Tool struct {
	Spec   readspec.Get
	Access fieldgate.Access
}

func New(
	spec *readspec.Get,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return &Tool{Spec: *spec, Access: fieldgate.NewAccess(permissions)}
}

func (t *Tool) Name() string { return t.Spec.Name }

func (t *Tool) Description() string { return t.Spec.Summary }

func (t *Tool) SearchTerms() []string { return t.Spec.SearchTerms }

func (t *Tool) Policy() agenttoolports.ToolPolicy {
	return querytool.ReadPolicy(t.Name(), querytool.ReadSpec{
		Resource:  t.Spec.Resource,
		Scope:     t.Spec.Scope,
		Reads:     t.Spec.Reads,
		Source:    t.Spec.Source,
		Rationale: t.Spec.Rationale,
	})
}

// ParamSchema marks the id with the tool's resource when that resource is one
// kind of record with one prefix: the record it reads is the resource it is
// permitted under. A tool reading one kind of a resource that covers several,
// a detention occurrence under detention, names that kind instead, so its id
// is still held to a prefix rather than checked for shape alone.
func (t *Tool) ParamSchema() map[string]any {
	property := agenttoolschema.IDText(t.Spec.IDDescription())
	if t.Spec.KeyResource != "" {
		property = agenttoolschema.RecordIDText(t.Spec.KeyResource, t.Spec.IDDescription())
	} else if len(t.Spec.Kinds) > 0 {
		property = agenttoolschema.KindID(t.Spec.IDDescription(), t.Spec.Kinds...)
	} else if _, typed := recordkind.ResourcePrefix(t.Spec.Resource); typed {
		property = agenttoolschema.RecordIDText(t.Spec.Resource, t.Spec.IDDescription())
	}

	return IDSchema(t.Spec.ParamName, property)
}

func (t *Tool) Query(
	ctx context.Context,
	params *agenttoolports.QueryToolParams,
) (any, error) {
	if err := querytool.GuardQuery(params); err != nil {
		return nil, err
	}

	// Parsing rather than passing the string through is what turns a
	// paraphrased id — or a pro number pasted where an id belongs — into a
	// plain error, instead of an empty result the model reports as a missing
	// record.
	id, err := querytool.RequirePulid(params.Params, t.Spec.ParamName)
	if err != nil {
		return nil, err
	}

	entity, err := t.Spec.Fetch(ctx, id, pagination.TenantInfo{
		OrgID:  params.OrganizationID,
		BuID:   params.BusinessUnitID,
		UserID: params.Actor.UserID,
	})
	if err != nil {
		return nil, err
	}

	return t.Access.Redactor(ctx, params).Withhold(entity)
}

func specOfGet(tool agenttoolports.AgentQueryTool) readspec.Get {
	return tool.(*Tool).Spec //nolint:errcheck,forcetypeassert // constructed above
}

func IDSchema(param string, property map[string]any) map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{param: property},
		"required":             []string{param},
		"additionalProperties": false,
	}
}
