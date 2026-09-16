package billingqueueservice

import (
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// convertAmountSplitsField is the field a stale amount split is reported on. The
// client reads it as the cue to offer converting the split to percentages and
// retry with ConvertAmountSplitsToPercent set.
const convertAmountSplitsField = "convertAmountSplitsToPercent"

func accessorialNames(charges []*shipment.AdditionalCharge) map[pulid.ID]string {
	names := make(map[pulid.ID]string, len(charges))
	for _, charge := range charges {
		if charge == nil || charge.AccessorialCharge == nil {
			continue
		}
		name := charge.AccessorialCharge.Description
		if name == "" {
			name = charge.AccessorialCharge.Code
		}
		names[charge.ID] = name
	}

	return names
}

// staleAmountSplitError refuses a charge edit that would leave an amount split
// that no longer adds up. Each stale charge is named with what its split totals
// and what the charge is now.
func staleAmountSplitError(
	stale []shipment.StaleAmountSplit,
	names map[pulid.ID]string,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	for _, split := range stale {
		charge := "freight"
		if split.Kind == shipment.ChargeAllocationKindAccessorial {
			charge = names[split.AdditionalChargeID]
			if charge == "" {
				charge = "accessorial charge " + strconv.Itoa(split.ChargeIndex+1)
			}
		}
		multiErr.Add(
			convertAmountSplitsField,
			errortypes.ErrInvalidOperation,
			"The {0} split no longer adds up: the payer amounts total {1} but the charge is now {2}. Convert the split to percentages to keep each payer's proportion, or change the split on the shipment.",
			charge,
			split.AllocatedTotal.StringFixed(shipment.SharePlaces),
			split.ChargeTotal.StringFixed(shipment.SharePlaces),
		)
	}

	return multiErr
}

func chargesUpdatedComment(convertedSplits int) string {
	if convertedSplits == 0 {
		return "Charges updated from billing queue"
	}

	return "Charges updated from billing queue; " + strconv.Itoa(convertedSplits) +
		" amount split(s) converted to percentages"
}
