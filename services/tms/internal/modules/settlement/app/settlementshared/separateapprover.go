package settlementshared

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func RequireSeparateApprover(required bool, submittedBy, approver pulid.ID) error {
	if !required || submittedBy != approver {
		return nil
	}

	return errortypes.NewValidationError(
		"settlementId",
		errortypes.ErrInvalidOperation,
		"You submitted this settlement, so someone else must approve it",
	)
}
