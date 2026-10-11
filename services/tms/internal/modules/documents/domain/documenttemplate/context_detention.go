package documenttemplate

import "html/template"

// DetentionNoticeContext is the data a detention notice renders against.
//
// Every number here comes from the frozen policy snapshot on the occurrence, not
// from live configuration. That is what lets the notice and the invoice that follows
// quote identical terms — a customer who receives the same free-time figure twice
// while the truck is on their dock rarely disputes the charge.
type DetentionNoticeContext struct {
	// Kind is Warning, Started, Update, or Final. Branch on it to vary the wording
	// for each stage of the same event.
	Kind string

	ShipmentProNumber string
	FacilityName      string
	CustomerName      string
	CompanyName       string

	ArrivedAt     string
	ArrivalSource string
	DepartedAt    string

	FreeTime          string
	FreeTimeExpiresAt string
	RateLabel         string

	TotalTimeOnSite string
	BillableTime    string
	BillableAmount  string
	Currency        string

	// Tiers is populated when the policy bills in bands rather than at a flat rate,
	// so a template can show the schedule a charge was derived from.
	Tiers []DetentionTierRow

	LogoDataURI template.URL
}

// DetentionTierRow is one band of a tiered detention rate.
type DetentionTierRow struct {
	Band  string
	Rate  string
	Label string
}

func newDetentionNoticeSampleContext() any {
	return DetentionNoticeContext{
		Kind:              "Final",
		ShipmentProNumber: sampleProNumber,
		FacilityName:      sampleFacilityName,
		CustomerName:      sampleCustomerName,
		CompanyName:       sampleCompanyName,
		ArrivedAt:         "Sat 11 Jul 2026 09:14 CDT",
		ArrivalSource:     "Telematics geofence",
		DepartedAt:        "Sat 11 Jul 2026 14:42 CDT",
		FreeTime:          "2h",
		FreeTimeExpiresAt: "Sat 11 Jul 2026 11:14 CDT",
		RateLabel:         "65.00/hr after free time",
		TotalTimeOnSite:   "5h 28m",
		BillableTime:      "3h 30m",
		BillableAmount:    "227.50",
		Currency:          sampleCurrency,
		Tiers: []DetentionTierRow{
			{Band: "2h–4h", Rate: "65.00/hr", Label: "Standard"},
			{Band: "4h and beyond", Rate: "95.00/hr", Label: "Extended"},
		},
		//nolint:gosec // A compile-time constant data: URI; see the field's doc comment.
		LogoDataURI: template.URL(sampleLogoDataURI),
	}
}

func (r *Registry) registerDetentionKinds() {
	variables := detentionNoticeVariables()

	_ = r.Register(&KindDefinition{
		Kind:        KindDetentionNoticeEmail,
		DisplayName: "Detention Notice Email",
		Description: "Sent to a customer while a driver is detained, and again once the " +
			"charge is final. Branch on Kind to vary the wording per stage.",
		Category:       "Detention",
		Channels:       []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		CustomerScoped: true,
		sampleFactory:  newDetentionNoticeSampleContext,
		Variables:      variables,
	})

	_ = r.Register(&KindDefinition{
		Kind:        KindDetentionNoticePDF,
		DisplayName: "Detention Notice Document",
		Description: "The printable notice attached to the email, which is the artifact " +
			"a customer is shown when a detention charge is disputed.",
		Category:       "Detention",
		Channels:       []Channel{ChannelPDF},
		Paged:          true,
		CustomerScoped: true,
		sampleFactory:  newDetentionNoticeSampleContext,
		Variables:      variables,
	})
}

func detentionNoticeVariables() []VariableDefinition {
	return []VariableDefinition{
		{
			Path:        "Kind",
			Type:        VariableString,
			Required:    true,
			Description: "Which stage this notice is: Warning, Started, Update, or Final. Required, because one template covers all four and the recipient must be able to tell them apart.",
		},
		proNumberVariable(
			true,
			"The shipment being detained. A notice that does not identify the load cannot be acted on.",
		),
		{
			Path:        "FacilityName",
			Type:        VariableString,
			Required:    true,
			Description: "Where the driver is waiting. Required: the customer needs to know which of their docks to expedite.",
		},
		customerNameVariable(false, "The customer responsible for the detention."),
		companyNameVariable(),
		{
			Path:        "ArrivedAt",
			Type:        VariableDateTime,
			Description: "When the driver arrived, with weekday and timezone. This is the figure most often disputed, so it is stated in full.",
		},
		{
			Path:        "ArrivalSource",
			Type:        VariableString,
			Description: "How arrival was determined, such as a telematics geofence or a manual entry. Naming the source is what makes the timestamp hard to argue with.",
		},
		{
			Path:        "DepartedAt",
			Type:        VariableDateTime,
			Description: "When the driver left. Blank while the clock is still running.",
		},
		{Path: "FreeTime", Type: VariableString, Description: "Contracted free time, such as 2h."},
		{
			Path:        "FreeTimeExpiresAt",
			Type:        VariableDateTime,
			Description: "The moment free time runs out.",
		},
		{
			Path:        "RateLabel",
			Type:        VariableString,
			Description: "The rate that applies after free time, in words.",
		},
		{
			Path:        "TotalTimeOnSite",
			Type:        VariableString,
			Description: "Raw dwell time, before rounding.",
		},
		{
			Path:        "BillableTime",
			Type:        VariableString,
			Description: "Dwell time after free time and rounding are applied.",
		},
		{
			Path:        "BillableAmount",
			Type:        VariableString,
			Description: "The detention charge, unformatted. Combine with Currency, or use the money function.",
		},
		{Path: "Currency", Type: VariableString, Description: "Currency of the charge."},
		logoVariable(),
		{
			Path:        "Tiers",
			Type:        VariableCollection,
			Description: "The rate bands, when the policy bills in tiers rather than at a flat rate. Empty for a flat rate.",
			Fields: []VariableDefinition{
				{
					Path:        "Band",
					Type:        VariableString,
					Description: "The time range this tier covers.",
				},
				{Path: "Rate", Type: VariableString, Description: "The rate charged in this band."},
				{
					Path:        "Label",
					Type:        VariableString,
					Description: "The tier's name, if the policy gave it one.",
				},
			},
		},
	}
}
