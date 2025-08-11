/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Shipment)(nil)
	_ domain.Validatable        = (*Shipment)(nil)
	_ infra.PostgresSearchable  = (*Shipment)(nil)
)

type Shipment struct {
	bun.BaseModel `bun:"table:shipments,alias:sp" json:"-"`

	ID                   pulid.ID            `json:"id"                   bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID       pulid.ID            `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID       pulid.ID            `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ServiceTypeID        pulid.ID            `json:"serviceTypeId"        bun:"service_type_id,type:VARCHAR(100),notnull"`
	ShipmentTypeID       pulid.ID            `json:"shipmentTypeId"       bun:"shipment_type_id,type:VARCHAR(100),notnull"`
	CustomerID           pulid.ID            `json:"customerId"           bun:"customer_id,type:VARCHAR(100),notnull"`
	TractorTypeID        *pulid.ID           `json:"tractorTypeId"        bun:"tractor_type_id,type:VARCHAR(100),nullzero"`
	TrailerTypeID        *pulid.ID           `json:"trailerTypeId"        bun:"trailer_type_id,type:VARCHAR(100),nullzero"`
	OwnerID              *pulid.ID           `json:"ownerId"              bun:"owner_id,type:VARCHAR(100),nullzero"`
	EnteredByID          *pulid.ID           `json:"enteredById"          bun:"entered_by_id,type:VARCHAR(100),nullzero"`
	CanceledByID         *pulid.ID           `json:"canceledById"         bun:"canceled_by_id,type:VARCHAR(100),nullzero"`
	FormulaTemplateID    *pulid.ID           `json:"formulaTemplateId"    bun:"formula_template_id,type:VARCHAR(100),nullzero"`
	ConsolidationGroupID *pulid.ID           `json:"consolidationGroupId" bun:"consolidation_group_id,type:VARCHAR(100),nullzero"`
	Status               Status              `json:"status"               bun:"status,type:status_enum,notnull,default:'New'"`
	ProNumber            string              `json:"proNumber"            bun:"pro_number,type:VARCHAR(100),notnull"`
	BOL                  string              `json:"bol"                  bun:"bol,type:VARCHAR(100),notnull"`
	CancelReason         string              `json:"cancelReason"         bun:"cancel_reason,type:VARCHAR(100),nullzero"`
	SearchVector         string              `json:"-"                    bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                 string              `json:"-"                    bun:"rank,type:VARCHAR(100),scanonly"`
	RatingMethod         RatingMethod        `json:"ratingMethod"         bun:"rating_method,type:rating_method_enum,notnull,default:'Flat'"`
	OtherChargeAmount    decimal.NullDecimal `json:"otherChargeAmount"    bun:"other_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	FreightChargeAmount  decimal.NullDecimal `json:"freightChargeAmount"  bun:"freight_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	TotalChargeAmount    decimal.NullDecimal `json:"totalChargeAmount"    bun:"total_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	Pieces               *int64              `json:"pieces"               bun:"pieces,type:INTEGER,nullzero"`
	Weight               *int64              `json:"weight"               bun:"weight,type:INTEGER,nullzero"`
	TemperatureMin       *int16              `json:"temperatureMin"       bun:"temperature_min,type:temperature_fahrenheit,nullzero"`
	TemperatureMax       *int16              `json:"temperatureMax"       bun:"temperature_max,type:temperature_fahrenheit,nullzero"`
	ActualDeliveryDate   *int64              `json:"actualDeliveryDate"   bun:"actual_delivery_date,type:BIGINT,nullzero"`
	ActualShipDate       *int64              `json:"actualShipDate"       bun:"actual_ship_date,type:BIGINT,nullzero"`
	CanceledAt           *int64              `json:"canceledAt"           bun:"canceled_at,type:BIGINT,nullzero"`
	RatingUnit           int64               `json:"ratingUnit"           bun:"rating_unit,type:INTEGER,notnull,default:1"`
	Version              int64               `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64               `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64               `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit      *businessunit.BusinessUnit       `json:"businessUnit,omitempty"     bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *organization.Organization       `json:"organization,omitempty"     bun:"rel:belongs-to,join:organization_id=id"`
	ShipmentType      *shipmenttype.ShipmentType       `json:"shipmentType,omitempty"     bun:"rel:belongs-to,join:shipment_type_id=id"`
	ServiceType       *servicetype.ServiceType         `json:"serviceType,omitempty"      bun:"rel:belongs-to,join:service_type_id=id"`
	Customer          *customer.Customer               `json:"customer,omitempty"         bun:"rel:belongs-to,join:customer_id=id"`
	TractorType       *equipmenttype.EquipmentType     `json:"tractorType,omitempty"      bun:"rel:belongs-to,join:tractor_type_id=id"`
	TrailerType       *equipmenttype.EquipmentType     `json:"trailerType,omitempty"      bun:"rel:belongs-to,join:trailer_type_id=id"`
	CanceledBy        *user.User                       `json:"canceledBy,omitempty"       bun:"rel:belongs-to,join:canceled_by_id=id"`
	Owner             *user.User                       `json:"owner,omitempty"            bun:"rel:belongs-to,join:owner_id=id"`
	EnteredBy         *user.User                       `json:"enteredBy,omitempty"        bun:"rel:belongs-to,join:entered_by_id=id"`
	FormulaTemplate   *formulatemplate.FormulaTemplate `json:"formulaTemplate,omitempty"  bun:"rel:belongs-to,join:formula_template_id=id"`
	Moves             []*ShipmentMove                  `json:"moves,omitempty"            bun:"rel:has-many,join:id=shipment_id"`
	Comments          []*ShipmentComment               `json:"comments,omitempty"         bun:"rel:has-many,join:id=shipment_id"`
	Commodities       []*ShipmentCommodity             `json:"commodities,omitempty"      bun:"rel:has-many,join:id=shipment_id"`
	AdditionalCharges []*AdditionalCharge              `json:"additionalCharges,omitzero" bun:"rel:has-many,join:id=shipment_id"`
}

