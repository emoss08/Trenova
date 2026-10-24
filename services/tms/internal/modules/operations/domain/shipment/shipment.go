package shipment

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmenttype"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/customfield"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ pagination.CursorEntity            = (*Shipment)(nil)
	_ domaintypes.PostgresSearchable     = (*Shipment)(nil)
	_ customfield.CustomFieldsSupporter  = (*Shipment)(nil)
	_ validationframework.TenantedEntity = (*Shipment)(nil)
)

type RatingSkip struct {
	Code   string `json:"code"`
	Source string `json:"source"`
}

type RatingDetail struct {
	Bill                *tarifftypes.Bill     `json:"bill,omitempty"`
	Refusal             string                `json:"refusal,omitempty"`
	Choices             []tarifftypes.Choice  `json:"choices,omitempty"`
	Source              string                `json:"source,omitempty"`
	RatedAt             int64                 `json:"ratedAt"`
	RateQuoteID         string                `json:"rateQuoteId,omitempty"`
	AgreementID         string                `json:"agreementId,omitempty"`
	AgreementName       string                `json:"agreementName,omitempty"`
	RuleID              string                `json:"ruleId,omitempty"`
	RuleLabel           string                `json:"ruleLabel,omitempty"`
	FormulaTemplateID   string                `json:"formulaTemplateId,omitempty"`
	FormulaTemplateName string                `json:"formulaTemplateName,omitempty"`
	VersionNumber       int64                 `json:"versionNumber,omitempty"`
	Inputs              tarifftypes.Inputs    `json:"inputs,omitempty"`
	Skipped             []RatingSkip          `json:"skipped,omitempty"`
	Explanation         string                `json:"explanation,omitempty"`
	Trace               rules.RawJSON         `json:"trace,omitempty"`
	ReviewApproval      *RatingReviewApproval `json:"reviewApproval,omitempty"`
}

type FreightLine struct {
	Label        string           `json:"label"`
	Quantity     *decimal.Decimal `json:"quantity,omitempty"`
	QuantityUnit string           `json:"quantityUnit,omitempty"`
	Rate         *decimal.Decimal `json:"rate,omitempty"`
	RateUnit     string           `json:"rateUnit,omitempty"`
	Amount       decimal.Decimal  `json:"amount"`
}

