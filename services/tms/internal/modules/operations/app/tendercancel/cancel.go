package tendercancel

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type CancelTenderRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	TenderID   pulid.ID              `json:"tenderId"`
	Reason     string                `json:"reason"`
}

func (r *CancelTenderRequest) Validate() error {
	multiErr := errortypes.NewMultiError()
	if r.TenderID.IsNil() {
		multiErr.Add("tenderId", errortypes.ErrRequired, "Tender is required")
	}
	if r.Reason == "" {
		multiErr.Add("reason", errortypes.ErrRequired, "Cancellation reason is required")
	}
	return multiErr.Err()
}
