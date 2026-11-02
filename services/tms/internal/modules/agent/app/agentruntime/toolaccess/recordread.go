package toolaccess

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
)

const (
	recordReadPrefix = "get_"
)

type RecordRead struct {
	Tool   string
	Params map[string]any
	Data   any
}

func (s *Access) ReadRecord(
	ctx context.Context,
	actor *permissionports.RequestActor,
	ref *agentrecord.RecordRef,
) (*RecordRead, bool) {
	if ref == nil {
		return nil, false
	}

	name := agenttoolports.ToolNameFor(recordReadPrefix + ref.EntityType)
	if _, ok := s.queryTools.Get(name); !ok {
		name = recordReadPrefix + ref.EntityType
	}

	return s.ReadRecordWith(ctx, actor, name, ref.ID)
}

func (s *Access) ReadRecordWith(
	ctx context.Context,
	actor *permissionports.RequestActor,
	name string,
	id string,
) (*RecordRead, bool) {
	if actor == nil || strings.TrimSpace(id) == "" {
		return nil, false
	}

	tool, ok := s.queryTools.Get(name)
	if !ok || len(s.PermittedTools(ctx, actor, []string{name})) != 1 {
		return nil, false
	}
	params, ok := agenttoolports.RecordIDParams(tool.ParamSchema(), id)
	if !ok {
		return nil, false
	}
	data, err := tool.Query(ctx, &agenttoolports.QueryToolParams{
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
