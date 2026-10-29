package tendernegotiation

import (
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/tendereffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type counterAnnouncement struct {
	offer    *tender.TenderOffer
	rate     decimal.Decimal
	note     string
	source   tender.ResponseSource
	verdict  tender.CounterVerdict
	deadline int64
}

// DecideCounterRequest takes or refuses a carrier's counter-offer.
type DecideCounterRequest struct {
	TenantInfo pagination.TenantInfo         `json:"-"`
	Actor      *permissionports.RequestActor `json:"-"`
	OfferID    pulid.ID                      `json:"offerId"`
	Approve    bool                          `json:"approve"`
	Reason     string                        `json:"reason"`
}

func (r *DecideCounterRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if r.OfferID.IsNil() {
		multiErr.Add("offerId", errortypes.ErrRequired, "Offer is required")
	}
	if utf8.RuneCountInString(r.Reason) > MaxDeclineReasonLength {
		multiErr.Add("reason", errortypes.ErrInvalid, "Reason must be at most 500 characters")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (r *DecideCounterRequest) actor() *permissionports.RequestActor {
	if r.Actor != nil {
		return r.Actor
	}
	return permissionports.UserActor(r.TenantInfo)
}

func DecisionStamp(tenantInfo pagination.TenantInfo, at int64) *tenderports.DecisionStamp {
	return &tenderports.DecisionStamp{At: at, UserID: tendereffects.UserIDPtr(tenantInfo)}
}
