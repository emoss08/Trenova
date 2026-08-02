package customer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

type CustomerEmailProfile struct {
	bun.BaseModel `bun:"table:customer_email_profiles,alias:cem" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	CustomerID     pulid.ID `json:"customerId"     bun:"customer_id,pk,notnull,type:VARCHAR(100)"`

	Subject               string `json:"subject"                 bun:"subject,type:VARCHAR(255)"`
	Comment               string `json:"comment"                 bun:"comment,type:TEXT"`
	FromEmail             string `json:"fromEmail"               bun:"from_email,type:VARCHAR(255)"`
	ToRecipients          string `json:"toRecipients"            bun:"to_recipients,type:TEXT"`
	CCRecipients          string `json:"ccRecipients"            bun:"cc_recipients,type:TEXT,nullzero"`
	BCCRecipients         string `json:"bccRecipients"           bun:"bcc_recipients,type:TEXT,nullzero"`
	AttachmentName        string `json:"attachmentName"          bun:"attachment_name,type:VARCHAR(255)"`
	ReadReceipt           bool   `json:"readReceipt"           bun:"read_receipt,type:BOOLEAN,notnull,default:false"`
	IncludeShipmentDetail bool   `json:"includeShipmentDetail" bun:"include_shipment_detail,type:BOOLEAN,notnull,default:false"`
	Version               int64  `json:"version"               bun:"version,type:BIGINT"`
	CreatedAt             int64  `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64  `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

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
	// Tenancy and the owning customer are stamped when the customer is
	// written, so they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Subject,
			validation.Length(0, maxEmailSubjectLength).
				Error("Subject cannot be longer than 255 characters"),
		),
		validation.Field(&p.FromEmail,
			validation.Required.Error("From email is required"),
			is.EmailFormat.Error("From email must be a valid email address"),
			validation.Length(1, maxEmailAddressLength).
				Error("From email cannot be longer than 255 characters"),
		),
		// The recipient columns are comma separated lists, so an empty element
		// would address a message to nobody halfway down the list.
		validation.Field(&p.ToRecipients,
			validation.Required.Error("At least one recipient is required"),
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
	maxEmailAddressLength   = 255
	maxAttachmentNameLength = 255
)
