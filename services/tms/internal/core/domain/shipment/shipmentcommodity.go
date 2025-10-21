package shipment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*ShipmentCommodity)(nil)
	_ domain.Validatable        = (*ShipmentCommodity)(nil)
	_ framework.TenantedEntity  = (*ShipmentCommodity)(nil)
)

//nolint:revive // valid struct name
type ShipmentCommodity struct {
	bun.BaseModel `bun:"table:shipment_commodities,alias:sc" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	ShipmentID     pulid.ID `json:"shipmentId"     bun:"shipment_id,pk,notnull,type:VARCHAR(100)"`
	CommodityID    pulid.ID `json:"commodityId"    bun:"commodity_id,pk,notnull,type:VARCHAR(100)"`
	Weight         int64    `json:"weight"         bun:"weight,type:INTEGER,notnull"`
	Pieces         int64    `json:"pieces"         bun:"pieces,type:INTEGER,notnull"`
	Version        int64    `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	Shipment     *Shipment            `json:"-"                   bun:"rel:belongs-to,join:shipment_id=id"`
	Commodity    *commodity.Commodity `json:"commodity,omitempty" bun:"rel:belongs-to,join:commodity_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"-"                   bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"                   bun:"rel:belongs-to,join:organization_id=id"`
}

func (sc *ShipmentCommodity) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(sc,
		validation.Field(&sc.Pieces,
			validation.Required.Error("Pieces is required"),
			validation.Min(1).Error("Pieces must be at least 1"),
		),
		validation.Field(&sc.Weight,
			validation.Required.Error("Weight is required"),
			validation.Min(1).Error("Weight must be at least 1"),
		),
		validation.Field(&sc.CommodityID,
			validation.Required.Error("Commodity ID is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (sc *ShipmentCommodity) GetID() string {
	return sc.ID.String()
}

func (sc *ShipmentCommodity) GetOrganizationID() pulid.ID {
	return sc.OrganizationID
}

func (sc *ShipmentCommodity) GetBusinessUnitID() pulid.ID {
	return sc.BusinessUnitID
}

func (sc *ShipmentCommodity) GetTableName() string {
	return "shipment_commodities"
}

func (sc *ShipmentCommodity) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

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

func (sc *ShipmentCommodity) MeetsLinearFeetPerUnitRequirement() bool {
	if sc.Commodity == nil || sc.Commodity.LinearFeetPerUnit == nil {
		return false
	}

	return sc.Pieces > 0 && *sc.Commodity.LinearFeetPerUnit > 0
}
