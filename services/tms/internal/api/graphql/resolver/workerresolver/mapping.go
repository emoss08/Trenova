package workerresolver

import (
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
)

func ptoStatusString(value *worker.PTOStatus) string {
	if value == nil {
		return ""
	}
	return value.String()
}
