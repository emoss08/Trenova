package shipmentenvelope

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

type Envelope struct {
	LengthFeet        float64 `json:"lengthFeet"`
	WidthFeet         float64 `json:"widthFeet"`
	HeightFeet        float64 `json:"heightFeet"`
	OverallHeightFeet float64 `json:"overallHeightFeet"`
}

func (e Envelope) IsZero() bool {
	return e.LengthFeet == 0 && e.WidthFeet == 0 && e.HeightFeet == 0
}

// Compute derives the travel envelope from the cargo lines. Width and
// height take the maximum across lines because the widest and tallest single
// item governs what the combination can clear. Length also takes the maximum
// rather than the sum: total deck occupancy is a fit question answered by
// linear feet, while a length permit is triggered by the longest overhanging
// item.
func Compute(commodities []*shipment.ShipmentCommodity, deckHeightFeet float64) Envelope {
	var envelope Envelope

	for _, sc := range commodities {
		if sc == nil {
			continue
		}

		envelope.LengthFeet = max(envelope.LengthFeet, sc.ResolvedLengthFeet())
		envelope.WidthFeet = max(envelope.WidthFeet, sc.ResolvedWidthFeet())
		envelope.HeightFeet = max(envelope.HeightFeet, sc.ResolvedHeightFeet())
	}

	if envelope.HeightFeet > 0 {
		envelope.OverallHeightFeet = envelope.HeightFeet + deckHeightFeet
	}

	return envelope
}

func Apply(sp *shipment.Shipment, envelope Envelope) {
	if envelope.IsZero() {
		sp.EnvelopeLengthFeet = nil
		sp.EnvelopeWidthFeet = nil
		sp.EnvelopeHeightFeet = nil
		sp.EnvelopeOverallHeightFeet = nil
		return
	}

	sp.EnvelopeLengthFeet = nonZero(envelope.LengthFeet)
	sp.EnvelopeWidthFeet = nonZero(envelope.WidthFeet)
	sp.EnvelopeHeightFeet = nonZero(envelope.HeightFeet)
	sp.EnvelopeOverallHeightFeet = nonZero(envelope.OverallHeightFeet)
}

func Of(sp *shipment.Shipment) Envelope {
	return Envelope{
		LengthFeet:        deref(sp.EnvelopeLengthFeet),
		WidthFeet:         deref(sp.EnvelopeWidthFeet),
		HeightFeet:        deref(sp.EnvelopeHeightFeet),
		OverallHeightFeet: deref(sp.EnvelopeOverallHeightFeet),
	}
}

func Present(sp *shipment.Shipment) bool {
	return !Of(sp).IsZero()
}

func nonZero(value float64) *float64 {
	if value == 0 {
		return nil
	}
	return &value
}

func deref(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
