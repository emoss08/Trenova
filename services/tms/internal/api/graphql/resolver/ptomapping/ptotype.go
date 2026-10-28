package ptomapping

import (
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
)

func PtoTypeString(value *worker.PTOType) string {
	if value == nil {
		return ""
	}
	return value.String()
}
