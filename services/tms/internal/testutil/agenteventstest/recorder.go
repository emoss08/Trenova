package agenteventstest

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/core/ports/services"
)

type Recorder struct {
	mu     sync.Mutex
	Events []services.AgentEvent
}

func (r *Recorder) Publish(_ context.Context, event services.AgentEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Events = append(r.Events, event)
}

func (r *Recorder) Published() []services.AgentEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]services.AgentEvent, len(r.Events))
	copy(out, r.Events)

	return out
}
