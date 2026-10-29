package toolaccess

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/usableagents"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

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
func (s *Access) handOffNote(
	ctx context.Context,
	actor *permissionports.RequestActor,
	self pulid.ID,
	tools []string,
) (string, []pulid.ID) {
	if len(tools) == 0 || actor == nil || actor.PrincipalType != permissionports.PrincipalTypeUser {
		return "", nil
	}
	others, err := usableagents.Others(ctx, &usableagents.OthersRequest{
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
		held := s.HeldTools(other)
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
