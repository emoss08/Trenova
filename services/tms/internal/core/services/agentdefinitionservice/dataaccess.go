package agentdefinitionservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const dataAccessField = "dataAccessCeiling"

func (s *Service) checkDataAccess(
	ctx context.Context,
	definition, previous *agentdefinition.Definition,
	actor *services.RequestActor,
) error {
	wanted := definition.DataAccessSensitivity()
	if wanted == permission.SensitivityInternal {
		return nil
	}
	if previous != nil && wanted.Level() <= previous.DataAccessSensitivity().Level() {
		return nil
	}

	granted, err := s.actorSensitivity(ctx, actor)
	if err != nil {
		return err
	}
	if granted.CanAccess(wanted) {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		dataAccessField,
		errortypes.ErrInvalid,
		"You can give an agent only the data access your own role has",
	)

	return multiErr
}

func (s *Service) actorSensitivity(
	ctx context.Context,
	actor *services.RequestActor,
) (permission.FieldSensitivity, error) {
	if actor == nil || !actor.IsUser() || s.permissions == nil {
		return permission.SensitivityInternal, nil
	}

	detail, err := s.permissions.GetResourcePermissions(
		ctx,
		actor.UserID,
		actor.OrganizationID,
		permission.ResourceAgentDefinition.String(),
	)
	if err != nil {
		return "", fmt.Errorf("read the data access of the person saving the agent: %w", err)
	}
	if detail == nil {
		return permission.SensitivityInternal, nil
	}

	return detail.MaxSensitivity, nil
}
