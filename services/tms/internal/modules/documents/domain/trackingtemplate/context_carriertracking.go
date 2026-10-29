package trackingtemplate

import (
	"html/template"
	"slices"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
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

func newCarrierTrackingLinkEmailSampleContext() any {
	return CarrierTrackingLinkEmailContext{
		CarrierName:        "Redline Freight LLC",
		CompanyName:        templatecatalog.SampleCompanyName,
		ShipmentProNumber:  templatecatalog.SampleProNumber,
		OriginSummary:      templatecatalog.SampleOriginSummary,
		DestinationSummary: templatecatalog.SampleDestinationSummary,
		PickupWindow:       templatecatalog.SamplePickupWindow,
		DeliveryWindow:     templatecatalog.SampleDeliveryWindow,
		TrackingURL: template.URL(
			"https://app.example.com/carrier-tracking/8f2c41a9b7e34d05",
		),
		ExpiresAt: "2026-08-08 16:00 CDT",
	}
}

func RegisterCarrierTrackingKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindCarrierTrackingLinkEmail,
		DisplayName: "Carrier Tracking Link",
		Description: "Sends the carrier on a brokered load the link it posts " +
			"check calls, arrivals and departures through. Not customer-scoped: " +
			"the recipient is a carrier.",
		Category: trackingCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newCarrierTrackingLinkEmailSampleContext,
		Variables: slices.Concat(
			[]templatecatalog.VariableDefinition{
				{
					Path:        carrierNamePath,
					Type:        templatecatalog.VariableString,
					Description: "The carrier hauling the load.",
				},
				templatecatalog.CompanyNameVariable(),
				templatecatalog.ProNumberVariable(false, "The shipment the link tracks."),
			},
			templatecatalog.RouteScheduleVariables(),
			[]templatecatalog.VariableDefinition{
				{
					Path:        "TrackingURL",
					Type:        templatecatalog.VariableString,
					Required:    true,
					Description: "The carrier's tracking link. Required: without it the carrier has nowhere to post updates.",
				},
				{
					Path:        templatecatalog.ExpiresAtPath,
					Type:        templatecatalog.VariableDateTime,
					Description: "When the link stops working.",
				},
			},
		),
	})
}