// Freight shipments and their lifecycle state
type Shipment struct {
	keel.Table `                                        keel:"shipments,report=Operations"`

	ID                        pulid.ID              `json:"id" keel:"id,pk,autoid:shp_"`
	BusinessUnitID            pulid.ID              `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID            pulid.ID              `json:"organizationId" keel:"organization_id,pk,tenant"`
	ServiceTypeID             pulid.ID              `json:"serviceTypeId" keel:"service_type_id"`
	ShipmentTypeID            pulid.ID              `json:"shipmentTypeId" keel:"shipment_type_id,nullzero"`
	CustomerID                pulid.ID              `json:"customerId" keel:"customer_id"`
	TractorTypeID             pulid.ID              `json:"tractorTypeId" keel:"tractor_type_id,nullzero"`
	TrailerTypeID             pulid.ID              `json:"trailerTypeId" keel:"trailer_type_id,nullzero"`
	OwnerID                   pulid.ID              `json:"ownerId" keel:"owner_id,nullzero"`
	EnteredByID               pulid.ID              `json:"enteredById" keel:"entered_by_id,nullzero"`
	CanceledByID              pulid.ID              `json:"canceledById" keel:"canceled_by_id,nullzero"`
	FormulaTemplateID         pulid.ID              `json:"formulaTemplateId" keel:"formula_template_id,nullzero"`
	ConsolidationGroupID      pulid.ID              `json:"consolidationGroupId" keel:"consolidation_group_id,nullzero"`
	OrderID                   pulid.ID              `json:"orderId" keel:"order_id,nullzero"`
	Status                    Status                `json:"status" keel:"status"`
	StageRank                 int16                 `json:"stageRank" keel:"stage_rank,readonly,noreport"`
	TenderStatus              *TenderStatus         `json:"tenderStatus" keel:"tender_status"`
	EntryMethod               EntryMethod           `json:"entryMethod" keel:"entry_method"`
	ProNumber                 string                `json:"proNumber" keel:"pro_number"`
	BOL                       string                `json:"bol" keel:"bol,nullzero"`
	ExternalReference         string                `json:"externalReference" keel:"external_reference,nullzero,label=Customer Reference"`
	IdempotencyKey            string                `json:"-" keel:"idempotency_key,nullzero,noreport"`
	CancelReason              string                `json:"cancelReason" keel:"cancel_reason,nullzero"`
	OtherChargeAmount         decimal.NullDecimal   `json:"otherChargeAmount" keel:"other_charge_amount,label=Accessorial Charges,format=money"`
	FreightChargeAmount       decimal.NullDecimal   `json:"freightChargeAmount" keel:"freight_charge_amount,label=Freight Charge,format=money"`
	TariffInputs              tarifftypes.Inputs    `json:"tariffInputs" keel:"tariff_inputs,jsonb"`
	FreightLines              []FreightLine         `json:"freightLines" keel:"freight_lines,jsonb"`
	TotalChargeAmount         decimal.NullDecimal   `json:"totalChargeAmount" keel:"total_charge_amount,label=Total Charge,format=money"`
	Pieces                    *int64                `json:"pieces" keel:"pieces,format=count"`
	Weight                    *int64                `json:"weight" keel:"weight,format=weight"`
	EnvelopeLengthFeet        *float64              `json:"envelopeLengthFeet" keel:"envelope_length_feet"`
	EnvelopeWidthFeet         *float64              `json:"envelopeWidthFeet" keel:"envelope_width_feet"`
	EnvelopeHeightFeet        *float64              `json:"envelopeHeightFeet" keel:"envelope_height_feet"`
	EnvelopeOverallHeightFeet *float64              `json:"envelopeOverallHeightFeet" keel:"envelope_overall_height_feet"`
	TemperatureMin            *int16                `json:"temperatureMin" keel:"temperature_min,dimension"`
	TemperatureMax            *int16                `json:"temperatureMax" keel:"temperature_max,dimension"`
	ActualDeliveryDate        *int64                `json:"actualDeliveryDate" keel:"actual_delivery_date,label=Delivery Date"`
	ActualShipDate            *int64                `json:"actualShipDate" keel:"actual_ship_date,label=Ship Date"`
	CanceledAt                *int64                `json:"canceledAt" keel:"canceled_at"`
	BillingTransferStatus     BillingTransferStatus `json:"billingTransferStatus" keel:"billing_transfer_status,nullzero"`
	TransferredToBillingAt    *int64                `json:"transferredToBillingAt" keel:"transferred_to_billing_at"`
	MarkedReadyToBillAt       *int64                `json:"markedReadyToBillAt" keel:"marked_ready_to_bill_at,label=Ready to Bill At"`
	BilledAt                  *int64                `json:"billedAt" keel:"billed_at"`
	FuelSurchargeLocked       bool                  `json:"fuelSurchargeLocked" keel:"fuel_surcharge_locked"`
	RatingDetail              *RatingDetail         `json:"ratingDetail" keel:"rating_detail,jsonb"`
	RateQuoteID               *pulid.ID             `json:"rateQuoteId" keel:"rate_quote_id"`
	RateAgreementID           *pulid.ID             `json:"rateAgreementId" keel:"rate_agreement_id"`
	RateAgreementRuleID       *pulid.ID             `json:"rateAgreementRuleId" keel:"rate_agreement_rule_id"`
	RateOverrideAmount        decimal.NullDecimal   `json:"rateOverrideAmount" keel:"rate_override_amount"`
	RateOverrideReason        string                `json:"rateOverrideReason" keel:"rate_override_reason,nullzero"`
	RateOverrideByID          *pulid.ID             `json:"rateOverrideById" keel:"rate_override_by_id"`
	RateOverrideAt            *int64                `json:"rateOverrideAt" keel:"rate_override_at"`
	RateLocked                bool                  `json:"rateLocked" keel:"rate_locked"`
	AutoRated                 bool                  `json:"autoRated" keel:"auto_rated"`
	AutoRatedAt               *int64                `json:"autoRatedAt" keel:"auto_rated_at"`
	BillToCustomerID          *pulid.ID             `json:"billToCustomerId" keel:"bill_to_customer_id"`
	FreightTerms              FreightTerms          `json:"freightTerms" keel:"freight_terms"`
	SourceDocumentID          string                `json:"sourceDocumentId,omitempty"`
	PreviousCustomerID        pulid.ID              `json:"-"`

	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-" keel:"rank,computed"`
	Version      int64  `json:"version" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit      *tenant.BusinessUnit             `json:"businessUnit,omitempty"`
	Organization      *tenant.Organization             `json:"organization,omitempty"`
	ShipmentType      *shipmenttype.ShipmentType       `json:"shipmentType,omitempty" keel:"belongs:shipment_type_id"`
	ServiceType       *servicetype.ServiceType         `json:"serviceType,omitempty" keel:"belongs:service_type_id"`
	Customer          *customer.Customer               `json:"customer,omitempty" keel:"belongs:customer_id"`
	BillToCustomer    *customer.Customer               `json:"billToCustomer,omitempty" keel:"belongs:bill_to_customer_id"`
	TractorType       *equipmenttype.EquipmentType     `json:"tractorType,omitempty" keel:"belongs:tractor_type_id,label=Tractor Equipment Type"`
	TrailerType       *equipmenttype.EquipmentType     `json:"trailerType,omitempty" keel:"belongs:trailer_type_id,label=Trailer Equipment Type"`
	CanceledBy        *tenant.User                     `json:"canceledBy,omitempty" keel:"belongs:canceled_by_id,noreport"`
	Owner             *tenant.User                     `json:"owner,omitempty" keel:"belongs:owner_id,noreport"`
	EnteredBy         *tenant.User                     `json:"enteredBy,omitempty" keel:"belongs:entered_by_id,noreport"`
	FormulaTemplate   *formulatemplate.FormulaTemplate `json:"formulaTemplate,omitempty" keel:"belongs:formula_template_id,noreport"`
	Moves             []*ShipmentMove                  `json:"moves,omitempty" keel:"many:shipment_id"`
	Commodities       []*ShipmentCommodity             `json:"commodities,omitempty" keel:"many:shipment_id"`
	AdditionalCharges []*AdditionalCharge              `json:"additionalCharges,omitempty" keel:"many:shipment_id,noreport"`
	ChargeAllocations []*ChargeAllocation              `json:"chargeAllocations,omitempty" keel:"many:shipment_id,noreport"`
	Comments          []*ShipmentComment               `json:"comments,omitempty" keel:"many:shipment_id,noreport"`
	OriginStop        *Stop                            `json:"originStop,omitempty" keel:"pick:moves.stops,first=moves.sequence|stops.sequence"`
	DestinationStop   *Stop                            `json:"destinationStop,omitempty" keel:"pick:moves.stops,last=moves.sequence|stops.sequence"`
}

// PayerID is the customer the shipment's charges default to. An explicit
// bill-to overrides the ordering customer; otherwise the two are the same party.
func (s *Shipment) PayerID() pulid.ID {
	if s == nil {
		return pulid.Nil
	}
	if s.BillToCustomerID != nil && s.BillToCustomerID.IsNotNil() {
		return *s.BillToCustomerID
	}

	return s.CustomerID
}

func (s *Shipment) HasExplicitBillTo() bool {
	return s != nil && s.BillToCustomerID != nil && s.BillToCustomerID.IsNotNil()
}

// NormalizeBillTo folds a bill-to that names the ordering customer back to
// "same as customer", so the two spellings never diverge in storage.
func (s *Shipment) NormalizeBillTo() {
	if s == nil || s.BillToCustomerID == nil {
		return
	}
	if s.BillToCustomerID.IsNil() || *s.BillToCustomerID == s.CustomerID {
		s.BillToCustomerID = nil
	}
}

// FollowCustomerChange keeps "same as customer" meaning the new customer: a
// bill-to that pointed at the old ordering customer was never a third party,
// so it moves with the customer rather than pinning the old one as payer.
func FollowCustomerChange(original, updated *Shipment) {
	if original == nil || updated == nil || original.CustomerID == updated.CustomerID {
		return
	}
	if updated.BillToCustomerID != nil && *updated.BillToCustomerID == original.CustomerID {
		updated.BillToCustomerID = nil
	}
}

func (s *Shipment) ApplyFreightTermsDefault(original *Shipment) {
	if s == nil || s.FreightTerms != "" {
		return
	}

	if original != nil && original.FreightTerms != "" {
		s.FreightTerms = original.FreightTerms
		return
	}

	s.FreightTerms = FreightTermsPrepaid
}

func (s *Shipment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		s,
		validation.Field(&s.ServiceTypeID, validation.Required.Error("Service type is required")),
		validation.Field(&s.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(&s.ExternalReference,
			validation.RuneLength(0, MaxExternalReferenceLength).
				Error("Customer reference cannot be longer than 100 characters"),
		),
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
			&s.FreightTerms,
			validation.Required.Error("Freight terms are required"),
			validation.In(
				FreightTermsPrepaid,
				FreightTermsCollect,
				FreightTermsThirdParty,
			).Error("Freight terms must be Prepaid, Collect or ThirdParty"),
		),
		validation.Field(
			&s.RateOverrideAmount,
			domainvalidation.DecimalAtLeast(decimal.Zero, "An overridden rate cannot be negative"),
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
	))
}

func (s *Shipment) HasRateOverride() bool {
	return s.RateOverrideAmount.Valid
}

func RestoreRateOwnedFields(original, updated *Shipment) {
	if original == nil || updated == nil {
		return
	}

	updated.RateOverrideAmount = original.RateOverrideAmount
	updated.RateOverrideByID = original.RateOverrideByID
	updated.RateOverrideAt = original.RateOverrideAt
	updated.RateLocked = original.RateLocked
	updated.RateQuoteID = original.RateQuoteID
	updated.RateAgreementID = original.RateAgreementID
	updated.RateAgreementRuleID = original.RateAgreementRuleID
	updated.AutoRated = original.AutoRated
	updated.AutoRatedAt = original.AutoRatedAt

	if updated.RateOverrideReason == "" {
		updated.RateOverrideReason = original.RateOverrideReason
	}

	updated.RatingDetail = original.RatingDetail
	updated.FreightLines = original.FreightLines
}

func (s *Shipment) ClearRatingOutcome() {
	s.FreightChargeAmount = decimal.NullDecimal{}
	s.FreightLines = []FreightLine{}
	s.OtherChargeAmount = decimal.NullDecimal{}
	s.TotalChargeAmount = decimal.NullDecimal{}
	s.RatingDetail = nil
	s.RateQuoteID = nil
	s.RateAgreementID = nil
	s.RateAgreementRuleID = nil
	s.RateOverrideAmount = decimal.NullDecimal{}
	s.RateOverrideReason = ""
	s.RateOverrideByID = nil
	s.RateOverrideAt = nil
	s.RateLocked = false
	s.ClearAutoRating()
}

func (s *Shipment) ClearAutoRating() {
	s.AutoRated = false
	s.AutoRatedAt = nil
}

func (s *Shipment) MarkAutoRated(at int64) {
	s.AutoRated = true
	s.AutoRatedAt = &at
}

func RatedFieldsEdited(original, updated *Shipment) bool {
	if original == nil || updated == nil {
		return false
	}

	if original.FormulaTemplateID != updated.FormulaTemplateID {
		return true
	}

	if !original.TariffInputs.Equal(updated.TariffInputs) {
		return true
	}

	return agreementChargesEdited(original.AdditionalCharges, updated.AdditionalCharges)
}

func agreementChargesEdited(original, updated []*AdditionalCharge) bool {
	remaining := make(map[pulid.ID]*AdditionalCharge, len(updated))
	for _, charge := range updated {
		if charge == nil || charge.ID.IsNil() {
			continue
		}
		remaining[charge.ID] = charge
	}

	for _, charge := range original {
		if charge == nil || charge.Owner() != SystemOwnerAgreement {
			continue
		}

		current, kept := remaining[charge.ID]
		if !kept {
			return true
		}

		if current.Method != charge.Method || current.Unit != charge.Unit ||
			!current.Amount.Equal(charge.Amount) {
			return true
		}
	}

	return false
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

// DeliveryStop is the last place the shipment is delivered to, which is the
// destination end of the lane a rate is written against.
func (s *Shipment) DeliveryStop() *Stop {
	if s == nil {
		return nil
	}

	return LastConsigneeStop(s.Moves)
}

// LastConsigneeStop mirrors FirstShipperStop from the other end of the trip.
//
// The ordering is the same one read backwards — latest move, then latest stop
// within it — and the stop id settles a tie so the answer never depends on the
// order the moves happened to be loaded in.
func LastConsigneeStop(moves []*ShipmentMove) *Stop {
	var best *Stop
	var bestMoveSeq int64
	var bestStopSeq int64
	bestStopID := ""

	for _, move := range moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil || !stop.IsDestinationStop() {
				continue
			}
			stopID := stop.ID.String()
			if best == nil ||
				move.Sequence > bestMoveSeq ||
				(move.Sequence == bestMoveSeq &&
					(stop.Sequence > bestStopSeq ||
						(stop.Sequence == bestStopSeq && stopID > bestStopID))) {
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

func (s *Shipment) GetResourceID() string {
	return s.ID.String()
}

func (s *Shipment) GetResourceType() string {
	return "shipment"
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
				Field:        "billToCustomer",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*customer.Customer)(nil),
				TargetTable:  "customers",
				ForeignKey:   "bill_to_customer_id",
				ReferenceKey: "id",
				Alias:        "bill_cus",
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
				Field:        ShipperStopRelationship,
				Type:         dbtype.RelationshipTypeCustom,
				TargetEntity: (*Stop)(nil),
				TargetTable:  "stops",
				Alias:        ShipperStopAlias,
				Queryable:    true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "stops",
						Alias:     ShipperStopAlias,
						Condition: ShipperStopAlias + ".id = " + ShipperStopIDSQL("sp"),
						JoinType:  dbtype.JoinTypeLeft,
					},
				},
			},
			{
				Field:        ConsigneeStopRelationship,
				Type:         dbtype.RelationshipTypeCustom,
				TargetEntity: (*Stop)(nil),
				TargetTable:  "stops",
				Alias:        ConsigneeStopAlias,
				Queryable:    true,
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "stops",
						Alias:     ConsigneeStopAlias,
						Condition: ConsigneeStopAlias + ".id = " + ConsigneeStopIDSQL("sp"),
						JoinType:  dbtype.JoinTypeLeft,
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

func (s *Shipment) BeforeCreate(ctx context.Context) error {
	if s.Status == "" {
		s.Status = StatusNew
	}
	if s.EntryMethod == "" {
		s.EntryMethod = EntryMethodManual
	}
	if s.FreightTerms == "" {
		s.FreightTerms = FreightTermsPrepaid
	}
	s.OtherChargeAmount = zeroWhenUnset(s.OtherChargeAmount)
	s.FreightChargeAmount = zeroWhenUnset(s.FreightChargeAmount)
	s.TotalChargeAmount = zeroWhenUnset(s.TotalChargeAmount)
	s.TariffInputs = s.TariffInputs.OrEmpty()
	if s.FreightLines == nil {
		s.FreightLines = []FreightLine{}
	}
	s.ExternalReference = strings.TrimSpace(s.ExternalReference)
	s.ApplyEntryMethodDefault(nil)
	s.ApplyFreightTermsDefault(nil)

	return nil
}

func zeroWhenUnset(amount decimal.NullDecimal) decimal.NullDecimal {
	if amount.Valid {
		return amount
	}

	return decimal.NewNullDecimal(decimal.Zero)
}

func (s *Shipment) BeforeUpdate(ctx context.Context) error {
	s.TariffInputs = s.TariffInputs.OrEmpty()
	if s.FreightLines == nil {
		s.FreightLines = []FreightLine{}
	}
	s.ExternalReference = strings.TrimSpace(s.ExternalReference)

	return nil
}

const MaxExternalReferenceLength = 100
