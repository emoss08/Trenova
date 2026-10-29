package turnmemory

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

// Turn is what keeping a turn's memories reads of it: what it was
// asked, what it has done and the memories about tools it has not called.
type Turn interface {
	Request() *agentruntimeports.RunRequest
	Result() *agentruntimeports.RunResult
	ToolMemories() map[string][]pulid.ID
}

// Emitter is where a turn's events go.
type Emitter interface {
	Emit(event kernelports.StreamEvent)
}
