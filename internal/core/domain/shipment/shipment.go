package shipment

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
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

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`

	// Relationship identifiers (Non-Primary-Keys)
	ServiceTypeID  pulid.ID  `bun:"service_type_id,type:VARCHAR(100),notnull" json:"serviceTypeId"`
	ShipmentTypeID pulid.ID  `bun:"shipment_type_id,type:VARCHAR(100),notnull" json:"shipmentTypeId"`
	CustomerID     pulid.ID  `bun:"customer_id,type:VARCHAR(100),notnull" json:"customerId"`
	TractorTypeID  *pulid.ID `bun:"tractor_type_id,type:VARCHAR(100),nullzero" json:"tractorTypeId"`
	TrailerTypeID  *pulid.ID `bun:"trailer_type_id,type:VARCHAR(100),nullzero" json:"trailerTypeId"`

	// Core fields
	Status    Status `json:"status" bun:"status,type:status_enum,notnull,default:'New'"`
	ProNumber string `json:"proNumber" bun:"pro_number,type:VARCHAR(100),notnull"`

	// Billing Related Fields
	RatingUnit          int64               `json:"ratingUnit" bun:"rating_unit,type:INTEGER,notnull,default:1"`
	OtherChargeAmount   decimal.NullDecimal `json:"otherChargeAmount" bun:"other_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	RatingMethod        RatingMethod        `json:"ratingMethod" bun:"rating_method,type:rating_method_enum,notnull,default:'Flat'"`
	FreightChargeAmount decimal.NullDecimal `json:"freightChargeAmount" bun:"freight_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	TotalChargeAmount   decimal.NullDecimal `json:"totalChargeAmount" bun:"total_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	Pieces              *int64              `json:"pieces" bun:"pieces,type:INTEGER,nullzero"`
	Weight              *int64              `json:"weight" bun:"weight,type:INTEGER,nullzero"`

	// Misc. Shipment Related Fields
	TemperatureMin     decimal.NullDecimal `json:"temperatureMin" bun:"temperature_min,type:NUMERIC(10,2),nullzero"`
	TemperatureMax     decimal.NullDecimal `json:"temperatureMax" bun:"temperature_max,type:NUMERIC(10,2),nullzero"`
	BOL                string              `json:"bol" bun:"bol,type:VARCHAR(100),notnull"`
	ActualDeliveryDate *int64              `json:"actualDeliveryDate" bun:"actual_delivery_date,type:BIGINT,nullzero"`
	ActualShipDate     *int64              `json:"actualShipDate" bun:"actual_ship_date,type:BIGINT,nullzero"`

	// Cancelation Related Fields
	CanceledAt   *int64    `json:"canceledAt" bun:"canceled_at,type:BIGINT,nullzero"`
	CanceledByID *pulid.ID `json:"canceledById" bun:"canceled_by_id,type:VARCHAR(100),nullzero"`
	CancelReason string    `json:"cancelReason" bun:"cancel_reason,type:VARCHAR(100),nullzero"`

	// Metadata
	Version      int64  `json:"version" bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-" bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-" bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit   `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization   `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	ShipmentType *shipmenttype.ShipmentType   `json:"shipmentType,omitempty" bun:"rel:belongs-to,join:shipment_type_id=id"`
	ServiceType  *servicetype.ServiceType     `json:"serviceType,omitempty" bun:"rel:belongs-to,join:service_type_id=id"`
	Customer     *customer.Customer           `json:"customer,omitempty" bun:"rel:belongs-to,join:customer_id=id"`
	TractorType  *equipmenttype.EquipmentType `json:"tractorType,omitempty" bun:"rel:belongs-to,join:tractor_type_id=id"`
	TrailerType  *equipmenttype.EquipmentType `json:"trailerType,omitempty" bun:"rel:belongs-to,join:trailer_type_id=id"`
	CanceledBy   *user.User                   `json:"canceledBy,omitempty" bun:"rel:belongs-to,join:canceled_by_id=id"`
	Moves        []*ShipmentMove              `json:"moves,omitempty" bun:"rel:has-many,join:id=shipment_id"`
	Commodities  []*ShipmentCommodity         `json:"commodities,omitempty" bun:"rel:has-many,join:id=shipment_id"`
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
			).Error("Rating Method must be a valid rating method"),
		),

		// Freight Charge Amount is required when rating method is flat
		validation.Field(&st.FreightChargeAmount,
			validation.When(st.RatingMethod == RatingMethodFlatRate,
				validation.Required.Error("Freight Charge Amount is required when rating method is Flat"),
			),
		),

		// Weight is reuqired method is per pound
		validation.Field(&st.Weight,
			validation.When(st.RatingMethod == RatingMethodPerPound,
				validation.Required.Error("Weight is required when rating method is Per Pound"),
			),
		),

		// Ensure rating unit is greater than 0 and required when rating method is Per Mile
		validation.Field(&st.RatingUnit,
			validation.When(st.RatingMethod == RatingMethodPerMile,
				validation.Required.Error("Rating Unit is required when rating method is Per Mile"),
				validation.Min(1).Error("Rating Unit must be greater than 0"),
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

// Pagination Configuration
func (st *Shipment) GetID() string {
	return st.ID.String()
}

func (st *Shipment) GetTableName() string {
	return "shipments"
}

// Search Configuration
func (st *Shipment) GetSearchType() string {
	return "shipment"
}

func (st *Shipment) ToDocument() infra.SearchDocument {
	searchableText := []string{
		st.ProNumber,
		st.BOL,
	}

	return infra.SearchDocument{
		ID:             st.ID.String(),
		Type:           "shipment",
		BusinessUnitID: st.BusinessUnitID.String(),
		OrganizationID: st.OrganizationID.String(),
		CreatedAt:      st.CreatedAt,
		UpdatedAt:      st.UpdatedAt,
		Title:          st.ProNumber,
		Description:    fmt.Sprintf("Pro Number: %s, BOL: %s, Status: %s", st.ProNumber, st.BOL, st.Status),
		SearchableText: strings.Join(searchableText, " "),
		Metadata: map[string]any{
			"bol":        st.BOL,
			"customerID": st.CustomerID.String(),
			"proNumber":  st.ProNumber,
			"status":     st.Status,
		},
	}
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
