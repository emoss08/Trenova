package shipmentautorate

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

type autoRatePlan struct {
	original        *shipment.Shipment
	entity          *shipment.Shipment
	rated           *ratequoteports.RatedShipment
	previousFreight decimal.NullDecimal
}

// validateShipmentRateNotLocked refuses to reprice a shipment whose numbers the
// customer has already been invoiced against.
func validateShipmentRateNotLocked(entity *shipment.Shipment) *errortypes.MultiError {
	if entity == nil || !entity.RateLocked {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"rateLocked",
		errortypes.ErrInvalidOperation,
		"This shipment's rate is locked and cannot be re-rated",
	)

	return multiErr
}

func contractRateCharge(charge *shipment.AdditionalCharge) shipmentserviceports.ContractRateCharge {
	out := shipmentserviceports.ContractRateCharge{
		AccessorialChargeID: charge.AccessorialChargeID,
		Owner:               charge.Owner(),
		Label:               charge.Label,
		Amount:              charge.Amount,
		Quantity:            charge.Quantity,
		Rate:                charge.Rate,
		RateUnit:            charge.RateUnit,
	}
	if definition := charge.AccessorialCharge; definition != nil {
		out.Code = definition.Code
		out.Description = definition.Description
	}
	return out
}
