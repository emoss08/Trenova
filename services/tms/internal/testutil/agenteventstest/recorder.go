package agenteventstest

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
)

type Recorder struct {
	mu     sync.Mutex
	Events []agentports.AgentEvent
}

func (r *Recorder) Publish(_ context.Context, event agentports.AgentEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Events = append(r.Events, event)
}

func (r *Recorder) Published() []agentports.AgentEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]agentports.AgentEvent, len(r.Events))
	copy(out, r.Events)

	return out
}
