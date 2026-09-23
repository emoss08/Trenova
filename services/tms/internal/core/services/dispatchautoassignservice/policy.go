package dispatchautoassignservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const systemKeyDispatchAssignment = "dispatch_assignment"

type Policy struct {
	DefinitionID pulid.ID
	Enabled      bool
	ShadowMode   bool
	Tier         agent.AutonomyTier
}

func policyFor(definition *agentdefinition.Definition, organizationShadow bool) Policy {
	if definition == nil {
		return Policy{ShadowMode: organizationShadow, Tier: agent.TierPropose}
	}
	if !definition.Enabled {
		return Policy{
			DefinitionID: definition.ID,
			ShadowMode:   definition.EffectiveShadow(organizationShadow),
			Tier:         agent.TierPropose,
		}
	}

	return Policy{
		DefinitionID: definition.ID,
		Enabled:      true,
		ShadowMode:   definition.EffectiveShadow(organizationShadow),
		Tier:         definition.EffectiveTier(toolNameAssignMove, agent.TierPropose),
	}
}

func (s *Service) loadPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (Policy, error) {
	control, err := s.agentControlRepo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return Policy{}, err
	}

	definition, err := s.definitionRepo.GetBySystemKey(
		ctx,
		repositories.GetAgentDefinitionBySystemKeyRequest{
			SystemKey:  systemKeyDispatchAssignment,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return policyFor(nil, control.ShadowMode), nil
		}

		return Policy{}, err
	}

	return policyFor(definition, control.ShadowMode), nil
}
