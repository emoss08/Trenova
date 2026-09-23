package assistantservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// mayReadSubject reports whether the person may read the record a
// conversation is about. The subject is described into the prompt on every
// turn, so a conversation pinned to a record the person cannot open would
// read it to them through the model. A kind of subject that names no record
// needs no grant; a permission that cannot be checked is a refusal.
func (s *Service) mayReadSubject(
	ctx context.Context,
	actor *services.RequestActor,
	subjectType agent.SubjectType,
) (bool, error) {
	resource, names := subjectType.Resource()
	if !names {
		return true, nil
	}
	if s.permissions == nil || actor == nil {
		return false, nil
	}

	result, err := s.permissions.Check(ctx, &services.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       resource.String(),
		Operation:      permission.OpRead,
	})
	if err != nil {
		return false, fmt.Errorf("check access to the conversation's subject: %w", err)
	}

	return result.Allowed, nil
}

// assertSubjectReadable refuses to open a conversation about a record the
// person may not read.
func (s *Service) assertSubjectReadable(
	ctx context.Context,
	actor *services.RequestActor,
	subjectType agent.SubjectType,
) error {
	allowed, err := s.mayReadSubject(ctx, actor, subjectType)
	if err != nil {
		return err
	}
	if !allowed {
		return errortypes.NewValidationError(
			"subjectType", errortypes.ErrForbidden,
			"You do not have access to the record this conversation would be about",
		)
	}

	return nil
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
