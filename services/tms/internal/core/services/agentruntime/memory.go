package agentruntime

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) memoriesForPrompt(
	ctx context.Context,
	req *serviceports.RunRequest,
	rc agentdefinition.RuntimeContext,
) []*agent.Memory {
	if !req.Definition.HasContextProvider(agentdefinition.ContextMemory) {
		return nil
	}

	fitted := req.Definition.FitMemories(rc)
	if len(fitted) == 0 || s.memories == nil || req.Actor == nil {
		return fitted
	}

	ids := make([]pulid.ID, 0, len(fitted))
	for _, memory := range fitted {
		if memory.ID.IsNotNil() {
			ids = append(ids, memory.ID)
		}
	}
	if err := s.memories.RecordUse(ctx, serviceports.RecordMemoryUseRequest{
		TenantInfo: req.Actor.TenantInfo(),
		IDs:        ids,
	}); err != nil {
		s.logger.Warn("agent memory: could not count the memories a prompt carried",
			zap.Error(err),
		)
	}

	return fitted
}
