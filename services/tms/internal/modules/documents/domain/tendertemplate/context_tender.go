package tendertemplate

import (
	"html/template"
	"slices"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
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

func newTenderOfferEmailSampleContext() any {
	return TenderOfferEmailContext{
		CarrierName:        "Redline Freight LLC",
		CompanyName:        templatecatalog.SampleCompanyName,
		ShipmentProNumber:  templatecatalog.SampleProNumber,
		OriginSummary:      templatecatalog.SampleOriginSummary,
		DestinationSummary: templatecatalog.SampleDestinationSummary,
		PickupWindow:       templatecatalog.SamplePickupWindow,
		DeliveryWindow:     templatecatalog.SampleDeliveryWindow,
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

func RegisterTenderKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindTenderOfferEmail,
		DisplayName: "Carrier Tender Offer",
		Description: "Offers a brokered load to a carrier with accept and decline " +
			"links. Not customer-scoped: the recipient is a carrier, so a " +
			"per-customer assignment would never fire.",
		Category: "Tendering",
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newTenderOfferEmailSampleContext,
		Variables: slices.Concat(
			[]templatecatalog.VariableDefinition{
				{
					Path:        "CarrierName",
					Type:        templatecatalog.VariableString,
					Description: "The carrier the load is being offered to.",
				},
				templatecatalog.CompanyNameVariable(),
				templatecatalog.ProNumberVariable(false, "The shipment the offer covers."),
			},
			templatecatalog.RouteScheduleVariables(),
			[]templatecatalog.VariableDefinition{
				{
					Path:        "EquipmentSummary",
					Type:        templatecatalog.VariableString,
					Description: "The equipment the load requires.",
				},
				{
					Path:        "WeightSummary",
					Type:        templatecatalog.VariableString,
					Description: "The load's weight, formatted for display.",
				},
				{
					Path:        "RateAmount",
					Type:        templatecatalog.VariableString,
					Description: "The offered rate, formatted with currency.",
				},
				{
					Path:        "RateMethodLabel",
					Type:        templatecatalog.VariableString,
					Description: "How the rate is calculated, e.g. Flat or Per Mile.",
				},
				{
					Path:        "AcceptURL",
					Type:        templatecatalog.VariableString,
					Required:    true,
					Description: "The single-use accept link. Required: without it the carrier cannot take the load.",
				},
				{
					Path:        "DeclineURL",
					Type:        templatecatalog.VariableString,
					Required:    true,
					Description: "The single-use decline link. Required: without it the carrier cannot pass, and the offer sits until it expires.",
				},
				{
					Path:        templatecatalog.ExpiresAtPath,
					Type:        templatecatalog.VariableDateTime,
					Description: "The exact moment the offer stops being available.",
				},
				{
					Path:        "ExpiresInHours",
					Type:        templatecatalog.VariableInt,
					Description: "How many hours the offer is good for. Use this rather than writing a number into the copy, or the wording will drift from the real expiry.",
				},
			},
		),
	})
}
