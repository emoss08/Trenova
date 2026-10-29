package attribution

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// AgentName reads the agent's name for the heading. An agent that has since
// been deleted leaves the conversation readable under a plain name.
func (s *Attributor) AgentName(
	ctx context.Context,
	req *conversationports.GetThreadRequest,
	thread *conversation.Thread,
) string {
	if s.Definitions == nil || thread.AgentDefinitionID.IsNil() {
		return "Assistant"
	}

	definition, err := s.Definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         thread.AgentDefinitionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil || definition == nil || definition.Name == "" {
		if err != nil && !errortypes.IsNotFoundError(err) {
			s.Logger.Warn("transcript could not name the agent")
		}

		return "Assistant"
	}

	return definition.Name
}
