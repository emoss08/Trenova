package shipment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ pagination.CursorEntity = (*Shipment)(nil)

type RatingBreakdownItem struct {
	Name   string  `json:"name"`
	Label  string  `json:"label,omitempty"`
	Amount float64 `json:"amount"`
	Error  string  `json:"error,omitempty"`
}

type RatingGuardrail struct {
	Applied   bool     `json:"applied"`
	Bound     string   `json:"bound,omitempty"`
	RawResult float64  `json:"rawResult"`
	MinCharge *float64 `json:"minCharge,omitempty"`
	MaxCharge *float64 `json:"maxCharge,omitempty"`
}

type RatingDetail struct {
	FormulaTemplateID   string                `json:"formulaTemplateId"`
	FormulaTemplateName string                `json:"formulaTemplateName"`
	Expression          string                `json:"expression"`
	ResolvedVariables   map[string]any        `json:"resolvedVariables"`
	Result              float64               `json:"result"`
	RatedAt             int64                 `json:"ratedAt"`
	VersionNumber       int64                 `json:"versionNumber,omitempty"`
	Breakdown           []RatingBreakdownItem `json:"breakdown,omitempty"`
	Guardrail           *RatingGuardrail      `json:"guardrail,omitempty"`
}

type Shipment struct {
	bun.BaseModel             `json:"-" bun:"table:shipments,alias:sp"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                     pulid.ID              `json:"id"                         bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID              `json:"businessUnitId"             bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID         pulid.ID              `json:"organizationId"             bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ServiceTypeID          pulid.ID              `json:"serviceTypeId"              bun:"service_type_id,type:VARCHAR(100),notnull"`
	ShipmentTypeID         pulid.ID              `json:"shipmentTypeId"             bun:"shipment_type_id,type:VARCHAR(100),nullzero"`
	CustomerID             pulid.ID              `json:"customerId"                 bun:"customer_id,type:VARCHAR(100),notnull"`
	TractorTypeID          pulid.ID              `json:"tractorTypeId"              bun:"tractor_type_id,type:VARCHAR(100),nullzero"`
	TrailerTypeID          pulid.ID              `json:"trailerTypeId"              bun:"trailer_type_id,type:VARCHAR(100),nullzero"`
	OwnerID                pulid.ID              `json:"ownerId"                    bun:"owner_id,type:VARCHAR(100),nullzero"`
	EnteredByID            pulid.ID              `json:"enteredById"                bun:"entered_by_id,type:VARCHAR(100),nullzero"`
	CanceledByID           pulid.ID              `json:"canceledById"               bun:"canceled_by_id,type:VARCHAR(100),nullzero"`
	FormulaTemplateID      pulid.ID              `json:"formulaTemplateId"          bun:"formula_template_id,type:VARCHAR(100)"`
	ConsolidationGroupID   pulid.ID              `json:"consolidationGroupId"       bun:"consolidation_group_id,type:VARCHAR(100),nullzero"`
	OrderID                pulid.ID              `json:"orderId"                    bun:"order_id,type:VARCHAR(100),nullzero"`
	Status                 Status                `json:"status"                     bun:"status,type:shipment_status_enum,notnull,default:'New'"`
	TenderStatus           *TenderStatus         `json:"tenderStatus"               bun:"tender_status,type:shipment_tender_status_enum,nullzero"`
	EntryMethod            EntryMethod           `json:"entryMethod"                bun:"entry_method,type:shipment_entry_method_enum,notnull,default:'Manual'"`
	ProNumber              string                `json:"proNumber"                  bun:"pro_number,type:VARCHAR(100),notnull"`
	BOL                    string                `json:"bol"                        bun:"bol,type:VARCHAR(100),nullzero"`
	CancelReason           string                `json:"cancelReason"               bun:"cancel_reason,type:VARCHAR(100),nullzero"`
	OtherChargeAmount      decimal.NullDecimal   `json:"otherChargeAmount"          bun:"other_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	FreightChargeAmount    decimal.NullDecimal   `json:"freightChargeAmount"        bun:"freight_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	BaseRate               decimal.NullDecimal   `json:"baseRate"                   bun:"base_rate,type:NUMERIC(19,4),notnull,default:0"`
	TotalChargeAmount      decimal.NullDecimal   `json:"totalChargeAmount"          bun:"total_charge_amount,type:NUMERIC(19,4),notnull,default:0"`
	Pieces                 *int64                `json:"pieces"                     bun:"pieces,type:INTEGER,nullzero"`
	Weight                 *int64                `json:"weight"                     bun:"weight,type:INTEGER,nullzero"`
	TemperatureMin         *int16                `json:"temperatureMin"             bun:"temperature_min,type:temperature_fahrenheit,nullzero"`
	TemperatureMax         *int16                `json:"temperatureMax"             bun:"temperature_max,type:temperature_fahrenheit,nullzero"`
	ActualDeliveryDate     *int64                `json:"actualDeliveryDate"         bun:"actual_delivery_date,type:BIGINT,nullzero"`
	ActualShipDate         *int64                `json:"actualShipDate"             bun:"actual_ship_date,type:BIGINT,nullzero"`
	CanceledAt             *int64                `json:"canceledAt"                 bun:"canceled_at,type:BIGINT,nullzero"`
	BillingTransferStatus  BillingTransferStatus `json:"billingTransferStatus"      bun:"billing_transfer_status,type:VARCHAR(50),nullzero"`
	TransferredToBillingAt *int64                `json:"transferredToBillingAt"     bun:"transferred_to_billing_at,type:BIGINT,nullzero"`
	MarkedReadyToBillAt    *int64                `json:"markedReadyToBillAt"        bun:"marked_ready_to_bill_at,type:BIGINT,nullzero"`
	BilledAt               *int64                `json:"billedAt"                   bun:"billed_at,type:BIGINT,nullzero"`
	RatingUnit             int64                 `json:"ratingUnit"                 bun:"rating_unit,type:INTEGER,notnull,default:1"`
	FuelSurchargeLocked    bool                  `json:"fuelSurchargeLocked"        bun:"fuel_surcharge_locked,type:BOOLEAN,notnull,default:false"`
	RatingDetail           *RatingDetail         `json:"ratingDetail"               bun:"rating_detail,type:JSONB,nullzero"`
	SourceDocumentID       string                `json:"sourceDocumentId,omitempty" bun:"-"`
	SearchVector           string                `json:"-"                          bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                   string                `json:"-"                          bun:"rank,type:VARCHAR(100),scanonly"`
	Version                int64                 `json:"version"                    bun:"version,type:BIGINT"`
	CreatedAt              int64                 `json:"createdAt"                  bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64                 `json:"updatedAt"                  bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit      *tenant.BusinessUnit             `json:"businessUnit,omitempty"      bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization      *tenant.Organization             `json:"organization,omitempty"      bun:"rel:belongs-to,join:organization_id=id"`
	ShipmentType      *shipmenttype.ShipmentType       `json:"shipmentType,omitempty"      bun:"rel:belongs-to,join:shipment_type_id=id"`
	ServiceType       *servicetype.ServiceType         `json:"serviceType,omitempty"       bun:"rel:belongs-to,join:service_type_id=id"`
	Customer          *customer.Customer               `json:"customer,omitempty"          bun:"rel:belongs-to,join:customer_id=id"`
	TractorType       *equipmenttype.EquipmentType     `json:"tractorType,omitempty"       bun:"rel:belongs-to,join:tractor_type_id=id"`
	TrailerType       *equipmenttype.EquipmentType     `json:"trailerType,omitempty"       bun:"rel:belongs-to,join:trailer_type_id=id"`
	CanceledBy        *tenant.User                     `json:"canceledBy,omitempty"        bun:"rel:belongs-to,join:canceled_by_id=id"`
	Owner             *tenant.User                     `json:"owner,omitempty"             bun:"rel:belongs-to,join:owner_id=id"`
	EnteredBy         *tenant.User                     `json:"enteredBy,omitempty"         bun:"rel:belongs-to,join:entered_by_id=id"`
	FormulaTemplate   *formulatemplate.FormulaTemplate `json:"formulaTemplate,omitempty"   bun:"rel:belongs-to,join:formula_template_id=id"`
	Moves             []*ShipmentMove                  `json:"moves,omitempty"             bun:"rel:has-many,join:id=shipment_id"`
	Commodities       []*ShipmentCommodity             `json:"commodities,omitempty"       bun:"rel:has-many,join:id=shipment_id"`
	AdditionalCharges []*AdditionalCharge              `json:"additionalCharges,omitempty" bun:"rel:has-many,join:id=shipment_id"`
	Comments          []*ShipmentComment               `json:"comments,omitempty"          bun:"rel:has-many,join:id=shipment_id"`
}

