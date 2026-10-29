package factoringtemplate

import (
	"html/template"

	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/templatecatalog"
)

// FactoringNoaContext is a notice of assignment: the letter in which a carrier
// tells its customer that its invoices are assigned to a factor, that they must
// be paid only to the factor's remittance address, and asks the customer to sign
// and return the acknowledgement printed beneath it.
type FactoringNoaContext struct {
	LetterDate    string
	EffectiveDate string
	Reference     string
	Organization  templatecatalog.AddressBlock
	Customer      templatecatalog.AddressBlock
	Factor        templatecatalog.AddressBlock
	RemitTo       templatecatalog.AddressBlock
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
	RemitTo        templatecatalog.AddressBlock
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

func sampleCarrierOrganization() templatecatalog.AddressBlock {
	return templatecatalog.AddressBlock{
		Name:  templatecatalog.SampleCompanyName,
		Lines: []string{"2200 Commerce St", sampleDallasCityLine, sampleCountry},
		Details: []templatecatalog.KeyValue{
			{Label: "MC", Value: "MC-884120"},
			{Label: "USDOT", Value: "3120457"},
		},
	}
}

func sampleFactorRemitTo() templatecatalog.AddressBlock {
	return templatecatalog.AddressBlock{
		Name:  sampleFactorName,
		Lines: []string{"PO Box 660412", "Dallas, TX 75266", sampleCountry},
		Details: []templatecatalog.KeyValue{
			{Label: sampleACHLabel, Value: "Routing 111000025 / Account 5520193377"},
		},
	}
}

func newFactoringNoaSampleContext() any {
	return FactoringNoaContext{
		LetterDate:    "2026-08-24",
		EffectiveDate: sampleNoaEffectiveDate,
		Reference:     "noa_01K3R8V6Q2M4ZJ7XW9T5B0C1DE",
		Organization:  sampleCarrierOrganization(),
		Customer: templatecatalog.AddressBlock{
			Name:  templatecatalog.SampleCustomerName,
			Lines: []string{"410 Market Ave", "Fort Worth, TX 76102", sampleCountry},
			Details: []templatecatalog.KeyValue{
				{Label: "AP Contact", Value: "ap@halstead.example"},
			},
		},
		Factor: templatecatalog.AddressBlock{
			Name:  sampleFactorName,
			Lines: []string{"1700 Pacific Ave, Suite 2100", sampleDallasCityLine, sampleCountry},
			Details: []templatecatalog.KeyValue{
				{Label: "Phone", Value: "(214) 555-0187"},
				{Label: "Email", Value: "verifications@lonestarfunding.example"},
			},
		},
		RemitTo:     sampleFactorRemitTo(),
		LogoDataURI: templatecatalog.SampleLogoDataURI,
		Note:        sampleNoaNote,
		SignerName:  sampleNoaSigner,
		SignerTitle: "Controller",
	}
}

func newFactoringNoaEmailSampleContext() any {
	return FactoringNoaEmailContext{
		CustomerName:   templatecatalog.SampleCustomerName,
		CompanyName:    templatecatalog.SampleCompanyName,
		FactorName:     sampleFactorName,
		RemitTo:        sampleFactorRemitTo(),
		EffectiveDate:  sampleNoaEffectiveDate,
		Note:           sampleNoaNote,
		AttachmentName: "Notice-of-Assignment-HALSTEAD-2026-09-01.pdf",
		ReplyTo:        "billing@trenova.example",
	}
}

func factoringNoteVariable() templatecatalog.VariableDefinition {
	return templatecatalog.VariableDefinition{
		Path:        templatecatalog.NotePath,
		Type:        templatecatalog.VariableString,
		Description: "What the person sending the notice wrote. Print it when present.",
	}
}

//nolint:funlen // A variable catalog is one declaration; splitting it only hides it.
func RegisterFactoringNoaKinds(r *templatecatalog.Registry) {
	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:           documenttemplate.KindFactoringNoaPDF,
		DisplayName:    "Notice of Assignment",
		Description:    "The letter telling a customer that, from the effective date, every invoice is assigned to your factoring company and must be paid only to the factor's remittance address, with an acknowledgement for the customer to sign and return.",
		Category:       templatecatalog.BillingCategory,
		Channels:       []documenttemplate.Channel{documenttemplate.ChannelPDF},
		Paged:          true,
		CustomerScoped: true,
		SampleFactory:  newFactoringNoaSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			{
				Path:        "LetterDate",
				Type:        templatecatalog.VariableDate,
				Description: "The day the notice is dated, in the organization timezone.",
			},
			{
				Path:        factoringEffectiveDatePath,
				Type:        templatecatalog.VariableDate,
				Required:    true,
				Description: "The day the assignment takes effect. Required: a customer cannot know which payments belong to the factor without it.",
			},
			{
				Path:        "Reference",
				Type:        templatecatalog.VariableString,
				Description: "The notice's identifier, printed small so the factor can match a returned acknowledgement to it.",
			},
			{
				Path:        organizationPath,
				Type:        templatecatalog.VariableObject,
				Description: "Your organization, as the party assigning its invoices.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        templatecatalog.CustomerPath,
				Type:        templatecatalog.VariableObject,
				Required:    true,
				Description: "The customer the notice is addressed to. Required: a notice with no addressee does not bind anyone to pay the factor.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        "Factor",
				Type:        templatecatalog.VariableObject,
				Required:    true,
				Description: "The factoring company the invoices are assigned to. Required: the notice exists to name it.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        factoringRemitToPath,
				Type:        templatecatalog.VariableObject,
				Required:    true,
				Description: "The factor's remittance address, the only place payment may be sent. Required: without it the customer cannot pay the factor and a payment sent elsewhere is lost.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			templatecatalog.LogoVariable(),
			factoringNoteVariable(),
			{
				Path:        "SignerName",
				Type:        templatecatalog.VariableString,
				Description: "Who signs the notice for your organization. Blank when no one is named.",
			},
			{
				Path:        "SignerTitle",
				Type:        templatecatalog.VariableString,
				Description: "The signer's job title. Blank when no one is named.",
			},
		},
	})

	_ = r.Register(&templatecatalog.KindDefinition{
		Kind:        documenttemplate.KindFactoringNoaEmail,
		DisplayName: "Notice of Assignment Email",
		Description: "The message a notice of assignment is attached to.",
		Category:    templatecatalog.BillingCategory,
		Channels: []documenttemplate.Channel{
			documenttemplate.ChannelSubject,
			documenttemplate.ChannelEmailHTML,
			documenttemplate.ChannelEmailText,
		},
		CustomerScoped: true,
		SampleFactory:  newFactoringNoaEmailSampleContext,
		Variables: []templatecatalog.VariableDefinition{
			templatecatalog.CustomerNameVariable(false, "The customer the notice is addressed to."),
			templatecatalog.CompanyNameVariable(),
			{
				Path:        "FactorName",
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The factoring company the invoices are assigned to. Required: the message exists to tell the customer who to pay.",
			},
			{
				Path:        factoringRemitToPath,
				Type:        templatecatalog.VariableObject,
				Description: "The factor's remittance address, the only place payment may be sent.",
				Fields:      templatecatalog.AddressBlockFields(),
			},
			{
				Path:        factoringEffectiveDatePath,
				Type:        templatecatalog.VariableDate,
				Description: "The day the assignment takes effect, in the organization timezone.",
			},
			factoringNoteVariable(),
			{
				Path:        attachmentNamePath,
				Type:        templatecatalog.VariableString,
				Required:    true,
				Description: "The file name of the attached notice. Required: the customer is asked to sign and return the attachment, so the message must name it.",
			},
			{
				Path:        "ReplyTo",
				Type:        templatecatalog.VariableString,
				Description: "The address replies and the signed acknowledgement go to.",
			},
		},
	})
}
