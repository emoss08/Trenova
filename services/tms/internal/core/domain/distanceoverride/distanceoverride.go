package distanceoverride

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Override)(nil)
	_ domain.Validatable             = (*Override)(nil)
	_ domaintypes.PostgresSearchable = (*Override)(nil)
	_ framework.TenantedEntity       = (*Override)(nil)
)

type Override struct {
	bun.BaseModel `bun:"table:distance_overrides,alias:diso" json:"-"`

	ID                    pulid.ID  `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID        pulid.ID  `json:"businessUnitId"        bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID        pulid.ID  `json:"organizationId"        bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	OriginLocationID      pulid.ID  `json:"originLocationId"      bun:"origin_location_id,type:VARCHAR(100),notnull"`
	DestinationLocationID pulid.ID  `json:"destinationLocationId" bun:"destination_location_id,type:VARCHAR(100),notnull"`
	CustomerID            *pulid.ID `json:"customerId"            bun:"customer_id,type:VARCHAR(100),nullzero"`
	SearchVector          string    `json:"-"                     bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                  string    `json:"-"                     bun:"rank,type:VARCHAR(100),scanonly"`
	Distance              float64   `json:"distance"              bun:"distance,type:FLOAT,nullzero"`
	Version               int64     `json:"version"               bun:"version,type:BIGINT"`
	CreatedAt             int64     `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64     `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	Customer            *customer.Customer   `json:"customer,omitzero"             bun:"rel:belongs-to,join:customer_id=id"`
	BusinessUnit        *tenant.BusinessUnit `json:"businessUnit,omitempty"        bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization        *tenant.Organization `json:"organization,omitempty"        bun:"rel:belongs-to,join:organization_id=id"`
	OriginLocation      *location.Location   `json:"originLocation,omitempty"      bun:"rel:belongs-to,join:origin_location_id=id"`
	DestinationLocation *location.Location   `json:"destinationLocation,omitempty" bun:"rel:belongs-to,join:destination_location_id=id"`
}

func (o *Override) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(o,
		validation.Field(&o.Distance,
			validation.Required.Error("Distance is required"),
			validation.Min(0.0).Error("Distance must be greater than 0"),
		),
		validation.Field(&o.OriginLocationID,
			validation.Required.Error("Origin location is required"),
		),
		validation.Field(&o.DestinationLocationID,
			validation.Required.Error("Destination location is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (o *Override) GetID() string {
	return o.ID.String()
}

func (o *Override) GetTableName() string {
	return "distance_overrides"
}

func (o *Override) GetBusinessUnitID() pulid.ID {
	return o.BusinessUnitID
}

func (o *Override) GetOrganizationID() pulid.ID {
	return o.OrganizationID
}

func (o *Override) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if o.ID == "" {
			o.ID = pulid.MustNew("do_")
		}

		o.CreatedAt = now
	case *bun.UpdateQuery:
		o.UpdatedAt = now
	}

	return nil
}

func (o *Override) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "do",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "distance",
				Type:   domaintypes.FieldTypeNumber,
				Weight: domaintypes.SearchWeightA,
			},
		},
	}
}
