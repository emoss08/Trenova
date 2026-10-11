package common

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
)

const FieldCarrierID = "carrierId"

// rateConfirmationChange is one revision before and after, limited to the
// fields the write moves; a time the write stamps is shown but left out of
// the digest.
func RateConfirmationChange(
	before, after *rateconfirmation.RateConfirmation,
	shipmentLabel string,
	fields ...string,
) (*agent.RecordChange, error) {
	return toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceRateConfirmation,
		ID:       before.ID,
		Label:    RateConfirmationLabel(before, shipmentLabel),
		Version:  PreviewVersion(before.Version),
	}, before, after,
		toolpreview.Only(fields...),
		toolpreview.Volatile("voidedAt", "sentAt", "confirmedAt"),
		toolpreview.Labels(RateConfirmationLabels),
	)
}

func RateConfirmationLabel(entity *rateconfirmation.RateConfirmation, shipmentLabel string) string {
	label := fmt.Sprintf("Rate confirmation revision %d", entity.Revision)
	if shipmentLabel != "" {
		label += " for " + shipmentLabel
	}

	return label
}

// carrierPay is the pay an assignment commits, line by line as the carrier's
// settlement reads it.
func CarrierPay(assignment *shipment.CarrierAssignment) *agent.MoneyPreview {
	lines := make([]agent.MoneyLine, 0, len(assignment.Accessorials)+2)
	lines = append(lines, agent.MoneyLine{
		Label: "Linehaul (" + RateMethodWords(assignment.RateMethod) + ")",
		After: KnownAmount(assignment.BaseAmount),
	})
	if !assignment.FuelSurcharge.IsZero() {
		lines = append(lines, agent.MoneyLine{
			Label: "Fuel surcharge",
			After: KnownAmount(assignment.FuelSurcharge),
		})
	}
	for _, accessorial := range assignment.Accessorials {
		if accessorial == nil {
			continue
		}
		lines = append(lines, agent.MoneyLine{
			Label: accessorial.Description,
			After: KnownAmount(accessorial.Amount),
		})
	}

	return toolpreview.MoneyBlock(assignment.CurrencyCode, lines...)
}

func CarrierName(entity *carrier.Carrier) string {
	if entity == nil || strings.TrimSpace(entity.Name) == "" {
		return "the carrier"
	}

	return strings.TrimSpace(entity.Name)
}
