package customer

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Customer)(nil)
	_ domain.Validatable             = (*Customer)(nil)
	_ framework.TenantedEntity       = (*Customer)(nil)
	_ domaintypes.PostgresSearchable = (*Customer)(nil)
)

type Customer struct {
	bun.BaseModel `bun:"table:customers,alias:cus" json:"-"`

	ID                     pulid.ID      `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID      `json:"businessUnitId"         bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID         pulid.ID      `json:"organizationId"         bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	StateID                pulid.ID      `json:"stateId"                bun:"state_id,notnull,type:VARCHAR(100)"`
	Status                 domain.Status `json:"status"                 bun:"status,type:status_enum,notnull,default:'Active'"`
	Code                   string        `json:"code"                   bun:"code,type:VARCHAR(10),notnull"`
	Name                   string        `json:"name"                   bun:"name,type:VARCHAR(255),notnull"`
	AddressLine1           string        `json:"addressLine1"           bun:"address_line_1,type:VARCHAR(150),notnull"`
	AddressLine2           string        `json:"addressLine2"           bun:"address_line_2,type:VARCHAR(150)"`
	City                   string        `json:"city"                   bun:"city,type:VARCHAR(100),notnull"`
	PostalCode             string        `json:"postalCode"             bun:"postal_code,type:us_postal_code,notnull"`
	SearchVector           string        `json:"-"                      bun:"search_vector,type:TSVECTOR,scanonly"`
	PlaceID                string        `json:"placeId"                bun:"place_id,type:TEXT"`
	ExternalID             string        `json:"externalId"             bun:"external_id,type:TEXT"`
	Rank                   string        `json:"-"                      bun:"rank,type:VARCHAR(100),scanonly"`
	AllowConsolidation     bool          `json:"allowConsolidation"     bun:"allow_consolidation,type:BOOLEAN,default:true"`
	ExclusiveConsolidation bool          `json:"exclusiveConsolidation" bun:"exclusive_consolidation,type:BOOLEAN,default:false"`
	IsGeocoded             bool          `json:"isGeocoded"             bun:"is_geocoded,type:BOOLEAN,default:false"`
	ConsolidationPriority  int           `json:"consolidationPriority"  bun:"consolidation_priority,type:INTEGER,default:1"`
	Version                int64         `json:"version"                bun:"version,type:BIGINT"`
	CreatedAt              int64         `json:"createdAt"              bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64         `json:"updatedAt"              bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Longitude              *float64      `json:"longitude"              bun:"longitude,type:FLOAT,nullzero"`
	Latitude               *float64      `json:"latitude"               bun:"latitude,type:FLOAT,nullzero"`

	// Relationships
	BusinessUnit   *tenant.BusinessUnit    `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization   *tenant.Organization    `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
	BillingProfile *CustomerBillingProfile `bun:"rel:has-one,join:id=customer_id"         json:"billingProfile,omitempty"`
	EmailProfile   *CustomerEmailProfile   `bun:"rel:has-one,join:id=customer_id"         json:"emailProfile,omitempty"`
	State          *usstate.UsState        `bun:"rel:belongs-to,join:state_id=id"         json:"state,omitempty"`
}

func (c *Customer) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(c,
		validation.Field(&c.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 10).Error("Code must be between 1 and 100 characters"),
		),
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&c.AddressLine1,
			validation.Required.Error("Address Line 1 is required"),
			validation.Length(1, 150).Error("Address Line 1 must be between 1 and 150 characters"),
		),
		validation.Field(&c.City,
			validation.Required.Error("City is required"),
			validation.Length(1, 100).Error("City must be between 1 and 100 characters"),
		),
		validation.Field(&c.StateID,
			validation.Required.Error("State is required"),
		),
		validation.Field(&c.PostalCode,
			validation.Required.Error("Postal Code is required"),
			validation.By(domain.ValidatePostalCode),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (c *Customer) HasBillingProfile() bool {
	return c.BillingProfile != nil
}

func (c *Customer) HasEmailProfile() bool {
	return c.EmailProfile != nil
}

func (c *Customer) GetID() string {
	return c.ID.String()
}

func (c *Customer) GetTableName() string {
	return "customers"
}

func (c *Customer) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *Customer) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}

func (c *Customer) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "cus",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "code",
				Weight: domaintypes.SearchWeightA,
				Type:   domaintypes.FieldTypeText,
			},
			{
				Name:   "name",
				Weight: domaintypes.SearchWeightB,
				Type:   domaintypes.FieldTypeText,
			},
		},
	}
}

func (c *Customer) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

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