func (s *Shipment) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		s,
		validation.Field(&s.ServiceTypeID, validation.Required.Error("Service type is required")),
		validation.Field(&s.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(
			&s.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				StatusNew,
				StatusPartiallyAssigned,
				StatusAssigned,
				StatusInTransit,
				StatusDelayed,
				StatusPartiallyCompleted,
				StatusReadyToInvoice,
				StatusCompleted,
				StatusInvoiced,
				StatusCanceled,
			).Error("Status must be a valid status"),
		),
		validation.Field(
			&s.EntryMethod,
			validation.Required.Error("Entry method is required"),
			validation.In(
				EntryMethodManual,
				EntryMethodEDI,
			).Error("Entry method must be a valid entry method"),
		),
		validation.Field(
			&s.TenderStatus,
			validation.In(
				TenderStatusTendered,
				TenderStatusAccepted,
				TenderStatusRejected,
				TenderStatusExpired,
				TenderStatusCanceled,
			).Error("Tender status must be a valid tender status"),
		),
		validation.Field(
			&s.FormulaTemplateID,
			validation.Required.Error("Formula template is required"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (s *Shipment) ApplyEntryMethodDefault(original *Shipment) {
	if s == nil || s.EntryMethod != "" {
		return
	}

	if original != nil && original.EntryMethod != "" {
		s.EntryMethod = original.EntryMethod
		return
	}

	s.EntryMethod = EntryMethodManual
}

func (s *Shipment) ShipperStop() *Stop {
	if s == nil {
		return nil
	}

	return FirstShipperStop(s.Moves)
}

func FirstShipperStop(moves []*ShipmentMove) *Stop {
	var best *Stop
	var bestMoveSeq int64
	var bestStopSeq int64
	bestStopID := ""

	for _, move := range moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil || !stop.IsOriginStop() {
				continue
			}
			stopID := stop.ID.String()
			if best == nil ||
				move.Sequence < bestMoveSeq ||
				(move.Sequence == bestMoveSeq &&
					(stop.Sequence < bestStopSeq ||
						(stop.Sequence == bestStopSeq && stopID < bestStopID))) {
				best = stop
				bestMoveSeq = move.Sequence
				bestStopSeq = stop.Sequence
				bestStopID = stopID
			}
		}
	}

	return best
}

