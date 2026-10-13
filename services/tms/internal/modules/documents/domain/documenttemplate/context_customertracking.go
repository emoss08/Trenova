package documenttemplate

import (
	"html/template"
	"slices"
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
		CustomerName:       sampleCustomerName,
		CompanyName:        sampleCompanyName,
		ShipmentProNumber:  sampleProNumber,
		BOL:                sampleCustomerBOL,
		OriginSummary:      sampleOriginSummary,
		DestinationSummary: sampleDestinationSummary,
		PickupWindow:       samplePickupWindow,
		DeliveryWindow:     sampleDeliveryWindow,
		TrackingURL:        template.URL(sampleTrackingURL),
		ExpiresAt:          sampleLinkExpiry,
	}
}

//nolint:gosec // G101: a rendering fixture of fake values, not a credential
func newCustomerStatusEmailSampleContext() any {
	return CustomerStatusEmailContext{
		CustomerName:       sampleCustomerName,
		CompanyName:        sampleCompanyName,
		Event:              "Delayed",
		IsDelayed:          true,
		ShipmentProNumber:  sampleProNumber,
		BOL:                sampleCustomerBOL,
		OriginSummary:      sampleOriginSummary,
		DestinationSummary: sampleDestinationSummary,
		StopSummary:        sampleDestinationSummary,
		StopWindow:         sampleDeliveryWindow,
		OccurredAt:         "2026-07-19 07:42 CDT",
		EstimatedArrival:   "2026-07-19 16:30 CDT",
		TrackingURL:        template.URL(sampleTrackingURL),
		ExpiresAt:          sampleLinkExpiry,
	}
}

func bolVariable() VariableDefinition {
	return VariableDefinition{
		Path:        bolPath,
		Type:        VariableString,
		Description: "The bill of lading number, when the shipment has one.",
	}
}

func linkExpiryVariable(description string) VariableDefinition {
	return VariableDefinition{
		Path:        expiresAtPath,
		Type:        VariableDateTime,
		Description: description,
	}
}

func (r *Registry) registerCustomerTrackingKinds() {
	r.registerCustomerTrackingLinkKind()
	r.registerCustomerStatusKind()
}

func (r *Registry) registerCustomerTrackingLinkKind() {
	_ = r.Register(&KindDefinition{
		Kind:        KindCustomerTrackingLinkEmail,
		DisplayName: "Customer Tracking Link",
		Description: "Sends a customer the link to a shipment's tracking page. " +
			"What the page shows is set on the customer.",
		Category:       trackingCategory,
		Channels:       []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		CustomerScoped: true,
		sampleFactory:  newCustomerTrackingLinkEmailSampleContext,
		Variables: slices.Concat(
			[]VariableDefinition{
				customerNameVariable(false, "The customer the shipment is for."),
				companyNameVariable(),
				proNumberVariable(false, "The shipment the link tracks."),
				bolVariable(),
			},
			routeScheduleVariables(),
			[]VariableDefinition{
				{
					Path:        trackingURLPath,
					Type:        VariableString,
					Required:    true,
					Description: "The tracking page link. Required: it is the point of the email.",
				},
				linkExpiryVariable("When the link stops working."),
			},
		),
	})
}

func (r *Registry) registerCustomerStatusKind() {
	_ = r.Register(&KindDefinition{
		Kind:        KindCustomerStatusEmail,
		DisplayName: "Customer Status Update",
		Description: "Tells a customer its load was picked up, is running late, " +
			"was delivered, or has its proof of delivery on file, as the " +
			"customer's rules ask. Each event is sent once.",
		Category:       trackingCategory,
		Channels:       []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		CustomerScoped: true,
		sampleFactory:  newCustomerStatusEmailSampleContext,
		Variables: slices.Concat(
			[]VariableDefinition{
				customerNameVariable(false, "The customer the shipment is for."),
				companyNameVariable(),
				{
					Path:        "Event",
					Type:        VariableString,
					Description: "What happened: PickedUp, Delayed, Delivered or PodReady.",
				},
				{
					Path:        "IsPickedUp",
					Type:        VariableBool,
					Description: "True when the load has been picked up.",
				},
				{
					Path:        "IsDelayed",
					Type:        VariableBool,
					Description: "True when the load is projected to reach a stop late.",
				},
				{
					Path:        "IsDelivered",
					Type:        VariableBool,
					Description: "True when the load has been delivered.",
				},
				{
					Path:        "IsPodReady",
					Type:        VariableBool,
					Description: "True when the proof of delivery is on file.",
				},
				proNumberVariable(
					true,
					"The shipment the email is about. Required: it is how the customer knows which load.",
				),
				bolVariable(),
			},
			routeVariables(),
			[]VariableDefinition{
				{
					Path:        "StopSummary",
					Type:        VariableString,
					Description: "The stop a delay is about, as city and state.",
				},
				{
					Path:        "StopWindow",
					Type:        VariableString,
					Description: "The appointment window at the delayed stop.",
				},
				{
					Path:        "OccurredAt",
					Type:        VariableDateTime,
					Description: "When it happened: the departure, the projection, or the filing.",
				},
				{
					Path:        "EstimatedArrival",
					Type:        VariableDateTime,
					Description: "The projected arrival at the delayed stop.",
				},
				{
					Path:        trackingURLPath,
					Type:        VariableString,
					Description: "The tracking page link, when the customer's rules include it.",
				},
				linkExpiryVariable("When the tracking link stops working."),
			},
		),
	})
}
