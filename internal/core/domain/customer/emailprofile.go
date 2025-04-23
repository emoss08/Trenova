package customer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*CustomerEmailProfile)(nil)
	_ domain.Validatable        = (*CustomerEmailProfile)(nil)
)

//nolint:revive // This is a valid struct name
type CustomerEmailProfile struct {
	bun.BaseModel `bun:"table:customer_email_profiles,alias:cem" json:"-"`

	ID             pulid.ID `json:"id" bun:",pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	CustomerID     pulid.ID `json:"customerId" bun:"customer_id,pk,notnull,type:VARCHAR(100)"`
	Subject        string   `json:"subject" bun:"subject,type:VARCHAR(100)"`
	Comment        string   `json:"comment" bun:"comment,type:TEXT"`
	FromEmail      string   `json:"fromEmail" bun:"from_email,type:VARCHAR(255)"`
	BlindCopy      string   `json:"blindCopy" bun:"blind_copy,type:VARCHAR(255)"`
	AttachmentName string   `json:"attachmentName" bun:"attachment_name,type:VARCHAR(255)"`
	ReadReceipt    bool     `json:"readReceipt" bun:"read_receipt,type:BOOLEAN,notnull,default:false"`
	Version        int64    `json:"version" bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (c *CustomerEmailProfile) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, c,
		// * Ensure Customer ID is set
		validation.Field(&c.CustomerID, validation.Required.Error("Customer ID is required")),

		// * Ensure from email is a valid email
		validation.Field(&c.FromEmail, is.Email.Error("From Email must be a valid email address")),

		// * Ensure blind copy is a comma separated list of valid emails
		validation.Field(&c.BlindCopy, validation.By(domain.ValidateCommaSeparatedEmails)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (c *CustomerEmailProfile) GetID() string {
	return c.ID.String()
}

func (c *CustomerEmailProfile) GetTableName() string {
	return "customer_email_profiles"
}

func (c *CustomerEmailProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("cem_")
		}

		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
