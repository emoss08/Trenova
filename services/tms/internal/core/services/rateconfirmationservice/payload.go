package rateconfirmationservice

import (
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

const windowTimeLayout = "Mon 02 Jan 2006 15:04 MST"

type payloadParams struct {
	Shipment    *shipment.Shipment
	Move        *shipment.ShipmentMove
	Assignment  *shipment.CarrierAssignment
	Carrier     *carrier.Carrier
	Revision    int64
	CompanyName string
}

// buildContext assembles the render context for a rate confirmation. It is pure
// so a generated document can be reproduced from its inputs in a test, and the
// same context is frozen as the payload snapshot the send path re-renders from.
func buildContext(p payloadParams) *documenttemplate.RateConfirmationContext {
	out := &documenttemplate.RateConfirmationContext{
		CompanyName:           p.CompanyName,
		CarrierName:           p.Carrier.Name,
		CarrierSCAC:           p.Carrier.SCAC,
		CarrierMC:             p.Carrier.MCNumber,
		CarrierDOT:            p.Carrier.DOTNumber,
		ShipmentProNumber:     p.Shipment.ProNumber,
		BOL:                   p.Shipment.BOL,
		RevisionLabel:         fmt.Sprintf("Revision %d", p.Revision),
		ExternalDriverName:    p.Assignment.ExternalDriverName,
		ExternalDriverPhone:   p.Assignment.ExternalDriverPhone,
		ExternalTractorNumber: p.Assignment.ExternalTractorNumber,
		ExternalTrailerNumber: p.Assignment.ExternalTrailerNumber,
		RateMethodLabel:       rateMethodLabel(p.Assignment.RateMethod),
		BaseRateLabel:         baseRateLabel(p.Assignment),
		BaseAmount:            p.Assignment.BaseAmount.StringFixed(2),
		FuelSurcharge:         p.Assignment.FuelSurcharge.StringFixed(2),
		TotalCost:             p.Assignment.TotalCost.StringFixed(2),
		Currency:              p.Assignment.CurrencyCode,
		PaymentTermsLabel:     paymentTermsLabel(p.Carrier),
	}

	if p.Move != nil {
		out.Stops = stopRows(p.Move.Stops)
	}

	for _, acc := range p.Assignment.Accessorials {
		if acc == nil {
			continue
		}
		out.Accessorials = append(out.Accessorials, documenttemplate.RateConfirmationChargeRow{
			Description: acc.Description,
			Amount:      acc.Amount.StringFixed(2),
		})
	}

	return out
}

func stopRows(stops []*shipment.Stop) []documenttemplate.RateConfirmationStopRow {
	rows := make([]documenttemplate.RateConfirmationStopRow, 0, len(stops))
	for idx, stop := range stops {
		if stop == nil {
			continue
		}

		row := documenttemplate.RateConfirmationStopRow{
			Sequence: idx + 1,
			Type:     stopTypeLabel(stop.Type),
			Window:   windowLabel(stop.ScheduledWindowStart, stop.ScheduledWindowEnd),
			Address:  stop.AddressLine,
		}
		if stop.Location != nil {
			row.Name = stop.Location.Name
			if row.Address == "" {
				row.Address = fmt.Sprintf("%s, %s %s",
					stop.Location.AddressLine1, stop.Location.City, stop.Location.PostalCode)
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func stopTypeLabel(stopType shipment.StopType) string {
	switch stopType {
	case shipment.StopTypePickup, shipment.StopTypeSplitPickup:
		return "Pickup"
	case shipment.StopTypeDelivery, shipment.StopTypeSplitDelivery:
		return "Delivery"
	default:
		return string(stopType)
	}
}

func windowLabel(start int64, end *int64) string {
	if start <= 0 {
		return ""
	}

	startLabel := time.Unix(start, 0).UTC().Format(windowTimeLayout)
	if end == nil || *end <= 0 {
		return startLabel
	}

	endTime := time.Unix(*end, 0).UTC()
	if endTime.UTC().Truncate(24*time.Hour) == time.Unix(start, 0).UTC().Truncate(24*time.Hour) {
		return startLabel + "–" + endTime.Format("15:04 MST")
	}
	return startLabel + " – " + endTime.Format(windowTimeLayout)
}

func rateMethodLabel(method shipment.CarrierRateMethod) string {
	switch method {
	case shipment.CarrierRateMethodPerMile:
		return "Per mile"
	case shipment.CarrierRateMethodFlat:
		return "Flat"
	default:
		return string(method)
	}
}

func baseRateLabel(assignment *shipment.CarrierAssignment) string {
	switch assignment.RateMethod {
	case shipment.CarrierRateMethodPerMile:
		return assignment.BaseRate.StringFixed(2) + "/mi"
	case shipment.CarrierRateMethodFlat:
		return assignment.BaseRate.StringFixed(2) + " flat"
	default:
		return assignment.BaseRate.StringFixed(2)
	}
}

func paymentTermsLabel(entity *carrier.Carrier) string {
	if entity.PaymentTermDays <= 0 {
		return "Due on receipt of clean POD"
	}
	return fmt.Sprintf("Net %d from clean POD", entity.PaymentTermDays)
}
