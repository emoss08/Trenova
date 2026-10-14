package documenttemplate

import "html/template"

// FactoringNoaContext is a notice of assignment: the letter in which a carrier
// tells its customer that its invoices are assigned to a factor, that they must
// be paid only to the factor's remittance address, and asks the customer to sign
// and return the acknowledgement printed beneath it.
type FactoringNoaContext struct {
	LetterDate    string
	EffectiveDate string
	Reference     string
	Organization  AddressBlock
	Customer      AddressBlock
	Factor        AddressBlock
	RemitTo       AddressBlock
	// LogoDataURI is typed as trusted for the reason InvoiceContext gives: it
	// is produced by the asset inliner from bytes it has already decoded.
	LogoDataURI template.URL
	Note        string
	SignerName  string
	SignerTitle string
}

// FactoringNoaEmailContext is the message a notice of assignment is attached to.
type FactoringNoaEmailContext struct {
	CustomerName   string
	CompanyName    string
	FactorName     string
	RemitTo        AddressBlock
	EffectiveDate  string
	Note           string
	AttachmentName string
	ReplyTo        string
}

const (
	factoringEffectiveDatePath = "EffectiveDate"
	factoringRemitToPath       = "RemitTo"
	sampleFactorName           = "Lone Star Freight Funding, LLC"
	sampleNoaEffectiveDate     = "2026-09-01"
	sampleNoaNote              = "Invoices dated before September 1 that are still open are included."
	organizationPath           = "Organization"
	attachmentNamePath         = "AttachmentName"
	sampleCountry              = "United States"
	sampleDallasCityLine       = "Dallas, TX 75201"
	sampleNoaSigner            = "Dana Whitfield"
	sampleACHLabel             = "ACH"
)

func sampleFactorRemitTo() AddressBlock {
	return AddressBlock{
		Name:  sampleFactorName,
		Lines: []string{"PO Box 660412", "Dallas, TX 75266", sampleCountry},
		Details: []KeyValue{
			{Label: sampleACHLabel, Value: "Routing 111000025 / Account 5520193377"},
		},
	}
}

func newFactoringNoaSampleContext() any {
	return FactoringNoaContext{
		LetterDate:    "2026-08-24",
		EffectiveDate: sampleNoaEffectiveDate,
		Reference:     "noa_01K3R8V6Q2M4ZJ7XW9T5B0C1DE",
		Organization: AddressBlock{
			Name:  sampleCompanyName,
			Lines: []string{"2200 Commerce St", sampleDallasCityLine, sampleCountry},
			Details: []KeyValue{
				{Label: "MC", Value: "MC-884120"},
				{Label: "USDOT", Value: "3120457"},
			},
		},
		Customer: AddressBlock{
			Name:  sampleCustomerName,
			Lines: []string{"410 Market Ave", "Fort Worth, TX 76102", sampleCountry},
			Details: []KeyValue{
				{Label: "AP Contact", Value: "ap@halstead.example"},
			},
		},
		Factor: AddressBlock{
			Name:  sampleFactorName,
			Lines: []string{"1700 Pacific Ave, Suite 2100", sampleDallasCityLine, sampleCountry},
			Details: []KeyValue{
				{Label: "Phone", Value: "(214) 555-0187"},
				{Label: "Email", Value: "verifications@lonestarfunding.example"},
			},
		},
		RemitTo:     sampleFactorRemitTo(),
		LogoDataURI: sampleLogoDataURI,
		Note:        sampleNoaNote,
		SignerName:  sampleNoaSigner,
		SignerTitle: "Controller",
	}
}

func newFactoringNoaEmailSampleContext() any {
	return FactoringNoaEmailContext{
		CustomerName:   sampleCustomerName,
		CompanyName:    sampleCompanyName,
		FactorName:     sampleFactorName,
		RemitTo:        sampleFactorRemitTo(),
		EffectiveDate:  sampleNoaEffectiveDate,
		Note:           sampleNoaNote,
		AttachmentName: "Notice-of-Assignment-HALSTEAD-2026-09-01.pdf",
		ReplyTo:        "billing@trenova.example",
	}
}

