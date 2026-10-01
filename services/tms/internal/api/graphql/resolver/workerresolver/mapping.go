package workerresolver

import (
	"github.com/emoss08/trenova/internal/core/domain/worker"
)

func ptoStatusString(value *worker.PTOStatus) string {
	if value == nil {
		return ""
	}
	return value.String()
}
