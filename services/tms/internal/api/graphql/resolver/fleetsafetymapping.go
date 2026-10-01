package resolver

import (
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// violationWeight keeps the FMCSA's own range at the edge of the API, so a
// weight nobody could read against the published tables never reaches the
// domain.
func violationWeight(value *int) (int16, error) {
	if value == nil {
		return 0, nil
	}
	if *value < 1 || *value > 10 {
		return 0, errortypes.NewValidationError(
			"severityWeight",
			errortypes.ErrInvalid,
			"Severity weight is 1 to 10",
		)
	}
	return int16(*value), nil
}

// basicValue leaves an unstated BASIC empty so the service can fall back to
// what the event itself implies.
func basicValue(value *worker.CSABasic) worker.CSABasic {
	if value == nil {
		return ""
	}
	return *value
}
