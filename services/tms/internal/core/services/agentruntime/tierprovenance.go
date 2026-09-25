package agentruntime

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func (s *Service) decideCall(
	ctx context.Context,
	in agenttoolpolicy.DecideInput,
) agenttoolpolicy.Decision {
	in.TierSetByPerson = s.tierSetByPerson(ctx, in.Definition, in.Policy.Name)

	return agenttoolpolicy.Decide(ctx, in)
}

func (s *Service) tierSetByPerson(
	ctx context.Context,
	definition *agentdefinition.Definition,
	toolName string,
) bool {
	if definition == nil || !definition.SetsToolTier(toolName) {
		return false
	}

	trust, err := s.toolTrust(ctx, definition, toolName)
	if err != nil {
		s.logger.Warn("could not read whether trust set a tool's tier; treating it as a person's",
			zap.String("agent", definition.ID.String()),
			zap.String("tool", toolName),
			zap.Error(err),
		)

		return true
	}

	return agenttoolpolicy.TierSetByPerson(definition, toolName, trust)
}

func (s *Service) toolTrust(
	ctx context.Context,
	definition *agentdefinition.Definition,
	toolName string,
) (*agent.ToolTrust, error) {
	if s.trust == nil || definition.ID.IsNil() || definition.OrganizationID.IsNil() ||
		definition.BusinessUnitID.IsNil() {
		return new(agent.ToolTrust), nil
	}

	rows, err := s.trust.ListByDefinition(ctx, repositories.ListToolTrustRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: definition.OrganizationID,
			BuID:  definition.BusinessUnitID,
		},
		AgentDefinitionID: definition.ID,
	})
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		if row != nil && row.ToolName == toolName {
			return row, nil
		}
	}

	return new(agent.ToolTrust), nil
}