func (s *Shipment) GetID() pulid.ID {
	return s.ID
}

func (s *Shipment) GetCreatedAt() int64 {
	return s.CreatedAt
}

func (s *Shipment) GetTableName() string {
	return "shipments"
}

func (s *Shipment) GetOrganizationID() pulid.ID {
	return s.OrganizationID
}

func (s *Shipment) GetBusinessUnitID() pulid.ID {
	return s.BusinessUnitID
}

func (s *Shipment) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "sp",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "pro_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "bol", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "customer",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*customer.Customer)(nil),
				TargetTable:  "customers",
				ForeignKey:   "customer_id",
				ReferenceKey: "id",
				Alias:        "cus",
				Queryable:    true,
			},
			{
				Field:        "originLocation",
				Type:         dbtype.RelationshipTypeCustom,
				TargetEntity: (*location.Location)(nil),
				TargetTable:  "locations",
				Alias:        "orig_loc",
				Queryable:    true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "shipment_moves",
						Alias:     "sm_orig",
						Condition: "sp.id = sm_orig.shipment_id",
						JoinType:  dbtype.JoinTypeLeft,
					},
					{
						Table:     "stops",
						Alias:     "stop_orig",
						Condition: "sm_orig.id = stop_orig.shipment_move_id AND stop_orig.type = 'Pickup' AND stop_orig.sequence = 0",
						JoinType:  dbtype.JoinTypeLeft,
					},
					{
						Table:     "locations",
						Alias:     "orig_loc",
						Condition: "stop_orig.location_id = orig_loc.id",
						JoinType:  dbtype.JoinTypeLeft,
					},
				},
			},
			{
				Field:        "destinationLocation",
				Type:         dbtype.RelationshipTypeCustom,
				TargetEntity: (*location.Location)(nil),
				TargetTable:  "locations",
				Alias:        "dest_loc",
				Queryable:    true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "shipment_moves",
						Alias:     "sm_dest",
						Condition: "sp.id = sm_dest.shipment_id AND sm_dest.sequence = (SELECT MAX(sm2.sequence) FROM shipment_moves AS sm2 WHERE sm2.shipment_id = sp.id)",
						JoinType:  dbtype.JoinTypeLeft,
					},
					{
						Table:     "stops",
						Alias:     "stop_dest",
						Condition: "sm_dest.id = stop_dest.shipment_move_id AND stop_dest.sequence = (SELECT MAX(stp2.sequence) FROM stops AS stp2 WHERE stp2.shipment_move_id = sm_dest.id)",
						JoinType:  dbtype.JoinTypeLeft,
					},
					{
						Table:     "locations",
						Alias:     "dest_loc",
						Condition: "stop_dest.location_id = dest_loc.id",
						JoinType:  dbtype.JoinTypeLeft,
					},
				},
			},
			{
				Field:        "pickupAppointment",
				Type:         dbtype.RelationshipTypeCustom,
				TargetEntity: (*Stop)(nil),
				TargetTable:  "stops",
				Alias:        "pickup_appt",
				Queryable:    true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table: "shipment_moves",
						Alias: "sm_pickup_appt",
						Condition: "sp.id = sm_pickup_appt.shipment_id AND sm_pickup_appt.sequence = " +
							"(SELECT MIN(sm2.sequence) FROM shipment_moves AS sm2 WHERE sm2.shipment_id = sp.id)",
						JoinType: dbtype.JoinTypeLeft,
					},
					{
						Table: "stops",
						Alias: "pickup_appt",
						Condition: "sm_pickup_appt.id = pickup_appt.shipment_move_id " +
							"AND pickup_appt.type IN ('Pickup', 'SplitPickup') " +
							"AND pickup_appt.schedule_type = 'Appointment' AND pickup_appt.sequence = " +
							"(SELECT MIN(stp2.sequence) FROM stops AS stp2 " +
							"WHERE stp2.shipment_move_id = sm_pickup_appt.id " +
							"AND stp2.type IN ('Pickup', 'SplitPickup'))",
						JoinType: dbtype.JoinTypeLeft,
					},
				},
			},
			{
				Field:        "deliveryAppointment",
				Type:         dbtype.RelationshipTypeCustom,
				TargetEntity: (*Stop)(nil),
				TargetTable:  "stops",
				Alias:        "delivery_appt",
				Queryable:    true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table: "shipment_moves",
						Alias: "sm_delivery_appt",
						Condition: "sp.id = sm_delivery_appt.shipment_id AND sm_delivery_appt.sequence = " +
							"(SELECT MAX(sm2.sequence) FROM shipment_moves AS sm2 WHERE sm2.shipment_id = sp.id)",
						JoinType: dbtype.JoinTypeLeft,
					},
					{
						Table: "stops",
						Alias: "delivery_appt",
						Condition: "sm_delivery_appt.id = delivery_appt.shipment_move_id " +
							"AND delivery_appt.type IN ('Delivery', 'SplitDelivery') " +
							"AND delivery_appt.schedule_type = 'Appointment' AND delivery_appt.sequence = " +
							"(SELECT MAX(stp2.sequence) FROM stops AS stp2 " +
							"WHERE stp2.shipment_move_id = sm_delivery_appt.id " +
							"AND stp2.type IN ('Delivery', 'SplitDelivery'))",
						JoinType: dbtype.JoinTypeLeft,
					},
				},
			},
			{
				Field:        "owner",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*tenant.User)(nil),
				TargetTable:  "users",
				ForeignKey:   "owner_id",
				ReferenceKey: "id",
				Alias:        "own",
				Queryable:    true,
			},
		},
	}
}

func (s *Shipment) StatusEquals(status Status) bool {
	return s.Status == status
}

func (s *Shipment) IsCompleted() bool {
	return s.Status == StatusCompleted
}

func (s *Shipment) IsInTransit() bool {
	return s.Status == StatusInTransit
}

func (s *Shipment) IsCanceled() bool {
	return s.Status == StatusCanceled
}

func (s *Shipment) IsNew() bool {
	return s.Status == StatusNew
}

func (s *Shipment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		s.ApplyEntryMethodDefault(nil)
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("shp_")
		}

		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}
