package agentruntime

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemoryprompt"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) memoriesForPrompt(
	req *agentruntimeports.RunRequest,
	rc *agentturn.RuntimeContext,
) agentmemoryprompt.MemoryFit {
	if !req.Definition.HasContextProvider(agentdefinition.ContextMemory) {
		return agentmemoryprompt.MemoryFit{}
	}

	return agentmemoryprompt.PlanMemories(req.Definition, rc)
}

func (s *Service) RecordMemoryUse(
	ctx context.Context,
	tenant pagination.TenantInfo,
	result *agentruntimeports.RunResult,
) {
	ids := UsedMemoryIDs(result)
	if len(ids) == 0 {
		return
	}

	if err := s.memories.RecordUse(ctx, agentmemoryports.RecordMemoryUseRequest{
		TenantInfo: tenant,
		IDs:        ids,
	}); err != nil {
		s.logger.Warn("agent memory: could not count the memories a turn used", zap.Error(err))
	}
}

func UsedMemoryIDs(result *agentruntimeports.RunResult) []pulid.ID {
	if result == nil {
		return nil
	}

	ids := slices.Clone(result.UsedMemoryIDs)
	for _, delegation := range result.Delegations {
		for _, id := range delegation.UsedMemoryIDs {
			if id.IsNotNil() && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}

	return ids
}
