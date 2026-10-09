package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/subjectaccess"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Service) mayReadSubject(
	ctx context.Context,
	actor *services.RequestActor,
	subjectType agent.SubjectType,
) (bool, error) {
	return subjectaccess.MayRead(ctx, s.permissions, actor, subjectType)
}

func (s *Service) assertSubjectReadable(
	ctx context.Context,
	actor *services.RequestActor,
	subjectType agent.SubjectType,
) error {
	return subjectaccess.AssertReadable(ctx, s.permissions, actor, subjectType)
}

// assertChatAgent refuses a conversation with an agent built to run on its
// own. A desk's earned autonomy was earned on its own work; a person talking
// to it would borrow that autonomy for writes of their own choosing.
func assertChatAgent(definition *agentdefinition.Definition) error {
	if definition.TriggerMode != "" && definition.TriggerMode != agentdefinition.TriggerChat {
		return errortypes.NewBusinessError(
			"Agent {0} runs on its own and cannot be talked to", definition.Name,
		)
	}

	return nil
}
