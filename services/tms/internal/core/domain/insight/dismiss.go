package insight

import (
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func (i *Insight) Dismiss(userID pulid.ID, reason string, now int64) error {
	if i.Status != StatusActive {
		return errortypes.NewBusinessError(
			"This insight is {0}, and only an active one can be dismissed",
			strings.ToLower(string(i.Status)),
		)
	}

	i.Status = StatusDismissed
	i.DismissedAt = &now
	i.DismissedByID = userID
	i.DismissReason = reason

	return nil
}
