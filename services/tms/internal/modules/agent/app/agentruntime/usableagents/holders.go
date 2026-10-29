package usableagents

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// otherAgentsPage is how many agents one read asks for, the most a list page
// reads; maxOtherAgents bounds how many are read in all.
const (
	otherAgentsPage = pagination.MaxLimit
	maxOtherAgents  = 1000
)

// OthersRequest names whose other agents to read.
type OthersRequest struct {
	Permissions permissionports.PermissionEngine
	Definitions agentdefinitionports.AgentDefinitionRepository
	Actor       *permissionports.RequestActor
	// Self is the agent the person is already talking to, left out.
	Self pulid.ID
}

// Others is every other enabled chat agent the person may use,
// by name. The Desk's case checklist reads it to say who can take a step,
// and the runtime to say which agent holds a tool the conversation's agent
// does not.
func Others(
	ctx context.Context,
	req *OthersRequest,
) ([]*agentdefinition.Definition, error) {
	if req.Permissions == nil || req.Definitions == nil || req.Actor == nil {
		return nil, nil
	}
	usable, err := req.Permissions.AgentsUsable(ctx, req.Actor, permission.OpCreate)
	if err != nil {
		return nil, fmt.Errorf("read the agents the person may use: %w", err)
	}
	if usable == nil || !usable.Assistant {
		return nil, nil
	}

	out := make([]*agentdefinition.Definition, 0, otherAgentsPage)
	seen := make(map[pulid.ID]struct{}, otherAgentsPage)
	for offset := 0; offset < maxOtherAgents; {
		result, listErr := req.Definitions.List(
			ctx,
			&agentdefinitionports.ListAgentDefinitionRequest{
				Filter: &pagination.QueryOptions{
					TenantInfo: req.Actor.TenantInfo(),
					Pagination: pagination.Info{Limit: otherAgentsPage, Offset: offset},
				},
				EnabledOnly: true,
				ChatOnly:    true,
				Audience:    usable.Audience(),
			},
		)
		if listErr != nil {
			return nil, fmt.Errorf("list the agents the person may use: %w", listErr)
		}
		for _, definition := range result.Items {
			if definition == nil || definition.ID == req.Self || !usable.Allows(definition) {
				continue
			}
			if _, dup := seen[definition.ID]; dup {
				continue
			}
			seen[definition.ID] = struct{}{}
			out = append(out, definition)
		}
		offset += len(result.Items)
		if len(result.Items) == 0 || (result.TotalExact && offset >= result.Total) {
			break
		}
	}
	slices.SortStableFunc(out, func(a, b *agentdefinition.Definition) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return out, nil
}
