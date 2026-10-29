package modelstream

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/toolaccess"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
)

// Streamer asks the model for a turn's next step, streaming the reply as it
// arrives and stopping one that falls into a loop.
type Streamer struct {
	completion llmports.CompletionService
	access     *toolaccess.Access
}

func NewStreamer(completion llmports.CompletionService, access *toolaccess.Access) *Streamer {
	return &Streamer{completion: completion, access: access}
}
