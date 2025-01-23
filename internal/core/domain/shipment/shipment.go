package shipment

import (
	"github.com/shopspring/decimal"
	"github.com/trenova-app/transport/internal/core/domain/businessunit"
	"github.com/trenova-app/transport/internal/core/domain/organization"
	"github.com/trenova-app/transport/internal/core/domain/servicetype"
	"github.com/trenova-app/transport/internal/core/domain/shipmenttype"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
)

type Shipment struct {
	bun.BaseModel `bun:"table:shipments,alias:sp" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`
	ServiceTypeID  pulid.ID `bun:"service_type_id,type:VARCHAR(100),pk,notnull" json:"serviceTypeId"`
	ShipmentTypeID pulid.ID `bun:"shipment_type_id,type:VARCHAR(100),pk,notnull" json:"shipmentTypeId"`

	// Core fields
	Status    Status `json:"status" bun:"status,type:status_enum,notnull,default:'New'"`
	ProNumber string `json:"proNumber" bun:"pro_number,type:VARCHAR(100),notnull"`

	// Billing Related Fields
	RatingUnit          int64               `json:"ratingUnit" bun:"rating_unit,type:INTEGER,notnull,default:1"`
	RatingMethod        RatingMethod        `json:"ratingMethod" bun:"rating_method,type:rating_method_enum,notnull,default:'Flat'"`
	OtherChargeAmount   decimal.NullDecimal `json:"otherChargeAmount" bun:"other_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	FreightChargeAmount decimal.NullDecimal `json:"freightChargeAmount" bun:"freight_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	TotalChargeAmount   decimal.NullDecimal `json:"totalChargeAmount" bun:"total_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	Pieces              *int64              `json:"pieces" bun:"pieces,type:INTEGER,nullzero"`
	Weight              *int64              `json:"weight" bun:"weight,type:INTEGER,nullzero"`
	ReadyToBillDate     *int64              `json:"readyToBillDate" bun:"ready_to_bill_date,type:BIGINT,nullzero"`
	SentToBillingDate   *int64              `json:"sentToBillingDate" bun:"sent_to_billing_date,type:BIGINT,nullzero"`
	BillDate            *int64              `json:"billDate" bun:"bill_date,type:BIGINT,nullzero"`
	ReadyToBill         bool                `json:"readyToBill" bun:"ready_to_bill,type:BOOLEAN,notnull,default:false"`
	SentToBilling       bool                `json:"sentToBilling" bun:"sent_to_billing,type:BOOLEAN,notnull,default:false"`
	Billed              bool                `json:"billed" bun:"billed,type:BOOLEAN,notnull,default:false"`

	// Misc. Shipment Related Fields
	TemperatureMin     decimal.NullDecimal `json:"temperatureMin" bun:"temperature_min,type:NUMERIC(10,2),nullzero"`
	TemperatureMax     decimal.NullDecimal `json:"temperatureMax" bun:"temperature_max,type:NUMERIC(10,2),nullzero"`
	BOL                string              `json:"bol" bun:"bol,type:VARCHAR(100),notnull"`
	IsHazardous        bool                `json:"isHazardous" bun:"is_hazardous,type:BOOLEAN,notnull,default:false"`
	ActualDeliveryDate *int64              `json:"actualDeliveryDate" bun:"actual_delivery_date,type:BIGINT,nullzero"`
	ActualShipDate     *int64              `json:"actualShipDate" bun:"actual_ship_date,type:BIGINT,nullzero"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	ShipmentType *shipmenttype.ShipmentType `json:"shipmentType,omitempty" bun:"rel:belongs-to,join:shipment_type_id=id"`
	ServiceType  *servicetype.ServiceType   `json:"serviceType,omitempty" bun:"rel:belongs-to,join:service_type_id=id"`
	Commodities  []*ShipmentCommodity       `json:"commodities,omitempty" bun:"rel:has-many,join:id=shipment_id"`
}
