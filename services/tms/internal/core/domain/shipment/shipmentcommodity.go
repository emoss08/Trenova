/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/commodity"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*ShipmentCommodity)(nil)
	_ domain.Validatable        = (*ShipmentCommodity)(nil)
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
	Shipment     *Shipment                  `json:"-"                   bun:"rel:belongs-to,join:shipment_id=id"`
	Commodity    *commodity.Commodity       `json:"commodity,omitempty" bun:"rel:belongs-to,join:commodity_id=id"`
	BusinessUnit *businessunit.BusinessUnit `json:"-"                   bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"-"                   bun:"rel:belongs-to,join:organization_id=id"`
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
			validation.Min(1).Error("Weight must be at least 1"),
		),
		// Commodity ID is required
		validation.Field(&sc.CommodityID,
			validation.Required.Error("Commodity ID is required"),
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

func (sc *ShipmentCommodity) MeetsLinearFeetPerUnitRequirement() bool {
	if sc.Commodity == nil || sc.Commodity.LinearFeetPerUnit == nil {
		return false
	}

	return sc.Pieces > 0 && *sc.Commodity.LinearFeetPerUnit > 0
}
