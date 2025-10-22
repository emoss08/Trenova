package shipment

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/meilisearchtype"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Shipment)(nil)
	_ domain.Validatable             = (*Shipment)(nil)
	_ framework.TenantedEntity       = (*Shipment)(nil)
	_ domaintypes.PostgresSearchable = (*Shipment)(nil)
	_ meilisearchtype.Searchable     = (*Shipment)(nil)
)

type Shipment struct {
	bun.BaseModel `bun:"table:shipments,alias:sp" json:"-"`

	ID                   pulid.ID                         `json:"id"                          bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID       pulid.ID                         `json:"businessUnitId"              bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID       pulid.ID                         `json:"organizationId"              bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ServiceTypeID        pulid.ID                         `json:"serviceTypeId"               bun:"service_type_id,type:VARCHAR(100),notnull"`
	ShipmentTypeID       pulid.ID                         `json:"shipmentTypeId"              bun:"shipment_type_id,type:VARCHAR(100),notnull"`
	CustomerID           pulid.ID                         `json:"customerId"                  bun:"customer_id,type:VARCHAR(100),notnull"`
	TractorTypeID        *pulid.ID                        `json:"tractorTypeId"               bun:"tractor_type_id,type:VARCHAR(100),nullzero"`
	TrailerTypeID        *pulid.ID                        `json:"trailerTypeId"               bun:"trailer_type_id,type:VARCHAR(100),nullzero"`
	OwnerID              *pulid.ID                        `json:"ownerId"                     bun:"owner_id,type:VARCHAR(100),nullzero"`
	EnteredByID          *pulid.ID                        `json:"enteredById"                 bun:"entered_by_id,type:VARCHAR(100),nullzero"`
	CanceledByID         *pulid.ID                        `json:"canceledById"                bun:"canceled_by_id,type:VARCHAR(100),nullzero"`
	FormulaTemplateID    *pulid.ID                        `json:"formulaTemplateId"           bun:"formula_template_id,type:VARCHAR(100),nullzero"`
	ConsolidationGroupID *pulid.ID                        `json:"consolidationGroupId"        bun:"consolidation_group_id,type:VARCHAR(100),nullzero"`
	Status               Status                           `json:"status"                      bun:"status,type:status_enum,notnull,default:'New'"`
	ProNumber            string                           `json:"proNumber"                   bun:"pro_number,type:VARCHAR(100),notnull"`
	BOL                  string                           `json:"bol"                         bun:"bol,type:VARCHAR(100),notnull"`
	CancelReason         string                           `json:"cancelReason"                bun:"cancel_reason,type:VARCHAR(100),nullzero"`
	SearchVector         string                           `json:"-"                           bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                 string                           `json:"-"                           bun:"rank,type:VARCHAR(100),scanonly"`
	RatingMethod         RatingMethod                     `json:"ratingMethod"                bun:"rating_method,type:rating_method_enum,notnull,default:'Flat'"`
	OtherChargeAmount    decimal.NullDecimal              `json:"otherChargeAmount"           bun:"other_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	FreightChargeAmount  decimal.NullDecimal              `json:"freightChargeAmount"         bun:"freight_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	TotalChargeAmount    decimal.NullDecimal              `json:"totalChargeAmount"           bun:"total_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	Pieces               *int64                           `json:"pieces"                      bun:"pieces,type:INTEGER,nullzero"`
	Weight               *int64                           `json:"weight"                      bun:"weight,type:INTEGER,nullzero"`
	TemperatureMin       *int16                           `json:"temperatureMin"              bun:"temperature_min,type:temperature_fahrenheit,nullzero"`
	TemperatureMax       *int16                           `json:"temperatureMax"              bun:"temperature_max,type:temperature_fahrenheit,nullzero"`
	ActualDeliveryDate   *int64                           `json:"actualDeliveryDate"          bun:"actual_delivery_date,type:BIGINT,nullzero"`
	ActualShipDate       *int64                           `json:"actualShipDate"              bun:"actual_ship_date,type:BIGINT,nullzero"`
	CanceledAt           *int64                           `json:"canceledAt"                  bun:"canceled_at,type:BIGINT,nullzero"`
	RatingUnit           int64                            `json:"ratingUnit"                  bun:"rating_unit,type:INTEGER,notnull,default:1"`
	Version              int64                            `json:"version"                     bun:"version,type:BIGINT"`
	CreatedAt            int64                            `json:"createdAt"                   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64                            `json:"updatedAt"                   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	BusinessUnit         *tenant.BusinessUnit             `json:"businessUnit,omitempty"      bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization         *tenant.Organization             `json:"organization,omitempty"      bun:"rel:belongs-to,join:organization_id=id"`
	ShipmentType         *shipmenttype.ShipmentType       `json:"shipmentType,omitempty"      bun:"rel:belongs-to,join:shipment_type_id=id"`
	ServiceType          *servicetype.ServiceType         `json:"serviceType,omitempty"       bun:"rel:belongs-to,join:service_type_id=id"`
	Customer             *customer.Customer               `json:"customer,omitempty"          bun:"rel:belongs-to,join:customer_id=id"`
	TractorType          *equipmenttype.EquipmentType     `json:"tractorType,omitempty"       bun:"rel:belongs-to,join:tractor_type_id=id"`
	TrailerType          *equipmenttype.EquipmentType     `json:"trailerType,omitempty"       bun:"rel:belongs-to,join:trailer_type_id=id"`
	CanceledBy           *tenant.User                     `json:"canceledBy,omitempty"        bun:"rel:belongs-to,join:canceled_by_id=id"`
	Owner                *tenant.User                     `json:"owner,omitempty"             bun:"rel:belongs-to,join:owner_id=id"`
	EnteredBy            *tenant.User                     `json:"enteredBy,omitempty"         bun:"rel:belongs-to,join:entered_by_id=id"`
	FormulaTemplate      *formulatemplate.FormulaTemplate `json:"formulaTemplate,omitempty"   bun:"rel:belongs-to,join:formula_template_id=id"`
	Moves                []*ShipmentMove                  `json:"moves,omitempty"             bun:"rel:has-many,join:id=shipment_id"`
	Comments             []*ShipmentComment               `json:"comments,omitempty"          bun:"rel:has-many,join:id=shipment_id"`
	Commodities          []*ShipmentCommodity             `json:"commodities,omitempty"       bun:"rel:has-many,join:id=shipment_id"`
	AdditionalCharges    []*AdditionalCharge              `json:"additionalCharges,omitempty" bun:"rel:has-many,join:id=shipment_id"`
	Holds                []*ShipmentHold                  `json:"holds,omitempty"             bun:"rel:has-many,join:id=shipment_id"`
}

func (st *Shipment) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(st,
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
		validation.Field(&st.ShipmentTypeID,
			validation.Required.Error("Shipment Type is required"),
		),
		validation.Field(&st.CustomerID,
			validation.Required.Error("Customer is required"),
		),
		validation.Field(&st.BOL,
			validation.Required.Error("BOL is required"),
			validation.Length(1, 100).Error("BOL must be between 1 and 100 characters"),
		),
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
		validation.Field(&st.FreightChargeAmount,
			validation.When(
				st.RatingMethod == RatingMethodFlatRate,
				validation.Required.Error(
					"Freight Charge Amount is required when rating method is Flat",
				),
			),
		),
		validation.Field(&st.Weight,
			validation.When(st.RatingMethod == RatingMethodPerPound,
				validation.Required.Error("Weight is required when rating method is Per Pound"),
			),
		),
		validation.Field(&st.TemperatureMax,
			validation.By(domain.ValidateTemperaturePointer),
			validation.When(
				st.TemperatureMin != nil,
				validation.Min(utils.ToInt16(st.TemperatureMin)).
					Error("Temperature Max must be greater than Temperature Min"),
			),
		),
		validation.Field(&st.TemperatureMin,
			validation.By(domain.ValidateTemperaturePointer),
			validation.When(
				st.TemperatureMax != nil,
				validation.Max(utils.ToInt16(st.TemperatureMax)).
					Error("Temperature Min must be less than Temperature Max"),
			),
		),
		validation.Field(&st.RatingUnit,
			validation.When(st.RatingMethod == RatingMethodPerMile,
				validation.Required.Error("Rating Unit is required when rating method is Per Mile"),
				validation.Min(1).Error("Rating Unit must be greater than 0"),
			),
		),
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
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (st *Shipment) GetID() string {
	return st.ID.String()
}

func (st *Shipment) GetOrganizationID() pulid.ID {
	return st.OrganizationID
}

func (st *Shipment) GetBusinessUnitID() pulid.ID {
	return st.BusinessUnitID
}

func (st *Shipment) GetTableName() string {
	return "shipments"
}

func (st *Shipment) HasCommodities() bool {
	if st.Commodities == nil {
		return false
	}

	return len(st.Commodities) > 0
}

func (st *Shipment) HasAdditionalCharge() bool {
	return len(st.AdditionalCharges) > 0
}

func (st *Shipment) HasMoves() bool {
	return len(st.Moves) > 0
}

func (st *Shipment) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig { //nolint:funlen // this is fine
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "sp",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "pro_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "bol",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
		},
		Relationships: []*domaintypes.RelationshipDefinition{
			{
				Field:        "customer",
				Type:         domaintypes.RelationshipTypeBelongsTo,
				TargetEntity: (*customer.Customer)(nil),
				TargetTable:  "customers",
				ForeignKey:   "customer_id",
				ReferenceKey: "id",
				Alias:        "cus",
				Queryable:    true,
			},
			{
				Field:        "originLocation",
				Type:         domaintypes.RelationshipTypeCustom,
				TargetEntity: (*location.Location)(nil),
				TargetTable:  "locations",
				Alias:        "orig_loc",
				Queryable:    true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "shipment_moves",
						Alias:     "sm_orig",
						Condition: "sp.id = sm_orig.shipment_id",
						JoinType:  domaintypes.JoinTypeLeft,
					},
					{
						Table:     "stops",
						Alias:     "stop_orig",
						Condition: "sm_orig.id = stop_orig.shipment_move_id AND stop_orig.type = 'Pickup' AND stop_orig.sequence = 0",
						JoinType:  domaintypes.JoinTypeLeft,
					},
					{
						Table:     "locations",
						Alias:     "orig_loc",
						Condition: "stop_orig.location_id = orig_loc.id",
						JoinType:  domaintypes.JoinTypeLeft,
					},
				},
			},
			{
				Field:        "destinationLocation",
				Type:         domaintypes.RelationshipTypeCustom,
				TargetEntity: (*location.Location)(nil),
				TargetTable:  "locations",
				Alias:        "dest_loc",
				Queryable:    true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "shipment_moves",
						Alias:     "sm_dest",
						Condition: "sp.id = sm_dest.shipment_id",
						JoinType:  domaintypes.JoinTypeLeft,
					},
					{
						Table:     "stops",
						Alias:     "stop_dest",
						Condition: "sm_dest.id = stop_dest.shipment_move_id AND stop_dest.type = 'Delivery'",
						JoinType:  domaintypes.JoinTypeLeft,
					},
					{
						Table:     "locations",
						Alias:     "dest_loc",
						Condition: "stop_dest.location_id = dest_loc.id",
						JoinType:  domaintypes.JoinTypeLeft,
					},
				},
			},
			{
				Field:       "originDate",
				Type:        domaintypes.RelationshipTypeCustom,
				TargetTable: "stops",
				Alias:       "stop_orig_date",
				TargetField: "planned_arrival",
				Queryable:   true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "shipment_moves",
						Alias:     "sm_orig_date",
						Condition: "sp.id = sm_orig_date.shipment_id",
						JoinType:  domaintypes.JoinTypeLeft,
					},
					{
						Table:     "stops",
						Alias:     "stop_orig_date",
						Condition: "sm_orig_date.id = stop_orig_date.shipment_move_id AND stop_orig_date.type = 'Pickup' AND stop_orig_date.sequence = 0",
						JoinType:  domaintypes.JoinTypeLeft,
					},
				},
			},
		},
	}
}

func (st *Shipment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

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

func (st *Shipment) IsCompleted() bool {
	return st.Status == StatusCompleted
}

func (st *Shipment) IsInTransit() bool {
	return st.Status == StatusInTransit
}

func (st *Shipment) IsBilled() bool {
	return st.Status == StatusBilled
}

func (st *Shipment) IsCanceled() bool {
	return st.Status == StatusCanceled
}

func (st *Shipment) IsDelayed() bool {
	return st.Status == StatusDelayed
}

func (st *Shipment) IsNew() bool {
	return st.Status == StatusNew
}

func (st *Shipment) GetSearchTitle() string {
	var b strings.Builder
	b.Grow(100) // Pre-allocate reasonable capacity

	if st.ProNumber != "" {
		b.WriteString(st.ProNumber)
	}

	if st.BOL != "" {
		if b.Len() > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(st.BOL)
	}

	if b.Len() == 0 {
		return st.ID.String()
	}

	return b.String()
}

func (st *Shipment) GetSearchSubtitle() string {
	var b strings.Builder
	b.Grow(150) // Pre-allocate reasonable capacity

	b.WriteString(string(st.Status))

	if st.Customer != nil && st.Customer.Name != "" {
		b.WriteString(" • ")
		b.WriteString(st.Customer.Name)
	}

	if st.ServiceType != nil && st.ServiceType.Code != "" {
		b.WriteString(" • ")
		b.WriteString(st.ServiceType.Code)
	}

	return b.String()
}

func (st *Shipment) GetSearchContent() string {
	var b strings.Builder
	b.Grow(200) // Pre-allocate reasonable capacity

	appendWithSpace := func(s string) {
		if s == "" {
			return
		}
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(s)
	}

	appendWithSpace(st.ProNumber)
	appendWithSpace(st.BOL)

	if st.Customer != nil {
		appendWithSpace(st.Customer.Name)
		appendWithSpace(st.Customer.Code)
	}

	if st.ServiceType != nil {
		appendWithSpace(st.ServiceType.Code)
	}

	return b.String()
}

func (st *Shipment) GetSearchMetadata() map[string]any {
	metadata := make(map[string]any, 10) // Pre-allocate with expected capacity

	metadata["proNumber"] = st.ProNumber
	metadata["bol"] = st.BOL
	metadata["status"] = string(st.Status)

	if st.Customer != nil {
		metadata["customerId"] = st.Customer.ID.String()
		metadata["customerName"] = st.Customer.Name
		if st.Customer.Code != "" {
			metadata["customerCode"] = st.Customer.Code
		}
	}

	if st.ServiceType != nil && st.ServiceType.Code != "" {
		metadata["serviceTypeCode"] = st.ServiceType.Code
		if st.ServiceType.Description != "" {
			metadata["serviceTypeDescription"] = st.ServiceType.Description
		}
	}

	if st.TotalChargeAmount.Valid {
		metadata["totalChargeAmount"] = st.TotalChargeAmount.Decimal.String()
	}

	if st.ActualShipDate != nil && *st.ActualShipDate > 0 {
		metadata["actualShipDate"] = *st.ActualShipDate
	}
	if st.ActualDeliveryDate != nil && *st.ActualDeliveryDate > 0 {
		metadata["actualDeliveryDate"] = *st.ActualDeliveryDate
	}

	return metadata
}

func (st *Shipment) GetSearchEntityType() meilisearchtype.EntityType {
	return meilisearchtype.EntityTypeShipment
}

func (st *Shipment) GetSearchTimestamps() (createdAt, updatedAt int64) {
	return st.CreatedAt, st.UpdatedAt
}
