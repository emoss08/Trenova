package agentruntime

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemoryprompt"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"go.uber.org/zap"
)

func (s *Service) memoriesForPrompt(
	ctx context.Context,
	req *agentruntimeports.RunRequest,
	rc *agentturn.RuntimeContext,
) agentmemoryprompt.MemoryFit {
	if !req.Definition.HasContextProvider(agentdefinition.ContextMemory) {
		return agentmemoryprompt.MemoryFit{}
	}

	fit := agentmemoryprompt.PlanMemories(req.Definition, rc)
	if len(fit.Used) == 0 || s.memories == nil || req.Actor == nil {
		return fit
	}

	if err := s.memories.RecordUse(ctx, agentmemoryports.RecordMemoryUseRequest{
		TenantInfo: req.Actor.TenantInfo(),
		IDs:        fit.Used,
	}); err != nil {
		s.logger.Warn("agent memory: could not count the memories a turn used",
			zap.Error(err),
		)
	}

	return fit
}
