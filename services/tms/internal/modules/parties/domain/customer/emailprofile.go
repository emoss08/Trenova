package customer

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type CustomerEmailProfile struct {
	bun.BaseModel `bun:"table:customer_email_profiles,alias:cem" json:"-"`
	keel.Table    `                                                       keel:"customer_email_profiles"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"               keel:"id,pk,autoid:cem_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"  keel:"organization_id,pk,tenant"`
	CustomerID     pulid.ID `json:"customerId"     bun:"customer_id,pk,notnull,type:VARCHAR(100)"      keel:"customer_id,pk"`

	Subject               string `json:"subject"               bun:"subject,type:VARCHAR(255)"                                                            keel:"subject,nullzero"`
	Comment               string `json:"comment"               bun:"comment,type:TEXT"                                                                    keel:"comment,nullzero"`
	ToRecipients          string `json:"toRecipients"          bun:"to_recipients,type:TEXT"                                                              keel:"to_recipients,nullzero"`
	CCRecipients          string `json:"ccRecipients"          bun:"cc_recipients,type:TEXT,nullzero"                                                     keel:"cc_recipients,nullzero"`
	BCCRecipients         string `json:"bccRecipients"         bun:"bcc_recipients,type:TEXT,nullzero"                                                    keel:"bcc_recipients,nullzero"`
	AttachmentName        string `json:"attachmentName"        bun:"attachment_name,type:VARCHAR(255)"                                                    keel:"attachment_name,nullzero"`
	ReadReceipt           bool   `json:"readReceipt"           bun:"read_receipt,type:BOOLEAN,notnull"                                                    keel:"read_receipt"`
	IncludeShipmentDetail bool   `json:"includeShipmentDetail" bun:"include_shipment_detail,type:BOOLEAN,notnull"                                         keel:"include_shipment_detail"`
	Version               int64  `json:"version"               bun:"version,type:BIGINT"                                                                  keel:"version,version"`
	CreatedAt             int64  `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt             int64  `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
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

func (e *CustomerEmailProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("cem_")
		}
		e.CreatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
	}

	return nil
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
		To:  stringutils.SplitEmailList(p.ToRecipients),
		CC:  stringutils.SplitEmailList(p.CCRecipients),
		BCC: stringutils.SplitEmailList(p.BCCRecipients),
	}
}
