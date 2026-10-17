package inboundmessageservice

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

func (s *Service) planTransition(ctx context.Context, t *transition) (*QueuePlan, error) {
	plan := &QueuePlan{}
	if len(t.ids) == 0 {
		return plan, nil
	}

	states, err := s.queueRepo.QueueStates(ctx, inboundmessageports.QueueStatesRequest{
		TenantInfo: t.tenant,
		IDs:        t.ids,
	})
	if err != nil {
		return nil, err
	}
	if t.check != nil {
		if err = t.check(states); err != nil {
			return nil, err
		}
	}

	patch := t.patch
	if t.patchFor != nil {
		patch = t.patchFor(states)
	}

	plan.Changes = make([]QueueChange, 0, len(states))
	for _, state := range states {
		if t.eligible != nil && !t.eligible(state) {
			plan.Unchanged++
			continue
		}
		after := *state
		patch.ApplyTo(&after)
		plan.Changes = append(plan.Changes, QueueChange{Before: state, After: &after})
	}
	plan.Missing = max(len(t.ids)-len(states), 0)

	return plan, nil
}
