package assistantjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/delegation"
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow"
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
	call delegation.Call,
) (*agentflow.DelegateOpening, error) {
	opened, err := o.assistant.OpenDelegate(ctx, &assistantservice.OpenDelegateRequest{
		Parent:    run.Definition,
		Actor:     run.Actor,
		ThreadID:  run.ThreadID,
		StepOwner: run.StepOwner,
		Call:      call,
		Records:   run.Records,
		Inline:    run.Inline,
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
