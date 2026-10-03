package agentruntime

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) memoriesForPrompt(
	ctx context.Context,
	req *serviceports.RunRequest,
	rc *agentdefinition.RuntimeContext,
) []*agent.Memory {
	if !req.Definition.HasContextProvider(agentdefinition.ContextMemory) {
		return nil
	}

	fitted := req.Definition.FitMemories(rc)
	if len(fitted) == 0 || s.memories == nil || req.Actor == nil {
		return fitted
	}

	if err := s.memories.RecordUse(ctx, serviceports.RecordMemoryUseRequest{
		TenantInfo: req.Actor.TenantInfo(),
		IDs:        memoryIDs(fitted),
	}); err != nil {
		s.logger.Warn("agent memory: could not count the memories a prompt carried",
			zap.Error(err),
		)
	}

	return fitted
}

func memoryIDs(memories []*agent.Memory) []pulid.ID {
	if len(memories) == 0 {
		return nil
	}

	ids := make([]pulid.ID, 0, len(memories))
	for _, memory := range memories {
		if memory != nil && memory.ID.IsNotNil() {
			ids = append(ids, memory.ID)
		}
	}

	return ids
}

// announceMemories tells the reader which memories the prompt carried, before
// the first word of the reply, so the note that says so sits above it.
func (t *Turn) announceMemories(fx TurnEffects) {
	if len(t.result.UsedMemoryIDs) == 0 {
		return
	}

	fx.Emit(memoryUsedEvent(t.result.UsedMemoryIDs))
}

// noteMemories keeps what a call did with memory: the memories a recall read
// back join those the turn used, and a memory a remember kept or offered is
// announced so the reply can show it.
func (t *Turn) noteMemories(fx TurnEffects, outcome *toolOutcome) {
	if t.req.Delegation != nil {
		return
	}

	added := false
	for _, id := range outcome.memories {
		if id.IsNotNil() && !slices.Contains(t.result.UsedMemoryIDs, id) {
			t.result.UsedMemoryIDs = append(t.result.UsedMemoryIDs, id)
			added = true
		}
	}
	if added {
		fx.Emit(memoryUsedEvent(t.result.UsedMemoryIDs))
	}

	if saved := outcome.saved; saved != nil && saved.ID.IsNotNil() {
		t.result.SavedMemories = append(t.result.SavedMemories, *saved)
		fx.Emit(serviceports.StreamEvent{
			Event: serviceports.AssistantEventMemorySaved,
			Data:  *saved,
		})
	}
}

func memoryUsedEvent(ids []pulid.ID) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventMemoryUsed,
		Data:  serviceports.AssistantMemoryUsedEvent{IDs: slices.Clone(ids)},
	}
}
