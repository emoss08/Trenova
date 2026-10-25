// Package factoringcompany is a factor: the company an organization sells its
// receivables to. A customer whose invoices are factored pays the factor, so
// its invoices are remitted to the factor and carry a notice of assignment.
package factoringcompany

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/addressutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	MaxRemittanceInstructions = 2000
	MaxNoticeOfAssignment     = 2000
)

var (
	_ validationframework.TenantedEntity = (*FactoringCompany)(nil)
	_ domaintypes.PostgresSearchable     = (*FactoringCompany)(nil)
	_ pagination.CursorEntity            = (*FactoringCompany)(nil)
)

// Factors that buy customer invoices and the remittance details invoices carry to them
type FactoringCompany struct {
	keel.Table `keel:"factoring_companies,report=Parties"`

	ID                     pulid.ID           `json:"id"                     keel:"id,pk,autoid:fac_"`
	BusinessUnitID         pulid.ID           `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID           `json:"organizationId"         keel:"organization_id,pk,tenant"`
	Status                 domaintypes.Status `json:"status"                 keel:"status"`
	Code                   string             `json:"code"                   keel:"code"`
	Name                   string             `json:"name"                   keel:"name"`
	AddressLine1           string             `json:"addressLine1"           keel:"address_line_1"`
	AddressLine2           string             `json:"addressLine2"           keel:"address_line_2,nullzero"`
	City                   string             `json:"city"                   keel:"city"`
	StateID                pulid.ID           `json:"stateId"                keel:"state_id"`
	PostalCode             string             `json:"postalCode"             keel:"postal_code"`
	Email                  string             `json:"email"                  keel:"email,nullzero"`
	PhoneNumber            string             `json:"phoneNumber"            keel:"phone_number,nullzero"`
	RemittanceInstructions string             `json:"remittanceInstructions" keel:"remittance_instructions,nullzero,noreport"`
	NoticeOfAssignment     string             `json:"noticeOfAssignment"     keel:"notice_of_assignment,nullzero,noreport"`
	NonRecourse            bool               `json:"nonRecourse"            keel:"non_recourse,label=Non-Recourse"`
	Notes                  string             `json:"notes"                  keel:"notes,nullzero,noreport"`
	SearchVector           string             `json:"-"                      keel:"search_vector,tsvector,readonly,noselect"`
	Version                int64              `json:"version"                keel:"version,version"`
	CreatedAt              int64              `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64              `json:"updatedAt"              keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
	State        *usstate.UsState     `json:"state,omitempty" keel:"belongs:state_id,noreport"`
}

func (f *FactoringCompany) BeforeUpdate(_ context.Context) error {
	f.Code = strings.ToUpper(strings.TrimSpace(f.Code))
	f.Email = strings.TrimSpace(f.Email)

	return nil
}

func (f *FactoringCompany) BeforeCreate(_ context.Context) error {
	f.Code = strings.ToUpper(strings.TrimSpace(f.Code))
	f.Email = strings.TrimSpace(f.Email)
	if f.Status == "" {
		f.Status = domaintypes.StatusActive
	}

	return nil
}

func (f *FactoringCompany) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(f,
		validation.Field(
			&f.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Invalid status"),
		),
		validation.Field(&f.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 10).Error("Code must be between 1 and 10 characters"),
		),
		validation.Field(&f.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&f.AddressLine1,
			validation.Required.Error("Address line 1 is required"),
			validation.Length(1, 150).Error("Address line 1 must be between 1 and 150 characters"),
		),
		validation.Field(&f.AddressLine2,
			validation.Length(0, 150).Error("Address line 2 must be at most 150 characters"),
		),
		validation.Field(&f.City,
			validation.Required.Error("City is required"),
			validation.Length(1, 100).Error("City must be between 1 and 100 characters"),
		),
		validation.Field(&f.StateID, validation.Required.Error("State is required")),
		validation.Field(&f.PostalCode,
			validation.Required.Error("Postal code is required"),
			validation.By(domaintypes.ValidatePostalCode),
		),
		validation.Field(&f.Email,
			validation.Length(0, 255).Error("Email must be at most 255 characters"),
			is.EmailFormat.Error("Enter a valid email address"),
		),
		validation.Field(&f.PhoneNumber,
			validation.Length(0, 30).Error("Phone number must be at most 30 characters"),
		),
		validation.Field(&f.RemittanceInstructions,
			validation.Length(0, MaxRemittanceInstructions).
				Error("Remittance instructions must be at most 2,000 characters"),
		),
		validation.Field(&f.NoticeOfAssignment,
			validation.Length(0, MaxNoticeOfAssignment).
				Error("The notice of assignment must be at most 2,000 characters"),
		),
	))
}

// IsActive reports whether invoices may be newly assigned to the factor.
func (f *FactoringCompany) IsActive() bool {
	return f != nil && f.Status == domaintypes.StatusActive
}

// MailingAddress is the factor's address as a document prints it.
func (f *FactoringCompany) MailingAddress() addressutils.Mailing {
	if f == nil {
		return addressutils.Mailing{}
	}
	m := addressutils.Mailing{
		Line1:      f.AddressLine1,
		Line2:      f.AddressLine2,
		City:       f.City,
		PostalCode: f.PostalCode,
	}
	if f.State != nil {
		m.State = f.State.Abbreviation
		m.Country = f.State.CountryName
	}
	return m
}

// RemitLines are what a document prints under the factor's name where it
// says where to pay: the address, the remittance instructions, then how to
// reach the factor.
func (f *FactoringCompany) RemitLines() []string {
	if f == nil {
		return nil
	}
	lines := f.MailingAddress().Lines()
	lines = append(lines, strings.Split(f.RemittanceInstructions, "\n")...)
	if email := strings.TrimSpace(f.Email); email != "" {
		lines = append(lines, "Email: "+email)
	}
	if phone := strings.TrimSpace(f.PhoneNumber); phone != "" {
		lines = append(lines, "Phone: "+phone)
	}
	return stringutils.FilterEmpty(lines)
}

// RemittanceText is the remit-to as one block of text, for a message rather
// than a document: who to pay, then RemitLines.
func (f *FactoringCompany) RemittanceText() string {
	if f == nil {
		return ""
	}
	return strings.Join(append([]string{"Remit to " + f.Name}, f.RemitLines()...), "\n")
}

// Notice is the notice of assignment an invoice assigned to the factor
// prints: the factor's own wording, or the standard notice naming it.
func (f *FactoringCompany) Notice() string {
	if f == nil {
		return ""
	}
	if notice := strings.TrimSpace(f.NoticeOfAssignment); notice != "" {
		return notice
	}
	return StandardNotice(f.Name)
}

// StandardNotice is the notice of assignment printed when a factor has no
// wording of its own.
func StandardNotice(factorName string) string {
	return "NOTICE OF ASSIGNMENT: This invoice has been assigned to, and is payable only to, " +
		strings.TrimSpace(factorName) +
		". Payment to any other party will not discharge your obligation. " +
		"Remit payment to the address below and direct any dispute to " +
		strings.TrimSpace(factorName) + "."
}

func (f *FactoringCompany) GetID() pulid.ID {
	return f.ID
}

func (f *FactoringCompany) GetCreatedAt() int64 {
	return f.CreatedAt
}

func (f *FactoringCompany) GetTableName() string {
	return "factoring_companies"
}

func (f *FactoringCompany) GetOrganizationID() pulid.ID {
	return f.OrganizationID
}

func (f *FactoringCompany) GetBusinessUnitID() pulid.ID {
	return f.BusinessUnitID
}

func (f *FactoringCompany) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "fac",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "city", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
			{Name: "email", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (f *FactoringCompany) BeforeSave(context.Context) error {
	f.Code = strings.ToUpper(strings.TrimSpace(f.Code))
	f.Email = strings.TrimSpace(f.Email)

	return nil
}
