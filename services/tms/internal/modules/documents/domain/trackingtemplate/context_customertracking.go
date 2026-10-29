package trackingtemplate

import (
	"html/template"
	"slices"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

// CustomerTrackingLinkEmailContext is the email that hands a customer the link
// to one shipment's tracking page.
type CustomerTrackingLinkEmailContext struct {
	CustomerName string
	CompanyName  string

	ShipmentProNumber  string
	BOL                string
	OriginSummary      string
	DestinationSummary string
	PickupWindow       string
	DeliveryWindow     string

	TrackingURL template.URL

	ExpiresAt string
}

// CustomerStatusEmailContext is one status email a customer's rules ask for:
// the load was picked up, is running late to a stop, was delivered, or has its
// proof of delivery on file. Exactly one of the Is* flags is set.
type CustomerStatusEmailContext struct {
	CustomerName string
	CompanyName  string

	Event       string
	IsPickedUp  bool
	IsDelayed   bool
	IsDelivered bool
	IsPodReady  bool

	ShipmentProNumber  string
	BOL                string
	OriginSummary      string
	DestinationSummary string

	StopSummary      string
	StopWindow       string
	OccurredAt       string
	EstimatedArrival string

	TrackingURL template.URL
	ExpiresAt   string
}

const (
	trackingURLPath   = "TrackingURL"
	bolPath           = "BOL"
	sampleTrackingURL = "https://app.example.com/shipment-tracking/5d1e09c4a8f24b7e"
	sampleLinkExpiry  = "2026-08-17 16:00 CDT"
	sampleCustomerBOL = "BOL-55120"
)

//nolint:gosec // G101: a rendering fixture of fake values, not a credential
func newCustomerTrackingLinkEmailSampleContext() any {
	return CustomerTrackingLinkEmailContext{
		CustomerName:       templatecatalog.SampleCustomerName,
		CompanyName:        templatecatalog.SampleCompanyName,
		ShipmentProNumber:  templatecatalog.SampleProNumber,
		BOL:                sampleCustomerBOL,
		OriginSummary:      templatecatalog.SampleOriginSummary,
		DestinationSummary: templatecatalog.SampleDestinationSummary,
		PickupWindow:       templatecatalog.SamplePickupWindow,
		DeliveryWindow:     templatecatalog.SampleDeliveryWindow,
		TrackingURL:        template.URL(sampleTrackingURL),
		ExpiresAt:          sampleLinkExpiry,
	}
}

//nolint:gosec // G101: a rendering fixture of fake values, not a credential
func newCustomerStatusEmailSampleContext() any {
	return CustomerStatusEmailContext{
		CustomerName:       templatecatalog.SampleCustomerName,
		CompanyName:        templatecatalog.SampleCompanyName,
		Event:              "Delayed",
		IsDelayed:          true,
		ShipmentProNumber:  templatecatalog.SampleProNumber,
		BOL:                sampleCustomerBOL,
		OriginSummary:      templatecatalog.SampleOriginSummary,
		DestinationSummary: templatecatalog.SampleDestinationSummary,
		StopSummary:        templatecatalog.SampleDestinationSummary,
		StopWindow:         templatecatalog.SampleDeliveryWindow,
		OccurredAt:         "2026-07-19 07:42 CDT",
		EstimatedArrival:   "2026-07-19 16:30 CDT",
		TrackingURL:        template.URL(sampleTrackingURL),
		ExpiresAt:          sampleLinkExpiry,
	}
}

func bolVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        bolPath,
		Type:        templatecatalog.VariableString,
		Description: "The bill of lading number, when the shipment has one.",
	}
}

func linkExpiryVariable(description string) templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        templatecatalog.ExpiresAtPath,
		Type:        templatecatalog.VariableDateTime,
		Description: description,
	}
}

func RegisterCustomerTrackingKinds(r *templatecatalog.Registry) {
	registerCustomerTrackingLinkKind(r)
	registerCustomerStatusKind(r)
}

func registerCustomerTrackingLinkKind(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindCustomerTrackingLinkEmail,
		DisplayName: "Customer Tracking Link",
		Description: "Sends a customer the link to a shipment's tracking page. " +
			"What the page shows is set on the customer.",
		Category: trackingCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		CustomerScoped: true,
		SampleFactory:  newCustomerTrackingLinkEmailSampleContext,
		Variables: slices.Concat(
			[]templatecatalog.VariableDefinition{
				templatecatalog.CustomerNameVariable(false, "The customer the shipment is for."),
				templatecatalog.CompanyNameVariable(),
				templatecatalog.ProNumberVariable(false, "The shipment the link tracks."),
				bolVariable(),
			},
			templatecatalog.RouteScheduleVariables(),
			[]templatecatalog.VariableDefinition{
				{
					Path:        trackingURLPath,
					Type:        templatecatalog.VariableString,
					Required:    true,
					Description: "The tracking page link. Required: it is the point of the email.",
				},
				linkExpiryVariable("When the link stops working."),
			},
		),
	})
}

func registerCustomerStatusKind(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindCustomerStatusEmail,
		DisplayName: "Customer Status Update",
		Description: "Tells a customer its load was picked up, is running late, " +
			"was delivered, or has its proof of delivery on file, as the " +
			"customer's rules ask. Each event is sent once.",
		Category: trackingCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		CustomerScoped: true,
		SampleFactory:  newCustomerStatusEmailSampleContext,
		Variables: slices.Concat(
			[]templatecatalog.VariableDefinition{
				templatecatalog.CustomerNameVariable(false, "The customer the shipment is for."),
				templatecatalog.CompanyNameVariable(),
				{
					Path:        "Event",
					Type:        templatecatalog.VariableString,
					Description: "What happened: PickedUp, Delayed, Delivered or PodReady.",
				},
				{
					Path:        "IsPickedUp",
					Type:        templatecatalog.VariableBool,
					Description: "True when the load has been picked up.",
				},
				{
					Path:        "IsDelayed",
					Type:        templatecatalog.VariableBool,
					Description: "True when the load is projected to reach a stop late.",
				},
				{
					Path:        "IsDelivered",
					Type:        templatecatalog.VariableBool,
					Description: "True when the load has been delivered.",
				},
				{
					Path:        "IsPodReady",
					Type:        templatecatalog.VariableBool,
					Description: "True when the proof of delivery is on file.",
				},
				templatecatalog.ProNumberVariable(
					true,
					"The shipment the email is about. Required: it is how the customer knows which load.",
				),
				bolVariable(),
			},
			templatecatalog.RouteVariables(),
			[]templatecatalog.VariableDefinition{
				{
					Path:        "StopSummary",
					Type:        templatecatalog.VariableString,
					Description: "The stop a delay is about, as city and state.",
				},
				{
					Path:        "StopWindow",
					Type:        templatecatalog.VariableString,
					Description: "The appointment window at the delayed stop.",
				},
				{
					Path:        "OccurredAt",
					Type:        templatecatalog.VariableDateTime,
					Description: "When it happened: the departure, the projection, or the filing.",
				},
				{
					Path:        "EstimatedArrival",
					Type:        templatecatalog.VariableDateTime,
					Description: "The projected arrival at the delayed stop.",
				},
				{
					Path:        trackingURLPath,
					Type:        templatecatalog.VariableString,
					Description: "The tracking page link, when the customer's rules include it.",
				},
				linkExpiryVariable("When the tracking link stops working."),
			},
		),
	})
}

const (
	trackingCategory = "Tracking"
)
