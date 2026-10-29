package tenderresponse

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

// invalidTokenError uses one piece of copy for every failure mode — missing,
// expired, used, revoked — so the public endpoint leaks nothing about which
// one applied.
func invalidTokenError() error {
	return errortypes.NewValidationError(
		"token",
		errortypes.ErrInvalid,
		"This offer link is no longer valid",
	)
}

// PublicOfferView is everything a carrier without a login may see about an
// offer: the lane, the timing, and the money — never customer data.
type PublicOfferView struct {
	CarrierName        string `json:"carrierName"`
	ShipmentProNumber  string `json:"shipmentProNumber"`
	OriginSummary      string `json:"originSummary"`
	DestinationSummary string `json:"destinationSummary"`
	PickupWindow       string `json:"pickupWindow"`
	DeliveryWindow     string `json:"deliveryWindow"`
	EquipmentSummary   string `json:"equipmentSummary"`
	WeightSummary      string `json:"weightSummary"`
	RateAmount         string `json:"rateAmount"`
	RateMethodLabel    string `json:"rateMethodLabel"`
	ExpiresAt          int64  `json:"expiresAt"`
	Responded          bool   `json:"responded"`
	// RateValue is the offered rate as a plain number, for a counter-offer
	// form to start from.
	RateValue string `json:"rateValue"`
	// CounterAllowed says whether the carrier may answer with a rate of its
	// own instead of accepting or declining.
	CounterAllowed bool `json:"counterAllowed"`
}

// PublicResponse is a carrier's answer from the public page.
type PublicResponse struct {
	Action        tender.ResponseAction
	DeclineReason string
	CounterRate   decimal.Decimal
	CounterNote   string
}
