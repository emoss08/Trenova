package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*ShipmentCommodity)(nil)
	_ domain.Validatable        = (*ShipmentCommodity)(nil)
)

type ShipmentCommodity struct {
	bun.BaseModel `bun:"table:shipment_commodities,alias:sc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:",pk,type:VARCHAR(100),notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,pk,notnull,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,pk,notnull,type:VARCHAR(100)" json:"organizationId"`
	ShipmentID     pulid.ID `bun:"shipment_id,pk,notnull,type:VARCHAR(100)" json:"shipmentId"`
	CommodityID    pulid.ID `bun:"commodity_id,pk,notnull,type:VARCHAR(100)" json:"commodityId"`

	// Core Fields
	Weight float64 `bun:"weight,type:FLOAT,notnull" json:"weight"`
	Pieces int64   `bun:"pieces,type:INTEGER,notnull" json:"pieces"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `bun:"created_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	Shipment     *Shipment                  `bun:"rel:belongs-to,join:shipment_id=id" json:"-"`
	Commodity    *commodity.Commodity       `bun:"rel:belongs-to,join:commodity_id=id" json:"commodity,omitempty"`
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (sc *ShipmentCommodity) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, sc,
		// Pieces is required
		validation.Field(&sc.Pieces,
			validation.Required.Error("Pieces is required"),
			validation.Min(1).Error("Pieces must be at least 1"),
		),
		// Weight is required
		validation.Field(&sc.Weight,
			validation.Required.Error("Weight is required"),
			validation.Min(0.0).Error("Weight must be greater than or equal to 0"),
		),
		// Commodity ID is required
		validation.Field(&sc.CommodityID,
			validation.Required.Error("Commodity ID is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromValidationErrors(validationErrs, multiErr, "")
		}
	}
}

// Pagination Configuration
func (sc *ShipmentCommodity) GetID() string {
	return sc.ID.String()
}

func (sc *ShipmentCommodity) GetTableName() string {
	return "shipment_commodities"
}

func (sc *ShipmentCommodity) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if sc.ID.IsNil() {
			sc.ID = pulid.MustNew("sc_")
		}

		sc.CreatedAt = now
	case *bun.UpdateQuery:
		sc.UpdatedAt = now
	}

	return nil
}
