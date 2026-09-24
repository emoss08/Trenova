package assistantjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/assistantservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
)

// delegateOpener opens the turn of an agent a conversation's agent hands a
// task to, through the assistant, which owns what a conversation may do.
type delegateOpener struct {
	assistant *assistantservice.Service
}

func newDelegateOpener(assistant *assistantservice.Service) agentflow.DelegateOpener {
	return &delegateOpener{assistant: assistant}
}

func (o *delegateOpener) OpenDelegate(
	ctx context.Context,
	run agentflow.RunContext,
	call agentruntime.DelegateCall,
) (*agentflow.DelegateOpening, error) {
	opened, err := o.assistant.OpenDelegate(ctx, &assistantservice.OpenDelegateRequest{
		Parent:    run.Definition,
		Actor:     run.Actor,
		ThreadID:  run.ThreadID,
		StepOwner: run.StepOwner,
		Call:      call,
		Records:   run.Records,
	})
	if err != nil {
		if refusal, ok := assistantservice.IsDelegateDeclined(err); ok {
			return nil, &agentflow.DelegateDeclinedError{Reason: refusal.Reason}
		}

		return nil, err
	}

	sub := agentflow.NewRunContext(opened.Request, run.PriorityKey)
	if sub.Timezone == "" {
		sub.Timezone = run.Timezone
	}

	return &agentflow.DelegateOpening{Run: sub, Turn: opened.Turn}, nil
}
