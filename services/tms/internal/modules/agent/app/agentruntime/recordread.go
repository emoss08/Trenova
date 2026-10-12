package agentruntime

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const recordReadPrefix = "get_"

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
	if actor == nil || ref == nil || strings.TrimSpace(ref.ID) == "" || s.queryTools == nil {
		return nil, false
	}

	name := recordReadPrefix + ref.EntityType
	tool, ok := s.queryTools.Get(name)
	if !ok || len(s.permittedTools(ctx, actor, []string{name})) != 1 {
		return nil, false
	}
	required, _ := tool.ParamSchema()[toolschema.KeyRequired].([]string)
	if len(required) != 1 {
		return nil, false
	}

	params := map[string]any{required[0]: ref.ID}
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
