package agentruntime

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
	"go.uber.org/zap"
)

// otherAgentsPage is how many agents one read asks for, the most a list page
// reads; maxOtherAgents bounds how many are read in all.
const (
	otherAgentsPage = pagination.MaxLimit
	maxOtherAgents  = 1000
)

// OtherUsableAgentsRequest names whose other agents to read.
type OtherUsableAgentsRequest struct {
	Permissions permissionports.PermissionEngine
	Definitions agentdefinitionports.AgentDefinitionRepository
	Actor       *permissionports.RequestActor
	// Self is the agent the person is already talking to, left out.
	Self pulid.ID
}

// OtherUsableAgents is every other enabled chat agent the person may use,
// by name. The Desk's case checklist reads it to say who can take a step,
// and the runtime to say which agent holds a tool the conversation's agent
// does not.
func OtherUsableAgents(
	ctx context.Context,
	req *OtherUsableAgentsRequest,
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
		if len(result.Items) == 0 || offset >= result.Total {
			break
		}
	}
	slices.SortStableFunc(out, func(a, b *agentdefinition.Definition) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return out, nil
}

// handOffNote names the person's other agents that hold tools the
// conversation's agent does not, so the model tells the person which agent
// to hand the conversation to.
//
// A shipment agent asked to mark a load ready to invoice found the transfer
// tool existed but was not its own, and told the person to ask an
// administrator; the person guessed at an agent in the hand-off menu and
// picked the billing exceptions analyst rather than the billing assistant.
// The agents that hold the tool were there to be named. It is read only when
// a find_tools search matches nothing the agent holds, so the listing costs
// nothing on an ordinary turn, and a failure to read it leaves the note out.
//
// The agents it names are returned too, in the note's order, and ride on the
// find_tools answer into the saved tool message, so the Desk's hand-off menu
// can put them first rather than leave the person to match a name in prose.
func (s *Service) handOffNote(
	ctx context.Context,
	actor *permissionports.RequestActor,
	self pulid.ID,
	tools []string,
) (string, []pulid.ID) {
	if len(tools) == 0 || actor == nil || actor.PrincipalType != permissionports.PrincipalTypeUser {
		return "", nil
	}
	others, err := OtherUsableAgents(ctx, &OtherUsableAgentsRequest{
		Permissions: s.permissions,
		Definitions: s.definitions,
		Actor:       actor,
		Self:        self,
	})
	if err != nil {
		s.logger.Warn("could not read which other agents hold a tool", zap.Error(err))

		return "", nil
	}

	lines := make([]string, 0, len(others))
	holders := make([]pulid.ID, 0, len(others))
	for _, other := range others {
		held := s.heldTools(other)
		matched := make([]string, 0, len(tools))
		for _, name := range tools {
			if slices.Contains(held, name) {
				matched = append(matched, name)
			}
		}
		if len(matched) > 0 {
			lines = append(lines, "- "+other.Name+": "+strings.Join(matched, ", "))
			holders = append(holders, other.ID)
		}
	}
	if len(lines) == 0 {
		return "", nil
	}

	note := "These of the person's other agents hold them:\n" + strings.Join(lines, "\n") +
		"\nTell the person which agent can do this, by its name, and that they can hand " +
		"this conversation to it with Hand off to another agent at the top of the " +
		"conversation; it carries a summary over. Say what you would have done so they " +
		"know what to ask it."

	return note, holders
}
