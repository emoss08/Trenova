package documenttemplate

import "html/template"

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
		CompanyName:           sampleCompanyName,
		CarrierName:           "Eastline Transport LLC",
		CarrierSCAC:           "ESTL",
		CarrierMC:             "123456",
		CarrierDOT:            "7654321",
		ShipmentProNumber:     sampleProNumber,
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
		Currency:          sampleCurrency,
		PaymentTermsLabel: "Net 30 from clean POD",
		SignURL: template.URL(
			"https://app.example.com/rate-confirmation/8f2c41a9b7e34d05",
		),
		SignExpiresLabel: "2026-07-27 16:00 UTC",
		//nolint:gosec // A compile-time constant data: URI; see the field's doc comment.
		LogoDataURI: template.URL(sampleLogoDataURI),
	}
}

func (r *Registry) registerRateConfirmationKinds() {
	variables := rateConfirmationVariables()

	_ = r.Register(&KindDefinition{
		Kind:        KindRateConfirmationPDF,
		DisplayName: "Rate Confirmation Document",
		Description: "The signed-rate agreement sent to an external carrier before it hauls " +
			"a brokered move. This is the artifact a disputed carrier invoice is settled against.",
		Category:      "Carrier",
		Channels:      []Channel{ChannelPDF},
		Paged:         true,
		sampleFactory: newRateConfirmationSampleContext,
		Variables:     variables,
	})

	_ = r.Register(&KindDefinition{
		Kind:        KindRateConfirmationEmail,
		DisplayName: "Rate Confirmation Email",
		Description: "Delivers the rate confirmation to the carrier's dispatch contacts " +
			"with the document attached.",
		Category:      "Carrier",
		Channels:      []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		sampleFactory: newRateConfirmationSampleContext,
		Variables:     variables,
	})
}

func rateConfirmationVariables() []VariableDefinition {
	return []VariableDefinition{
		companyNameVariable(),
		{
			Path:        "CarrierName",
			Type:        VariableString,
			Required:    true,
			Description: "The carrier being tendered the move. A rate confirmation that does not name its counterparty is unenforceable.",
		},
		{Path: "CarrierSCAC", Type: VariableString, Description: "The carrier's SCAC."},
		{Path: "CarrierMC", Type: VariableString, Description: "The carrier's MC number."},
		{Path: "CarrierDOT", Type: VariableString, Description: "The carrier's DOT number."},
		proNumberVariable(
			true,
			"The shipment the move belongs to, which is the reference both sides quote.",
		),
		{Path: "BOL", Type: VariableString, Description: "The bill of lading number."},
		{
			Path:        "RevisionLabel",
			Type:        VariableString,
			Description: "Which revision of the agreement this is. A renegotiated rate voids the prior revision, and the label is how a carrier tells the two apart.",
		},
		{
			Path:        "ExternalDriverName",
			Type:        VariableString,
			Description: "The carrier's driver, when known at tender time.",
		},
		{Path: "ExternalDriverPhone", Type: VariableString, Description: "The driver's phone."},
		{
			Path:        "ExternalTractorNumber",
			Type:        VariableString,
			Description: "The carrier's tractor unit, when known.",
		},
		{
			Path:        "ExternalTrailerNumber",
			Type:        VariableString,
			Description: "The carrier's trailer unit, when known.",
		},
		{
			Path:        "Stops",
			Type:        VariableCollection,
			Required:    true,
			Description: "The routing being tendered, in stop order.",
			Fields: []VariableDefinition{
				{Path: "Sequence", Type: VariableInt, Description: "Stop order, starting at 1."},
				{Path: "Type", Type: VariableString, Description: "Pickup or Delivery."},
				{Path: "Name", Type: VariableString, Description: "The facility name."},
				{Path: "Address", Type: VariableString, Description: "The facility address."},
				{Path: "Window", Type: VariableString, Description: "The scheduled window."},
			},
		},
		{
			Path:        "RateMethodLabel",
			Type:        VariableString,
			Description: "How the linehaul is rated: Flat or Per mile.",
		},
		{
			Path:        "BaseRateLabel",
			Type:        VariableString,
			Description: "The negotiated linehaul rate, in words, such as 1,850.00 flat or 2.35/mi.",
		},
		{
			Path:        "BaseAmount",
			Type:        VariableString,
			Description: "The extended linehaul amount, unformatted.",
		},
		{
			Path:        "FuelSurcharge",
			Type:        VariableString,
			Description: "The agreed fuel surcharge amount.",
		},
		{
			Path:        "Accessorials",
			Type:        VariableCollection,
			Description: "Agreed accessorial charges. Empty for an all-in rate.",
			Fields: []VariableDefinition{
				{
					Path:        "Description",
					Type:        VariableString,
					Description: "What the charge is for.",
				},
				{Path: "Amount", Type: VariableString, Description: "The charge amount."},
			},
		},
		{
			Path:        "TotalCost",
			Type:        VariableString,
			Required:    true,
			Description: "The all-in amount the carrier will be paid. This is the number a carrier invoice is matched against.",
		},
		{Path: "Currency", Type: VariableString, Description: "Currency of the agreement."},
		{
			Path:        "PaymentTermsLabel",
			Type:        VariableString,
			Description: "When the carrier is paid, in words.",
		},
		{
			Path:        "SignURL",
			Type:        VariableString,
			Description: "The single-use review-and-sign link. Present only on an email carrying a revision that still needs the carrier's signature; empty on the executed record copy.",
		},
		{
			Path:        "SignExpiresLabel",
			Type:        VariableString,
			Description: "When the sign link stops working, in words.",
		},
		logoVariable(),
	}
}
