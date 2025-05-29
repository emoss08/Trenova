package customer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Customer)(nil)
	_ domain.Validatable        = (*Customer)(nil)
	_ infra.PostgresSearchable  = (*Customer)(nil)
)

type Customer struct {
	bun.BaseModel `bun:"table:customers,alias:cus" json:"-"`

	ID             pulid.ID      `json:"id"             bun:",pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID      `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID      `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	StateID        pulid.ID      `json:"stateId"        bun:"state_id,notnull,type:VARCHAR(100)"`
	Status         domain.Status `json:"status"         bun:"status,type:status_enum,notnull,default:'Active'"`
	Code           string        `json:"code"           bun:"code,type:VARCHAR(10),notnull"`
	Name           string        `json:"name"           bun:"name,type:VARCHAR(255),notnull"`
	AddressLine1   string        `json:"addressLine1"   bun:"address_line_1,type:VARCHAR(150),notnull"`
	AddressLine2   string        `json:"addressLine2"   bun:"address_line_2,type:VARCHAR(150)"`
	City           string        `json:"city"           bun:"city,type:VARCHAR(100),notnull"`
	PostalCode     string        `json:"postalCode"     bun:"postal_code,type:us_postal_code,notnull"`
	SearchVector   string        `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank           string        `json:"-"              bun:"rank,type:VARCHAR(100),scanonly"`
	Version        int64         `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64         `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64         `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit   *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization   *organization.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
	BillingProfile *BillingProfile            `bun:"rel:has-one,join:id=customer_id"         json:"billingProfile,omitempty"`
	EmailProfile   *CustomerEmailProfile      `bun:"rel:has-one,join:id=customer_id"         json:"emailProfile,omitempty"`
	State          *usstate.UsState           `bun:"rel:belongs-to,join:state_id=id"         json:"state,omitempty"`
}

func (c *Customer) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, c,
		// * Code is required and must be within 1 and 10 characters.
		validation.Field(&c.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 10).Error("Code must be between 1 and 100 characters"),
		),

		// * Name is required and must be within 1 and 255 characters.
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),

		// * Address Line 1 is required and must be within 1 and 150 characters.
		validation.Field(&c.AddressLine1,
			validation.Required.Error("Address Line 1 is required"),
			validation.Length(1, 150).Error("Address Line 1 must be between 1 and 150 characters"),
		),

		// * City is required and must be within 1 and 100 characters.
		validation.Field(&c.City,
			validation.Required.Error("City is required"),
			validation.Length(1, 100).Error("City must be between 1 and 100 characters"),
		),

		// * State is required.
		validation.Field(&c.StateID,
			validation.Required.Error("State is required"),
		),

		// * Postal Code is required and must be a valid US or Canadian postal code.
		validation.Field(&c.PostalCode,
			validation.Required.Error("Postal Code is required"),
			validation.By(domain.ValidatePostalCode),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Pagination Configuration
func (c *Customer) GetID() string {
	return c.ID.String()
}

func (c *Customer) GetTableName() string {
	return "customers"
}

func (c *Customer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("cus_")
		}

		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

func (c *Customer) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "cus",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "code",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "name",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}

// Miscellaneous
func (c *Customer) HasBillingProfile() bool {
	return c.BillingProfile != nil
}

func (c *Customer) HasEmailProfile() bool {
	return c.EmailProfile != nil
}
