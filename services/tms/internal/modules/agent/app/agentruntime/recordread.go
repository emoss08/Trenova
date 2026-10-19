package agentruntime

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const (
	recordReadPrefix = "get_"
	recordReadSuffix = "_get"
)

type RecordRead struct {
	Tool   string
	Params map[string]any
	Data   any
}

func (s *Service) ReadRecord(
	ctx context.Context,
	actor *permissionports.RequestActor,
	ref *agent.RecordRef,
) (*RecordRead, bool) {
	if ref == nil || s.queryTools == nil {
		return nil, false
	}

	name := ref.EntityType + recordReadSuffix
	if _, ok := s.queryTools.Get(name); !ok {
		name = recordReadPrefix + ref.EntityType
	}

	return s.ReadRecordWith(ctx, actor, name, ref.ID)
}

func (s *Service) ReadRecordWith(
	ctx context.Context,
	actor *permissionports.RequestActor,
	name string,
	id string,
) (*RecordRead, bool) {
	if actor == nil || strings.TrimSpace(id) == "" || s.queryTools == nil {
		return nil, false
	}

	tool, ok := s.queryTools.Get(name)
	if !ok || len(s.permittedTools(ctx, actor, []string{name})) != 1 {
		return nil, false
	}
	param, ok := recordIDParam(tool.ParamSchema())
	if !ok {
		return nil, false
	}

	params := map[string]any{param: id}
	data, err := tool.Query(ctx, &agentports.QueryToolParams{
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Actor:          actor,
		Params:         params,
	})
	if err != nil || data == nil {
		return nil, false
	}

	return &RecordRead{Tool: name, Params: params, Data: data}, true
}

func recordIDParam(schema map[string]any) (string, bool) {
	if required, _ := schema[toolschema.KeyRequired].([]string); len(required) == 1 {
		return required[0], true
	}

	properties, _ := schema[toolschema.KeyProperties].(map[string]any)
	found := ""
	for name, raw := range properties {
		property, _ := raw.(map[string]any)
		if toolschema.RecordOf(property) == "" {
			continue
		}
		if found != "" {
			return "", false
		}
		found = name
	}

	return found, found != ""
}
