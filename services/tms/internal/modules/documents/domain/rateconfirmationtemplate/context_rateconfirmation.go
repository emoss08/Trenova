package rateconfirmationtemplate

import (
	"html/template"
	"slices"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

// RateConfirmationContext is the data a carrier rate confirmation renders
// against.
//
// Every figure comes from the payload snapshot frozen when the confirmation was
// generated, not from the live carrier assignment. A carrier holds the document
// they were sent; if dispatch renegotiates, a new revision is generated and the
// prior one voided, so the paper and the payable can never silently diverge.
type RateConfirmationContext struct {
	CompanyName string

	CarrierName string
	CarrierSCAC string
	CarrierMC   string
	CarrierDOT  string

	ShipmentProNumber string
	BOL               string
	RevisionLabel     string

	ExternalDriverName    string
	ExternalDriverPhone   string
	ExternalTractorNumber string
	ExternalTrailerNumber string

	// Stops is the routing the carrier is agreeing to run, in sequence.
	Stops []RateConfirmationStopRow

	RateMethodLabel string
	BaseRateLabel   string
	BaseAmount      string
	FuelSurcharge   string
	// Accessorials are the agreed extra charges. Empty when the linehaul is
	// all-in.
	Accessorials []RateConfirmationChargeRow
	TotalCost    string
	Currency     string

	PaymentTermsLabel string

	// SignURL is the single-use public sign link injected at send time for a
	// not-yet-executed revision. Empty on the frozen snapshot and on the
	// record copy of an already-confirmed agreement.
	//
	// Typed template.URL because this system builds it from the configured
	// public base URL and a generated token, so it is ours and cannot carry a
	// hostile scheme.
	SignURL          template.URL
	SignExpiresLabel string

	LogoDataURI template.URL
}

// RateConfirmationStopRow is one stop of the move being tendered.
type RateConfirmationStopRow struct {
	Sequence int
	Type     string
	Name     string
	Address  string
	Window   string
}

// RateConfirmationChargeRow is one agreed accessorial charge.
type RateConfirmationChargeRow struct {
	Description string
	Amount      string
}

func newRateConfirmationSampleContext() any {
	return RateConfirmationContext{
		CompanyName:           templatecatalog.SampleCompanyName,
		CarrierName:           "Eastline Transport LLC",
		CarrierSCAC:           "ESTL",
		CarrierMC:             "123456",
		CarrierDOT:            "7654321",
		ShipmentProNumber:     templatecatalog.SampleProNumber,
		BOL:                   "BOL-58291",
		RevisionLabel:         "Revision 2",
		ExternalDriverName:    "R. Alvarez",
		ExternalDriverPhone:   "(555) 201-8890",
		ExternalTractorNumber: "4482",
		ExternalTrailerNumber: "T-9917",
		Stops: []RateConfirmationStopRow{
			{
				Sequence: 1,
				Type:     "Pickup",
				Name:     "Lakeside Distribution",
				Address:  "2100 Harbor Dr, Green Bay, WI 54302",
				Window:   "Mon 13 Jul 2026 08:00–12:00 CDT",
			},
			{
				Sequence: 2,
				Type:     "Delivery",
				Name:     "Prairie Foods DC",
				Address:  "880 Commerce Pkwy, Des Moines, IA 50313",
				Window:   "Tue 14 Jul 2026 06:00–10:00 CDT",
			},
		},
		RateMethodLabel:   "Flat",
		BaseRateLabel:     "1,850.00 flat",
		BaseAmount:        "1850.00",
		FuelSurcharge:     "142.50",
		Accessorials:      []RateConfirmationChargeRow{{Description: "Liftgate", Amount: "75.00"}},
		TotalCost:         "2067.50",
		Currency:          templatecatalog.SampleCurrency,
		PaymentTermsLabel: "Net 30 from clean POD",
		SignURL: template.URL(
			"https://app.example.com/rate-confirmation/8f2c41a9b7e34d05",
		),
		SignExpiresLabel: "2026-07-27 16:00 UTC",
		//nolint:gosec // A compile-time constant data: URI; see the field's doc comment.
		LogoDataURI: template.URL(templatecatalog.SampleLogoDataURI),
	}
}

func RegisterRateConfirmationKinds(r *templatecatalog.Registry) {
	variables := rateConfirmationVariables()

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindRateConfirmationPDF,
		DisplayName: "Rate Confirmation Document",
		Description: "The signed-rate agreement sent to an external carrier before it hauls " +
			"a brokered move. This is the artifact a disputed carrier invoice is settled against.",
		Category:      "Carrier",
		Channels:      []documenttemplate.Channel{documenttemplate.ChannelPDF},
		Paged:         true,
		SampleFactory: newRateConfirmationSampleContext,
		Variables:     variables,
	})

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindRateConfirmationEmail,
		DisplayName: "Rate Confirmation Email",
		Description: "Delivers the rate confirmation to the carrier's dispatch contacts " +
			"with the document attached.",
		Category: "Carrier",
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		SampleFactory: newRateConfirmationSampleContext,
		Variables:     variables,
	})
}

func rateConfirmationVariables() []templatecatalog.VariableDefinition {
	return slices.Concat(
		rateConfirmationPartyVariables(),
		rateConfirmationEquipmentVariables(),
		rateConfirmationRoutingVariables(),
		rateConfirmationRateVariables(),
		rateConfirmationFooterVariables(),
	)
}

