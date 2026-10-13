package documenttemplate

import (
	"html/template"
	"slices"
)

// TenderOfferEmailContext is the data a carrier tender offer email renders against.
type TenderOfferEmailContext struct {
	CarrierName string
	CompanyName string

	ShipmentProNumber  string
	OriginSummary      string
	DestinationSummary string
	PickupWindow       string
	DeliveryWindow     string
	EquipmentSummary   string
	WeightSummary      string

	RateAmount      string
	RateMethodLabel string

	// AcceptURL and DeclineURL are single-use response links.
	//
	// Typed template.URL because this system builds them from the configured
	// public base URL and a generated token, so they are ours and cannot carry a
	// hostile scheme. Everything else on this context is text and is escaped
	// normally.
	AcceptURL  template.URL
	DeclineURL template.URL

	ExpiresAt      string
	ExpiresInHours int
}

//nolint:gosec // G101: a rendering fixture of fake values, not a credential
func newTenderOfferEmailSampleContext() any {
	return TenderOfferEmailContext{
		CarrierName:        "Redline Freight LLC",
		CompanyName:        sampleCompanyName,
		ShipmentProNumber:  sampleProNumber,
		OriginSummary:      sampleOriginSummary,
		DestinationSummary: sampleDestinationSummary,
		PickupWindow:       samplePickupWindow,
		DeliveryWindow:     sampleDeliveryWindow,
		EquipmentSummary:   "53' Dry Van",
		WeightSummary:      "42,000 lbs",
		RateAmount:         "$1,850.00",
		RateMethodLabel:    "Flat",
		AcceptURL: template.URL(
			"https://app.example.com/tender-offer/8f2c41a9b7e34d05/accept",
		),
		DeclineURL: template.URL(
			"https://app.example.com/tender-offer/8f2c41a9b7e34d05/decline",
		),
		ExpiresAt:      "2026-07-17 16:00 CDT",
		ExpiresInHours: 2,
	}
}

func (r *Registry) registerTenderKinds() {
	_ = r.Register(&KindDefinition{
		Kind:        KindTenderOfferEmail,
		DisplayName: "Carrier Tender Offer",
		Description: "Offers a brokered load to a carrier with accept and decline " +
			"links. Not customer-scoped: the recipient is a carrier, so a " +
			"per-customer assignment would never fire.",
		Category:      "Tendering",
		Channels:      []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		sampleFactory: newTenderOfferEmailSampleContext,
		Variables: slices.Concat(
			[]VariableDefinition{
				{
					Path:        "CarrierName",
					Type:        VariableString,
					Description: "The carrier the load is being offered to.",
				},
				companyNameVariable(),
				proNumberVariable(false, "The shipment the offer covers."),
			},
			routeScheduleVariables(),
			[]VariableDefinition{
				{
					Path:        "EquipmentSummary",
					Type:        VariableString,
					Description: "The equipment the load requires.",
				},
				{
					Path:        "WeightSummary",
					Type:        VariableString,
					Description: "The load's weight, formatted for display.",
				},
				{
					Path:        "RateAmount",
					Type:        VariableString,
					Description: "The offered rate, formatted with currency.",
				},
				{
					Path:        "RateMethodLabel",
					Type:        VariableString,
					Description: "How the rate is calculated, e.g. Flat or Per Mile.",
				},
				{
					Path:        "AcceptURL",
					Type:        VariableString,
					Required:    true,
					Description: "The single-use accept link. Required: without it the carrier cannot take the load.",
				},
				{
					Path:        "DeclineURL",
					Type:        VariableString,
					Required:    true,
					Description: "The single-use decline link. Required: without it the carrier cannot pass, and the offer sits until it expires.",
				},
				{
					Path:        expiresAtPath,
					Type:        VariableDateTime,
					Description: "The exact moment the offer stops being available.",
				},
				{
					Path:        "ExpiresInHours",
					Type:        VariableInt,
					Description: "How many hours the offer is good for. Use this rather than writing a number into the copy, or the wording will drift from the real expiry.",
				},
			},
		),
	})
}