func factoringNoteVariable() VariableDefinition {
	return VariableDefinition{
		Path:        notePath,
		Type:        VariableString,
		Description: "What the person sending the notice wrote. Print it when present.",
	}
}

//nolint:funlen // A variable catalog is one declaration; splitting it only hides it.
func (r *Registry) registerFactoringNoaKinds() {
	_ = r.Register(&KindDefinition{
		Kind:           KindFactoringNoaPDF,
		DisplayName:    "Notice of Assignment",
		Description:    "The letter telling a customer that, from the effective date, every invoice is assigned to your factoring company and must be paid only to the factor's remittance address, with an acknowledgement for the customer to sign and return.",
		Category:       billingCategory,
		Channels:       []Channel{ChannelPDF},
		Paged:          true,
		CustomerScoped: true,
		sampleFactory:  newFactoringNoaSampleContext,
		Variables: []VariableDefinition{
			{
				Path:        "LetterDate",
				Type:        VariableDate,
				Description: "The day the notice is dated, in the organization timezone.",
			},
			{
				Path:        factoringEffectiveDatePath,
				Type:        VariableDate,
				Required:    true,
				Description: "The day the assignment takes effect. Required: a customer cannot know which payments belong to the factor without it.",
			},
			{
				Path:        "Reference",
				Type:        VariableString,
				Description: "The notice's identifier, printed small so the factor can match a returned acknowledgement to it.",
			},
			{
				Path:        organizationPath,
				Type:        VariableObject,
				Description: "Your organization, as the party assigning its invoices.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        customerPath,
				Type:        VariableObject,
				Required:    true,
				Description: "The customer the notice is addressed to. Required: a notice with no addressee does not bind anyone to pay the factor.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        "Factor",
				Type:        VariableObject,
				Required:    true,
				Description: "The factoring company the invoices are assigned to. Required: the notice exists to name it.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        factoringRemitToPath,
				Type:        VariableObject,
				Required:    true,
				Description: "The factor's remittance address, the only place payment may be sent. Required: without it the customer cannot pay the factor and a payment sent elsewhere is lost.",
				Fields:      addressBlockFields(),
			},
			logoVariable(),
			factoringNoteVariable(),
			{
				Path:        "SignerName",
				Type:        VariableString,
				Description: "Who signs the notice for your organization. Blank when no one is named.",
			},
			{
				Path:        "SignerTitle",
				Type:        VariableString,
				Description: "The signer's job title. Blank when no one is named.",
			},
		},
	})

	_ = r.Register(&KindDefinition{
		Kind:           KindFactoringNoaEmail,
		DisplayName:    "Notice of Assignment Email",
		Description:    "The message a notice of assignment is attached to.",
		Category:       billingCategory,
		Channels:       []Channel{ChannelSubject, ChannelEmailHTML, ChannelEmailText},
		CustomerScoped: true,
		sampleFactory:  newFactoringNoaEmailSampleContext,
		Variables: []VariableDefinition{
			customerNameVariable(false, "The customer the notice is addressed to."),
			companyNameVariable(),
			{
				Path:        "FactorName",
				Type:        VariableString,
				Required:    true,
				Description: "The factoring company the invoices are assigned to. Required: the message exists to tell the customer who to pay.",
			},
			{
				Path:        factoringRemitToPath,
				Type:        VariableObject,
				Description: "The factor's remittance address, the only place payment may be sent.",
				Fields:      addressBlockFields(),
			},
			{
				Path:        factoringEffectiveDatePath,
				Type:        VariableDate,
				Description: "The day the assignment takes effect, in the organization timezone.",
			},
			factoringNoteVariable(),
			{
				Path:        attachmentNamePath,
				Type:        VariableString,
				Required:    true,
				Description: "The file name of the attached notice. Required: the customer is asked to sign and return the attachment, so the message must name it.",
			},
			{
				Path:        "ReplyTo",
				Type:        VariableString,
				Description: "The address replies and the signed acknowledgement go to.",
			},
		},
	})
}