func rateConfirmationPartyVariables() []templatecatalog.VariableDefinition {
	return []templatecatalog.VariableDefinition{
		templatecatalog.CompanyNameVariable(),
		{
			Path:        "CarrierName",
			Type:        templatecatalog.VariableString,
			Required:    true,
			Description: "The carrier being tendered the move. A rate confirmation that does not name its counterparty is unenforceable.",
		},
		{
			Path:        "CarrierSCAC",
			Type:        templatecatalog.VariableString,
			Description: "The carrier's SCAC.",
		},
		{
			Path:        "CarrierMC",
			Type:        templatecatalog.VariableString,
			Description: "The carrier's MC number.",
		},
		{
			Path:        "CarrierDOT",
			Type:        templatecatalog.VariableString,
			Description: "The carrier's DOT number.",
		},
		templatecatalog.ProNumberVariable(
			true,
			"The shipment the move belongs to, which is the reference both sides quote.",
		),
		{
			Path:        "BOL",
			Type:        templatecatalog.VariableString,
			Description: "The bill of lading number.",
		},
		{
			Path:        "RevisionLabel",
			Type:        templatecatalog.VariableString,
			Description: "Which revision of the agreement this is. A renegotiated rate voids the prior revision, and the label is how a carrier tells the two apart.",
		},
	}
}

func rateConfirmationEquipmentVariables() []templatecatalog.VariableDefinition {
	return []templatecatalog.VariableDefinition{
		{
			Path:        "ExternalDriverName",
			Type:        templatecatalog.VariableString,
			Description: "The carrier's driver, when known at tender time.",
		},
		{
			Path:        "ExternalDriverPhone",
			Type:        templatecatalog.VariableString,
			Description: "The driver's phone.",
		},
		{
			Path:        "ExternalTractorNumber",
			Type:        templatecatalog.VariableString,
			Description: "The carrier's tractor unit, when known.",
		},
		{
			Path:        "ExternalTrailerNumber",
			Type:        templatecatalog.VariableString,
			Description: "The carrier's trailer unit, when known.",
		},
	}
}

func rateConfirmationRoutingVariables() []templatecatalog.VariableDefinition {
	return []templatecatalog.VariableDefinition{
		{
			Path:        "Stops",
			Type:        templatecatalog.VariableCollection,
			Required:    true,
			Description: "The routing being tendered, in stop order.",
			Fields: []templatecatalog.VariableDefinition{
				{
					Path:        "Sequence",
					Type:        templatecatalog.VariableInt,
					Description: "Stop order, starting at 1.",
				},
				{
					Path:        "Type",
					Type:        templatecatalog.VariableString,
					Description: "Pickup or Delivery.",
				},
				{
					Path:        "Name",
					Type:        templatecatalog.VariableString,
					Description: "The facility name.",
				},
				{
					Path:        "Address",
					Type:        templatecatalog.VariableString,
					Description: "The facility address.",
				},
				{
					Path:        "Window",
					Type:        templatecatalog.VariableString,
					Description: "The scheduled window.",
				},
			},
		},
	}
}

func rateConfirmationRateVariables() []templatecatalog.VariableDefinition {
	return []templatecatalog.VariableDefinition{
		{
			Path:        "RateMethodLabel",
			Type:        templatecatalog.VariableString,
			Description: "How the linehaul is rated: Flat or Per mile.",
		},
		{
			Path:        "BaseRateLabel",
			Type:        templatecatalog.VariableString,
			Description: "The negotiated linehaul rate, in words, such as 1,850.00 flat or 2.35/mi.",
		},
		{
			Path:        "BaseAmount",
			Type:        templatecatalog.VariableString,
			Description: "The extended linehaul amount, unformatted.",
		},
		{
			Path:        "FuelSurcharge",
			Type:        templatecatalog.VariableString,
			Description: "The agreed fuel surcharge amount.",
		},
		{
			Path:        "Accessorials",
			Type:        templatecatalog.VariableCollection,
			Description: "Agreed accessorial charges. Empty for an all-in rate.",
			Fields: []templatecatalog.VariableDefinition{
				{
					Path:        "Description",
					Type:        templatecatalog.VariableString,
					Description: "What the charge is for.",
				},
				{
					Path:        "Amount",
					Type:        templatecatalog.VariableString,
					Description: "The charge amount.",
				},
			},
		},
		{
			Path:        "TotalCost",
			Type:        templatecatalog.VariableString,
			Required:    true,
			Description: "The all-in amount the carrier will be paid. This is the number a carrier invoice is matched against.",
		},
		{
			Path:        "Currency",
			Type:        templatecatalog.VariableString,
			Description: "Currency of the agreement.",
		},
		{
			Path:        "PaymentTermsLabel",
			Type:        templatecatalog.VariableString,
			Description: "When the carrier is paid, in words.",
		},
	}
}

func rateConfirmationFooterVariables() []templatecatalog.VariableDefinition {
	return []templatecatalog.VariableDefinition{
		{
			Path:        "SignURL",
			Type:        templatecatalog.VariableString,
			Description: "The single-use review-and-sign link. Present only on an email carrying a revision that still needs the carrier's signature; empty on the executed record copy.",
		},
		{
			Path:        "SignExpiresLabel",
			Type:        templatecatalog.VariableString,
			Description: "When the sign link stops working, in words.",
		},
		templatecatalog.LogoVariable(),
	}
}
