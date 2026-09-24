package agentdefinitionservice

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const maxSystemNameAttempts = 5

func (s *Service) EnsureSystem(
	ctx context.Context,
	req services.EnsureSystemAgentRequest,
) (*agentdefinition.Definition, error) {
	template, ok := agentdefinition.PageAgentTemplate(req.SystemKey)
	if !ok {
		return nil, fmt.Errorf("%q is not an agent Trenova creates on first use", req.SystemKey)
	}
	if req.TenantInfo.OrgID.IsNil() || req.TenantInfo.BuID.IsNil() {
		return nil, fmt.Errorf("system agent %s needs an organization", req.SystemKey)
	}

	existing, err := s.systemAgent(ctx, req)
	if err != nil || existing != nil {
		return existing, err
	}

	for attempt := range maxSystemNameAttempts {
		definition, _ := agentdefinition.NewPageAgent(
			req.SystemKey,
			req.TenantInfo,
			systemAgentName(template, attempt),
		)

		multiErr := errortypes.NewMultiError()
		definition.Validate(multiErr)
		if multiErr.HasErrors() {
			return nil, fmt.Errorf("system agent %s is not valid: %w", req.SystemKey, multiErr)
		}

		created, cErr := s.repo.CreateSystem(ctx, definition)
		if cErr != nil {
			return nil, cErr
		}

		found, gErr := s.systemAgent(ctx, req)
		if gErr != nil {
			return nil, gErr
		}
		if found != nil {
			if created && found.ID == definition.ID {
				s.logAudit(found, nil, permission.OpCreate, systemActor(req.TenantInfo),
					"System agent created on first use")
			}

			return found, nil
		}
	}

	return nil, errortypes.NewBusinessError(
		"{0} could not be set up because other agents already use its name. Rename the agent "+
			"called {0} in AI Control and try again.",
		template.Label(),
	)
}

func (s *Service) systemAgent(
	ctx context.Context,
	req services.EnsureSystemAgentRequest,
) (*agentdefinition.Definition, error) {
	found, err := s.repo.GetBySystemKey(ctx, repositories.GetAgentDefinitionBySystemKeyRequest{
		SystemKey: req.SystemKey,
		TenantInfo: pagination.TenantInfo{
			OrgID: req.TenantInfo.OrgID,
			BuID:  req.TenantInfo.BuID,
		},
	})
	if err == nil {
		return found, nil
	}
	if errortypes.IsNotFoundError(err) {
		return nil, nil
	}

	return nil, fmt.Errorf("read system agent %s: %w", req.SystemKey, err)
}

func systemAgentName(template agentdefinition.Template, attempt int) string {
	switch attempt {
	case 0:
		return template.Label()
	case 1:
		return template.Label() + " (built-in)"
	default:
		return template.Label() + " (built-in " + strconv.Itoa(attempt) + ")"
	}
}

func systemActor(tenant pagination.TenantInfo) *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    tenant.UserID,
		UserID:         tenant.UserID,
		BusinessUnitID: tenant.BuID,
		OrganizationID: tenant.OrgID,
	}
}
