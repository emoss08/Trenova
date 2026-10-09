package agentruntime

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// maxOtherAgents bounds how many of the person's other agents are read for
// one that holds a tool; it is the most a list page reads.
const maxOtherAgents = 100

// OtherUsableAgentsRequest names whose other agents to read.
type OtherUsableAgentsRequest struct {
	Permissions serviceports.PermissionEngine
	Definitions repositories.AgentDefinitionRepository
	Actor       *serviceports.RequestActor
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

	result, err := req.Definitions.List(ctx, &repositories.ListAgentDefinitionRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: req.Actor.TenantInfo(),
			Pagination: pagination.Info{Limit: maxOtherAgents},
		},
		EnabledOnly: true,
		ChatOnly:    true,
		Audience:    usable.Audience(),
	})
	if err != nil {
		return nil, fmt.Errorf("list the agents the person may use: %w", err)
	}

	out := make([]*agentdefinition.Definition, 0, len(result.Items))
	for _, definition := range result.Items {
		if definition == nil || definition.ID == req.Self || !usable.Allows(definition) {
			continue
		}
		out = append(out, definition)
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
func (s *Service) handOffNote(
	ctx context.Context,
	actor *serviceports.RequestActor,
	self pulid.ID,
	tools []string,
) string {
	if len(tools) == 0 || actor == nil || actor.PrincipalType != serviceports.PrincipalTypeUser {
		return ""
	}
	others, err := OtherUsableAgents(ctx, &OtherUsableAgentsRequest{
		Permissions: s.permissions,
		Definitions: s.definitions,
		Actor:       actor,
		Self:        self,
	})
	if err != nil {
		s.logger.Warn("could not read which other agents hold a tool", zap.Error(err))

		return ""
	}

	lines := make([]string, 0, len(others))
	for _, other := range others {
		held := other.EffectiveToolNames()
		matched := make([]string, 0, len(tools))
		for _, name := range tools {
			if slices.Contains(held, name) {
				matched = append(matched, name)
			}
		}
		if len(matched) > 0 {
			lines = append(lines, "- "+other.Name+": "+strings.Join(matched, ", "))
		}
	}
	if len(lines) == 0 {
		return ""
	}

	return "These of the person's other agents hold them:\n" + strings.Join(lines, "\n") +
		"\nTell the person which agent can do this, by its name, and that they can hand " +
		"this conversation to it with Hand off to another agent at the top of the " +
		"conversation; it carries a summary over. Say what you would have done so they " +
		"know what to ask it."
}
