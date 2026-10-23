package agentquerytoolservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
)

// workerName is how a person names a driver. Every tool that returns a worker
// row needs it, and a driver whose record carries only one of the two names
// should still read as a name rather than a stray space.
func workerName(w *worker.Worker) string {
	if w == nil {
		return ""
	}

	return strings.TrimSpace(w.FirstName + " " + w.LastName)
}