func (st *Shipment) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, st,
		// Status is required and must be a valid status
		validation.Field(&st.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				StatusNew,
				StatusInTransit,
				StatusDelayed,
				StatusCompleted,
				StatusBilled,
				StatusPartiallyAssigned,
				StatusAssigned,
				StatusPartiallyCompleted,
				StatusCanceled,
			).Error("Status must be a valid status"),
		),

		// ShipmentTypeID is required
		validation.Field(&st.ShipmentTypeID,
			validation.Required.Error("Shipment Type is required"),
		),

		// CustomerID is required
		validation.Field(&st.CustomerID,
			validation.Required.Error("Customer is required"),
		),

		// BOL is required and must be between 1 and 100 characters
		validation.Field(&st.BOL,
			validation.Required.Error("BOL is required"),
			validation.Length(1, 100).Error("BOL must be between 1 and 100 characters"),
		),

		// Rating method is required and must be a valid rating method
		validation.Field(&st.RatingMethod,
			validation.Required.Error("Rating Method is required"),
			validation.In(
				RatingMethodFlatRate,
				RatingMethodPerMile,
				RatingMethodPerStop,
				RatingMethodPerPound,
				RatingMethodPerPallet,
				RatingMethodPerLinearFoot,
				RatingMethodOther,
				RatingMethodFormulaTemplate,
			).Error("Rating Method must be a valid rating method"),
		),

		// Freight Charge Amount is required when rating method is flat
		validation.Field(&st.FreightChargeAmount,
			validation.When(
				st.RatingMethod == RatingMethodFlatRate,
				validation.Required.Error(
					"Freight Charge Amount is required when rating method is Flat",
				),
			),
		),

		// Weight is reuqired method is per pound
		validation.Field(&st.Weight,
			validation.When(st.RatingMethod == RatingMethodPerPound,
				validation.Required.Error("Weight is required when rating method is Per Pound"),
			),
		),

		// Temperature Max cannot be less than Temperature Min
		validation.Field(&st.TemperatureMax,
			validation.By(domain.ValidateTemperaturePointer),
			validation.When(
				st.TemperatureMin != nil,
				validation.Min(intutils.ToInt16(st.TemperatureMin)).
					Error("Temperature Max must be greater than Temperature Min"),
			),
		),

		// Temperature Min cannot be greater than Temperature Max
		validation.Field(&st.TemperatureMin,
			validation.By(domain.ValidateTemperaturePointer),
			validation.When(
				st.TemperatureMax != nil,
				validation.Max(intutils.ToInt16(st.TemperatureMax)).
					Error("Temperature Min must be less than Temperature Max"),
			),
		),

		// Ensure rating unit is greater than 0 and required when rating method is Per Mile
		validation.Field(&st.RatingUnit,
			validation.When(st.RatingMethod == RatingMethodPerMile,
				validation.Required.Error("Rating Unit is required when rating method is Per Mile"),
				validation.Min(1).Error("Rating Unit must be greater than 0"),
			),
		),

		// Formula Template ID is required when rating method is FormulaTemplate
		validation.Field(&st.FormulaTemplateID,
			validation.When(
				st.RatingMethod == RatingMethodFormulaTemplate,
				validation.Required.Error(
					"Formula Template is required when rating method is Formula Template",
				),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (st *Shipment) GetID() string {
	return st.ID.String()
}

func (st *Shipment) GetTableName() string {
	return "shipments"
}

// HasCommodities returns true if the shipment has commodities
func (st *Shipment) HasCommodities() bool {
	// Check if the shipment has commodities
	if st.Commodities == nil {
		return false
	}

	// Check if the shipment has any commodities
	return len(st.Commodities) > 0
}

// HasAdditionalCharge returns true if the shipment has additional charges
func (st *Shipment) HasAdditionalCharge() bool {
	return len(st.AdditionalCharges) > 0
}

// HasMoves returns true if the shipment has moves
func (st *Shipment) HasMoves() bool {
	return len(st.Moves) > 0
}

func (st *Shipment) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "sp",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "pro_number",
				Weight: "A",
				Type:   infra.PostgresSearchTypeComposite,
			},
			{
				Name:   "bol",
				Weight: "A",
				Type:   infra.PostgresSearchTypeComposite,
			},
			{
				Name:       "status",
				Weight:     "B",
				Type:       infra.PostgresSearchTypeEnum,
				Dictionary: "english",
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}

func (st *Shipment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if st.ID.IsNil() {
			st.ID = pulid.MustNew("shp_")
		}

		st.CreatedAt = now
	case *bun.UpdateQuery:
		st.UpdatedAt = now
	}

	return nil
}

func (st *Shipment) StatusEquals(status Status) bool {
	return st.Status == status
}
