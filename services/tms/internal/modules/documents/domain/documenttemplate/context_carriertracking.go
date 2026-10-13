package documenttemplate

import "html/template"

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

//nolint:gosec // G101: a rendering fixture of fake values, not a credential
func newCarrierTrackingLinkEmailSampleContext() any {
	return CarrierTrackingLinkEmailContext{
		CarrierName:        "Redline Freight LLC",
		CompanyName:        sampleCompanyName,
		ShipmentProNumber:  sampleProNumber,
		OriginSummary:      "Dallas, TX",
		DestinationSummary: "Memphis, TN",
		PickupWindow:       "2026-07-18 08:00 – 12:00 CDT",
		DeliveryWindow:     "2026-07-19 09:00 – 15:00 CDT",
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
		Category:      "Tracking",
		Channels:      []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		sampleFactory: newCarrierTrackingLinkEmailSampleContext,
		Variables: []VariableDefinition{
			{
				Path:        "CarrierName",
				Type:        VariableString,
				Description: "The carrier hauling the load.",
			},
			companyNameVariable(),
			proNumberVariable(false, "The shipment the link tracks."),
			{
				Path:        "OriginSummary",
				Type:        VariableString,
				Description: "Where the load picks up, as city and state.",
			},
			{
				Path:        "DestinationSummary",
				Type:        VariableString,
				Description: "Where the load delivers, as city and state.",
			},
			{
				Path:        "PickupWindow",
				Type:        VariableString,
				Description: "The planned pickup window.",
			},
			{
				Path:        "DeliveryWindow",
				Type:        VariableString,
				Description: "The planned delivery window.",
			},
			{
				Path:        "TrackingURL",
				Type:        VariableString,
				Required:    true,
				Description: "The carrier's tracking link. Required: without it the carrier has nowhere to post updates.",
			},
			{
				Path:        "ExpiresAt",
				Type:        VariableDateTime,
				Description: "When the link stops working.",
			},
		},
	})
}
