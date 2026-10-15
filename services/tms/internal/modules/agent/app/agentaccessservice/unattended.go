package agentaccessservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const unattendedRoleField = "roleId"

func (s *Service) SetUnattendedRole(
	ctx context.Context,
	req *agentdefinitionports.SetAgentUnattendedRoleRequest,
	actor *permissionports.RequestActor,
) (*agentdefinition.Definition, error) {
	if req == nil {
		return nil, errortypes.NewBusinessError("Nothing was given to save")
	}
	if err := s.authorize(
		ctx,
		actor,
		req.TenantInfo,
		permission.ResourceAgentDefinition,
		permission.ResourceRole,
	); err != nil {
		return nil, err
	}

	definition, err := s.definitions.GetByID(ctx, agentdefinitionports.GetAgentDefinitionByIDRequest{
		ID:         req.AgentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if req.RoleID.IsNotNil() {
		if _, err = s.tenantRoles(ctx, req.TenantInfo, []pulid.ID{req.RoleID}); err != nil {
			return nil, unattendedRoleError(err)
		}
		if err = s.withinActor(ctx, actor, req.RoleID); err != nil {
			return nil, err
		}
	}

	previous := definition.UnattendedRoleID
	if previous == req.RoleID {
		return definition, nil
	}

	if err = s.definitions.SetUnattendedRole(
		ctx,
		agentdefinitionports.SetAgentDefinitionUnattendedRoleRequest{
			ID:         definition.ID,
			TenantInfo: req.TenantInfo,
			RoleID:     req.RoleID,
		},
	); err != nil {
		return nil, err
	}
	definition.UnattendedRoleID = req.RoleID

	if s.notifier != nil {
		s.notifier.OrganizationChanged(ctx, req.TenantInfo)
	}
	s.logUnattendedRoleAudit(definition, previous, actor)

	return definition, nil
}

func (s *Service) withinActor(
	ctx context.Context,
	actor *permissionports.RequestActor,
	roleID pulid.ID,
) error {
	held, err := s.permissions.GetEffectivePermissions(ctx, actor.UserID, actor.OrganizationID)
	if err != nil {
		return fmt.Errorf("read the permissions of the person setting the role: %w", err)
	}

	closure, err := s.roles.GetRolesWithInheritance(ctx, []pulid.ID{roleID})
	if err != nil {
		return fmt.Errorf("read the role an agent would run as: %w", err)
	}

	for _, role := range closure {
		if role.MaxSensitivity.Level() > held.MaxSensitivity.Level() {
			return errortypes.NewValidationError(
				unattendedRoleField,
				errortypes.ErrInvalid,
				"An agent cannot run as a role that sees more sensitive fields than you can",
			)
		}
		for _, rp := range role.Permissions {
			if violation, refused := held.DelegationViolation(
				rp.Resource,
				rp.Operations,
				rp.DataScope,
			); refused {
				return violation.FieldError(unattendedRoleField)
			}
		}
	}

	return nil
}

func unattendedRoleError(err error) error {
	var multiErr *errortypes.MultiError
	if !errors.As(err, &multiErr) {
		return err
	}
	return errortypes.NewValidationError(
		unattendedRoleField,
		errortypes.ErrInvalid,
		"This role does not exist in this organization",
	)
}

func (s *Service) logUnattendedRoleAudit(
	definition *agentdefinition.Definition,
	previous pulid.ID,
	actor *permissionports.RequestActor,
) {
	auditActor := actor.AuditActor()
	if err := s.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceAgentDefinition,
		ResourceID:     definition.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		OrganizationID: definition.OrganizationID,
		BusinessUnitID: definition.BusinessUnitID,
		PreviousState:  map[string]any{"unattendedRoleId": previous.String()},
		CurrentState:   map[string]any{"unattendedRoleId": definition.UnattendedRoleID.String()},
		Critical:       true,
	}, auditservice.WithComment("Agent unattended role changed")); err != nil {
		s.l.Error("failed to log agent unattended role audit", zap.Error(err))
	}
}
