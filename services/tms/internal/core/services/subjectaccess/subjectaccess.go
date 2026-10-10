package subjectaccess

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// MayRead reports whether the person may read the record a conversation is
// about. The subject is described into the prompt on every turn, so a
// conversation pinned to a record the person cannot open would read it to
// them through the model. A kind of subject that names no record needs no
// grant; a permission that cannot be checked is a refusal.
func MayRead(
	ctx context.Context,
	engine services.PermissionEngine,
	actor *services.RequestActor,
	subjectType agent.SubjectType,
) (bool, error) {
	resource, names := subjectType.Resource()
	if !names {
		return true, nil
	}

	allowed, err := MayReadResource(ctx, engine, actor, resource)
	if err != nil {
		return false, fmt.Errorf("check access to the conversation's subject: %w", err)
	}

	return allowed, nil
}

func MayReadResource(
	ctx context.Context,
	engine services.PermissionEngine,
	actor *services.RequestActor,
	resource permission.Resource,
) (bool, error) {
	if engine == nil || actor == nil {
		return false, nil
	}

	result, err := engine.Check(ctx, &services.PermissionCheckRequest{
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
		return false, err
	}

	return result.Allowed, nil
}

// AssertReadable refuses a conversation about a record the person may not
// read.
func AssertReadable(
	ctx context.Context,
	engine services.PermissionEngine,
	actor *services.RequestActor,
	subjectType agent.SubjectType,
) error {
	allowed, err := MayRead(ctx, engine, actor, subjectType)
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
