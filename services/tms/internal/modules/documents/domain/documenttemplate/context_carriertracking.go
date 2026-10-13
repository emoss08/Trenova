package documenttemplate

import (
	"html/template"
	"slices"
)

type CarrierTrackingLinkEmailContext struct {
	CarrierName string
	CompanyName string

	ShipmentProNumber  string
	OriginSummary      string
	DestinationSummary string
	PickupWindow       string
	DeliveryWindow     string

	TrackingURL template.URL

	ExpiresAt string
}

const carrierNamePath = "CarrierName"

//nolint:gosec // G101: a rendering fixture of fake values, not a credential
func newCarrierTrackingLinkEmailSampleContext() any {
	return CarrierTrackingLinkEmailContext{
		CarrierName:        "Redline Freight LLC",
		CompanyName:        sampleCompanyName,
		ShipmentProNumber:  sampleProNumber,
		OriginSummary:      sampleOriginSummary,
		DestinationSummary: sampleDestinationSummary,
		PickupWindow:       samplePickupWindow,
		DeliveryWindow:     sampleDeliveryWindow,
		TrackingURL: template.URL(
			"https://app.example.com/carrier-tracking/8f2c41a9b7e34d05",
		),
		ExpiresAt: "2026-08-08 16:00 CDT",
	}
}

func (r *Registry) registerCarrierTrackingKinds() {
	_ = r.Register(&KindDefinition{
		Kind:        KindCarrierTrackingLinkEmail,
		DisplayName: "Carrier Tracking Link",
		Description: "Sends the carrier on a brokered load the link it posts " +
			"check calls, arrivals and departures through. Not customer-scoped: " +
			"the recipient is a carrier.",
		Category:      trackingCategory,
		Channels:      []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		sampleFactory: newCarrierTrackingLinkEmailSampleContext,
		Variables: slices.Concat(
			[]VariableDefinition{
				{
					Path:        carrierNamePath,
					Type:        VariableString,
					Description: "The carrier hauling the load.",
				},
				companyNameVariable(),
				proNumberVariable(false, "The shipment the link tracks."),
			},
			routeScheduleVariables(),
			[]VariableDefinition{
				{
					Path:        "TrackingURL",
					Type:        VariableString,
					Required:    true,
					Description: "The carrier's tracking link. Required: without it the carrier has nowhere to post updates.",
				},
				{
					Path:        expiresAtPath,
					Type:        VariableDateTime,
					Description: "When the link stops working.",
				},
			},
		),
	})
}
