package runtimetest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/pagination"
)

// KeyedLedger answers by key, the way the real ledger does: a key it has
// settled is replayed, and any other key is fresh. The fixed-verdict stub
// above answers the same for every key, which is how a retry that minted a
// new key for an old write passed.
type KeyedLedger struct {
	Steps map[string]agentruntimeports.RunStep
}

func NewKeyedLedger() *KeyedLedger {
	return &KeyedLedger{Steps: map[string]agentruntimeports.RunStep{}}
}

func (l *KeyedLedger) Claim(
	_ context.Context,
	_ pagination.TenantInfo,
	step *agentruntimeports.RunStep,
) (agentruntimeports.StepVerdict, error) {
	recorded, seen := l.Steps[step.Key]
	switch {
	case !seen:
		l.Steps[step.Key] = *step
		return agentruntimeports.StepVerdict{State: agentruntimeports.StepFresh}, nil
	case recorded.Status == agentruntimeports.RunStepCompleted:
		return agentruntimeports.StepVerdict{
			State:   agentruntimeports.StepCompleted,
			Outcome: recorded.Outcome,
		}, nil
	case recorded.Status == agentruntimeports.RunStepFailed:
		return agentruntimeports.StepVerdict{
			State:   agentruntimeports.StepFailed,
			Outcome: recorded.Outcome,
		}, nil
	default:
		return agentruntimeports.StepVerdict{State: agentruntimeports.StepUnknown}, nil
	}
}

func (l *KeyedLedger) Settle(
	_ context.Context,
	_ pagination.TenantInfo,
	step *agentruntimeports.RunStep,
) error {
	l.Steps[step.Key] = *step

	return nil
}

func (l *KeyedLedger) Record(
	context.Context,
	pagination.TenantInfo,
	*agentruntimeports.RunStep,
) error {
	return nil
}

func (l *KeyedLedger) Find(
	_ context.Context,
	_ pagination.TenantInfo,
	_ agentruntimeports.RunStepOwner,
	key string,
) (agentruntimeports.RunStep, bool, error) {
	step, found := l.Steps[key]

	return step, found, nil
}

func (l *KeyedLedger) Loaded(
	context.Context,
	pagination.TenantInfo,
	agentruntimeports.RunStepOwner,
) ([]agentruntimeports.RunStep, error) {
	steps := make([]agentruntimeports.RunStep, 0, len(l.Steps))
	for _, step := range l.Steps {
		steps = append(steps, step)
	}

	return steps, nil
}
