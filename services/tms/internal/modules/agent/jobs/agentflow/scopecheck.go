package agentflow

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentguard"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.temporal.io/sdk/temporal"
)

// scopeCheck is the scope classifier running beside a model call.
//
// The classifier used to stand in front of the turn, and every question waited
// on its verdict before a model was asked anything. Here the model is asked at
// once and the two run together: what the model streams is held until the
// verdict, and the call's reply reaches the loop only after it, so nothing the
// model wrote is shown and no tool it asked for runs on a question the guard
// refuses. A refusal cancels the model call where it stands.
type scopeCheck struct {
	gate    *streamGate
	verdict chan agentguard.Decision
	cancel  context.CancelFunc
}

// checkScope starts the classifier on req and returns the context the model
// call runs on and the emitter it streams to. A nil req checks nothing and
// returns what it was given.
func (a *Activities) checkScope(
	ctx context.Context,
	req *agentguard.EvaluateRequest,
	emit kernelports.AssistantStreamEmitter,
) (context.Context, kernelports.AssistantStreamEmitter, *scopeCheck) {
	if req == nil {
		return ctx, emit, nil
	}

	callCtx, cancel := context.WithCancel(ctx)
	check := &scopeCheck{
		gate:    &streamGate{emit: emit},
		verdict: make(chan agentguard.Decision, 1),
		cancel:  cancel,
	}
	go func() {
		decision := a.guard.Evaluate(ctx, *req)
		check.gate.settle(decision.Allowed)
		if !decision.Allowed {
			cancel()
		}
		check.verdict <- decision
	}()

	return callCtx, check.gate.Emit, check
}

// wait is the verdict, once the classifier has given it. The classifier
// bounds itself, so this never outlasts its timeout.
func (c *scopeCheck) wait() agentguard.Decision {
	return <-c.verdict
}

func scopeRefused(decision *agentguard.Decision) error {
	return temporal.NewNonRetryableApplicationError(
		decision.Message, ErrTypeScopeRefused, nil, *decision,
	)
}

// streamGate holds a model call's stream until the scope verdict: released on
// an allowed question, in the order it arrived, and dropped on a refused one.
type streamGate struct {
	mu      sync.Mutex
	emit    kernelports.AssistantStreamEmitter
	held    []kernelports.StreamEvent
	open    bool
	refused bool
}

func (g *streamGate) Emit(event kernelports.StreamEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()

	switch {
	case g.refused:
	case g.open:
		g.emit(event)
	default:
		g.held = append(g.held, event)
	}
}

func (g *streamGate) settle(allowed bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if allowed {
		for idx := range g.held {
			g.emit(g.held[idx])
		}
		g.open = true
	} else {
		g.refused = true
	}
	g.held = nil
}
