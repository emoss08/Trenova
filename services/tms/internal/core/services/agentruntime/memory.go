package agentruntime

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) memoriesForPrompt(
	ctx context.Context,
	req *serviceports.RunRequest,
	rc *agentdefinition.RuntimeContext,
) agentdefinition.MemoryFit {
	if !req.Definition.HasContextProvider(agentdefinition.ContextMemory) {
		return agentdefinition.MemoryFit{}
	}

	fit := req.Definition.PlanMemories(rc)
	if len(fit.Used) == 0 || s.memories == nil || req.Actor == nil {
		return fit
	}

	if err := s.memories.RecordUse(ctx, serviceports.RecordMemoryUseRequest{
		TenantInfo: req.Actor.TenantInfo(),
		IDs:        fit.Used,
	}); err != nil {
		s.logger.Warn("agent memory: could not count the memories a turn used",
			zap.Error(err),
		)
	}

	return fit
}

// announceMemories tells the reader which memories bore on the turn, before
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

// noteToolMemories counts the memories about a tool as used once the turn
// calls it: they were carried because the tool was on hand, and bear on the
// reply only when it is.
func (t *Turn) noteToolMemories(fx TurnEffects, tool string) {
	if t.req.Delegation != nil || len(t.toolMemories) == 0 {
		return
	}
	ids, ok := t.toolMemories[tool]
	if !ok {
		return
	}
	delete(t.toolMemories, tool)

	added := false
	for _, id := range ids {
		if id.IsNotNil() && !slices.Contains(t.result.UsedMemoryIDs, id) {
			t.result.UsedMemoryIDs = append(t.result.UsedMemoryIDs, id)
			added = true
		}
	}
	if added {
		fx.Emit(memoryUsedEvent(t.result.UsedMemoryIDs))
	}
}

func memoryUsedEvent(ids []pulid.ID) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventMemoryUsed,
		Data:  serviceports.AssistantMemoryUsedEvent{IDs: slices.Clone(ids)},
	}
}
