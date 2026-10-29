package agenteventstest

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
)

type Recorder struct {
	mu     sync.Mutex
	Events []agenteventports.AgentEvent
}

func (r *Recorder) Publish(_ context.Context, event agenteventports.AgentEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Events = append(r.Events, event)
}

func (r *Recorder) Published() []agenteventports.AgentEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]agenteventports.AgentEvent, len(r.Events))
	copy(out, r.Events)

	return out
}
