package inboundqueue

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
)

type QueueChange struct {
	Before *inboundmessageports.QueueState
	After  *inboundmessageports.QueueState
}

type QueuePlan struct {
	Changes   []QueueChange
	Unchanged int
	Missing   int
	More      bool
}

func (s *Service) planTransition(ctx context.Context, t *Transition) (*QueuePlan, error) {
	plan := &QueuePlan{}
	if len(t.IDs) == 0 {
		return plan, nil
	}

	states, err := s.queueRepo.QueueStates(ctx, inboundmessageports.QueueStatesRequest{
		TenantInfo: t.Tenant,
		IDs:        t.IDs,
	})
	if err != nil {
		return nil, err
	}
	if t.Check != nil {
		if err = t.Check(states); err != nil {
			return nil, err
		}
	}

	patch := t.Patch
	if t.PatchFor != nil {
		patch = t.PatchFor(states)
	}

	plan.Changes = make([]QueueChange, 0, len(states))
	for _, state := range states {
		if t.Eligible != nil && !t.Eligible(state) {
			plan.Unchanged++
			continue
		}
		after := *state
		patch.ApplyTo(&after)
		plan.Changes = append(plan.Changes, QueueChange{Before: state, After: &after})
	}
	plan.Missing = max(len(t.IDs)-len(states), 0)

	return plan, nil
}
