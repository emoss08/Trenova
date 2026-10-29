package customer

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/mailtext"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CustomerEmailProfile struct {
	keel.Table `keel:"customer_email_profiles"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:cem_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	CustomerID     pulid.ID `json:"customerId"     keel:"customer_id,pk"`

	Subject               string `json:"subject"               keel:"subject,nullzero"`
	Comment               string `json:"comment"               keel:"comment,nullzero"`
	ToRecipients          string `json:"toRecipients"          keel:"to_recipients,nullzero"`
	CCRecipients          string `json:"ccRecipients"          keel:"cc_recipients,nullzero"`
	BCCRecipients         string `json:"bccRecipients"         keel:"bcc_recipients,nullzero"`
	AttachmentName        string `json:"attachmentName"        keel:"attachment_name,nullzero"`
	ReadReceipt           bool   `json:"readReceipt"           keel:"read_receipt"`
	IncludeShipmentDetail bool   `json:"includeShipmentDetail" keel:"include_shipment_detail"`
	Version               int64  `json:"version"               keel:"version,version"`
	CreatedAt             int64  `json:"createdAt"             keel:"created_at,autocreate"`
	UpdatedAt             int64  `json:"updatedAt"             keel:"updated_at,autoupdate"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func (e *CustomerEmailProfile) GetID() string {
	return e.ID.String()
}

func (e *CustomerEmailProfile) GetTableName() string {
	return "customer_email_profiles"
}

func (e *CustomerEmailProfile) GetOrganizationID() pulid.ID {
	return e.OrganizationID
}

func (e *CustomerEmailProfile) GetBusinessUnitID() pulid.ID {
	return e.BusinessUnitID
}

func (p *CustomerEmailProfile) Validate(multiErr *errortypes.MultiError) {
	p.ValidateForDelivery(multiErr, true)
}

func (p *CustomerEmailProfile) ValidateForDelivery(
	multiErr *errortypes.MultiError,
	emailDeliveryEnabled bool,
) {
	// Tenancy and the owning customer are stamped when the customer is
	// written, so they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Subject,
			validation.Length(0, maxEmailSubjectLength).
				Error("Subject cannot be longer than 255 characters"),
		),
		// The recipient columns are comma separated lists, so an empty element
		// would address a message to nobody halfway down the list.
		validation.Field(&p.ToRecipients,
			validation.When(
				emailDeliveryEnabled,
				validation.Required.Error("At least one recipient is required to email invoices"),
			),
			validation.By(domaintypes.ValidateStringOrCommaSeparated),
		),
		validation.Field(&p.CCRecipients,
			validation.By(domaintypes.ValidateStringOrCommaSeparated),
		),
		validation.Field(&p.BCCRecipients,
			validation.By(domaintypes.ValidateStringOrCommaSeparated),
		),
		validation.Field(&p.AttachmentName,
			validation.Length(0, maxAttachmentNameLength).
				Error("Attachment name cannot be longer than 255 characters"),
		),
	))
}

const (
	maxEmailSubjectLength   = 255
	maxAttachmentNameLength = 255
)

// EmailRecipients is who a customer's billing email goes to.
type EmailRecipients struct {
	To  []string
	CC  []string
	BCC []string
}

// Recipients splits the profile's address lists. A nil profile has none.
func (p *CustomerEmailProfile) Recipients() EmailRecipients {
	if p == nil {
		return EmailRecipients{}
	}
	return EmailRecipients{
		To:  mailtext.SplitEmailList(p.ToRecipients),
		CC:  mailtext.SplitEmailList(p.CCRecipients),
		BCC: mailtext.SplitEmailList(p.BCCRecipients),
	}
}
