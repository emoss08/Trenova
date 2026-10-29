package tendereffects

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

// IsAssignBusinessFailure separates deterministic assignment rejections
// (eligibility, coverage, validation — park for review) from transient
// failures (deadlocks, serialization conflicts mapped to ConflictError, raw
// infra errors — let the activity retry).
func IsAssignBusinessFailure(err error) bool {
	if errortypes.IsConflictError(err) {
		return false
	}
	return errortypes.IsBusinessError(err) || errortypes.IsError(err)
}
