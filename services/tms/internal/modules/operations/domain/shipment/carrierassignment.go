package shipment

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// CarrierBaseAmount extends a buy rate into the base amount: per-mile rates
// multiply by the move distance, flat rates pass through. Every surface that
// prices a carrier offer or assignment must share this math so documents and
// settlements cannot diverge on rounding.
func CarrierBaseAmount(
	method shipmentvocab.CarrierRateMethod,
	rate decimal.Decimal,
	distance *float64,
) decimal.Decimal {
	if method == shipmentvocab.CarrierRateMethodPerMile {
		miles := decimal.Zero
		if distance != nil {
			miles = decimal.NewFromFloat(*distance)
		}
		return rate.Mul(miles).Round(4)
	}
	return rate
}

type CarrierAssignment struct {
	keel.Table `keel:"carrier_assignments"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:casn_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	ShipmentMoveID pulid.ID `json:"shipmentMoveId" keel:"shipment_move_id"`
	CarrierID      pulid.ID `json:"carrierId"      keel:"carrier_id"`

	Status                shipmentvocab.CarrierAssignmentStatus `json:"status"                keel:"status"`
	RateMethod            shipmentvocab.CarrierRateMethod       `json:"rateMethod"            keel:"rate_method"`
	BaseRate              decimal.Decimal                       `json:"baseRate"              keel:"base_rate"`
	BaseAmount            decimal.Decimal                       `json:"baseAmount"            keel:"base_amount"`
	FuelSurcharge         decimal.Decimal                       `json:"fuelSurcharge"         keel:"fuel_surcharge"`
	AccessorialTotal      decimal.Decimal                       `json:"accessorialTotal"      keel:"accessorial_total"`
	TotalCost             decimal.Decimal                       `json:"totalCost"             keel:"total_cost"`
	CurrencyCode          string                                `json:"currencyCode"          keel:"currency_code"`
	ProNumber             string                                `json:"proNumber"             keel:"pro_number,nullzero"`
	ExternalDriverName    string                                `json:"externalDriverName"    keel:"external_driver_name,nullzero"`
	ExternalDriverPhone   string                                `json:"externalDriverPhone"   keel:"external_driver_phone,nullzero"`
	ExternalTractorNumber string                                `json:"externalTractorNumber" keel:"external_tractor_number,nullzero"`
	ExternalTrailerNumber string                                `json:"externalTrailerNumber" keel:"external_trailer_number,nullzero"`
	AssignedByID          *pulid.ID                             `json:"assignedById"          keel:"assigned_by_id"`
	// RateQuoteID names the buy side quote that decided what this carrier is
	// paid. Absent when somebody typed the rate, which stays the ordinary case
	// until an organization writes carrier contracts.
	RateQuoteID        *pulid.ID `json:"rateQuoteId"           keel:"rate_quote_id"`
	ConfirmedAt        *int64    `json:"confirmedAt"           keel:"confirmed_at"`
	CanceledAt         *int64    `json:"canceledAt"            keel:"canceled_at"`
	CancellationReason string    `json:"cancellationReason"    keel:"cancellation_reason,nullzero"`
	Version            int64     `json:"version"               keel:"version,version"`
	CreatedAt          int64     `json:"createdAt"             keel:"created_at,autocreate"`
	UpdatedAt          int64     `json:"updatedAt"             keel:"updated_at,autoupdate"`

	ShipmentMove *ShipmentMove                   `json:"shipmentMove,omitempty" keel:"belongs:shipment_move_id"`
	Carrier      *carrier.Carrier                `json:"carrier,omitempty"      keel:"belongs:carrier_id"`
	Accessorials []*CarrierAssignmentAccessorial `json:"accessorials,omitempty" keel:"many:carrier_assignment_id"`
}

func (ca *CarrierAssignment) applyDefaults() {
	if ca.Status == "" {
		ca.Status = shipmentvocab.CarrierAssignmentStatusPending
	}
	if ca.RateMethod == "" {
		ca.RateMethod = shipmentvocab.CarrierRateMethodFlat
	}
	if ca.CurrencyCode == "" {
		ca.CurrencyCode = "USD"
	}
}

func (ca *CarrierAssignment) Validate(multiErr *errortypes.MultiError) {
	ca.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(
		ca,
		validation.Field(
			&ca.CarrierID,
			validation.Required.Error("Carrier is required"),
		),
		validation.Field(
			&ca.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[shipmentvocab.CarrierAssignmentStatus]("Status is invalid"),
		),
		validation.Field(
			&ca.RateMethod,
			validation.Required.Error("Rate method is required"),
			domainvalidation.ValidEnum[shipmentvocab.CarrierRateMethod]("Rate method is invalid"),
		),
		validation.Field(
			&ca.CancellationReason,
			validation.When(
				ca.Status == shipmentvocab.CarrierAssignmentStatusCanceled,
				validation.Required.Error(
					"A canceled carrier assignment must record its reason",
				),
			),
		),
	))

	if ca.BaseRate.IsNegative() {
		multiErr.Add("baseRate", errortypes.ErrInvalid, "Base rate cannot be negative")
	}
	if ca.FuelSurcharge.IsNegative() {
		multiErr.Add("fuelSurcharge", errortypes.ErrInvalid, "Fuel surcharge cannot be negative")
	}

	for idx, acc := range ca.Accessorials {
		if acc == nil {
			continue
		}
		acc.Validate(multiErr.WithIndex("accessorials", idx))
	}
}

// SyncTotals derives the extended base amount and total cost from the rate
// inputs. distance is the move distance in miles and is only consulted for
// per-mile rates.
func (ca *CarrierAssignment) SyncTotals(distance *float64) {
	ca.applyDefaults()

	ca.BaseAmount = CarrierBaseAmount(ca.RateMethod, ca.BaseRate, distance)

	total := decimal.Zero
	for _, acc := range ca.Accessorials {
		if acc == nil {
			continue
		}
		total = total.Add(acc.Amount)
	}
	ca.AccessorialTotal = total
	ca.TotalCost = ca.BaseAmount.Add(ca.FuelSurcharge).Add(ca.AccessorialTotal)
}

func (ca *CarrierAssignment) IsActive() bool {
	return ca != nil && ca.Status != shipmentvocab.CarrierAssignmentStatusCanceled
}

// Cancel retires the assignment, recording when and why.
func (ca *CarrierAssignment) Cancel(at int64, reason string) {
	ca.Status = shipmentvocab.CarrierAssignmentStatusCanceled
	ca.CanceledAt = &at
	ca.CancellationReason = reason
}

// Confirm marks a pending assignment confirmed and reports whether it moved;
// an assignment in any other state is left as it is.
func (ca *CarrierAssignment) Confirm(at int64) bool {
	if ca.Status != shipmentvocab.CarrierAssignmentStatusPending {
		return false
	}

	ca.Status = shipmentvocab.CarrierAssignmentStatusConfirmed
	ca.ConfirmedAt = &at

	return true
}

// RevertConfirmation returns a confirmed assignment to pending, the
// counterpart of Confirm, and reports whether it moved.
func (ca *CarrierAssignment) RevertConfirmation() bool {
	if ca.Status != shipmentvocab.CarrierAssignmentStatusConfirmed {
		return false
	}

	ca.Status = shipmentvocab.CarrierAssignmentStatusPending
	ca.ConfirmedAt = nil

	return true
}

func (ca *CarrierAssignment) GetID() pulid.ID {
	return ca.ID
}

func (ca *CarrierAssignment) GetCreatedAt() int64 {
	return ca.CreatedAt
}

func (ca *CarrierAssignment) GetTableName() string {
	return "carrier_assignments"
}

func (ca *CarrierAssignment) GetOrganizationID() pulid.ID {
	return ca.OrganizationID
}

func (ca *CarrierAssignment) GetBusinessUnitID() pulid.ID {
	return ca.BusinessUnitID
}

func (ca *CarrierAssignment) BeforeCreate(_ context.Context) error {
	if ca.Status == "" {
		ca.Status = shipmentvocab.CarrierAssignmentStatusPending
	}
	if ca.RateMethod == "" {
		ca.RateMethod = shipmentvocab.CarrierRateMethodFlat
	}
	if ca.CurrencyCode == "" {
		ca.CurrencyCode = "USD"
	}
	ca.applyDefaults()

	return nil
}

func (CarrierAssignment) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_carrier_assignments_active_move",
			Message: "Move already has an active carrier assignment",
		},
		{
			Name:    "fk_carrier_assignments_shipment_move",
			Message: "Move does not exist in your organization",
		},
		{
			Name:    "fk_carrier_assignments_carrier",
			Message: "Carrier does not exist in your organization",
		},
	}
}

type CarrierAssignmentAccessorial struct {
	keel.Table `keel:"carrier_assignment_accessorials"`

	ID                  pulid.ID        `json:"id"                  keel:"id,pk,autoid:casna_"`
	BusinessUnitID      pulid.ID        `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID        `json:"organizationId"      keel:"organization_id,pk,tenant"`
	CarrierAssignmentID pulid.ID        `json:"carrierAssignmentId" keel:"carrier_assignment_id"`
	AccessorialChargeID *pulid.ID       `json:"accessorialChargeId" keel:"accessorial_charge_id"`
	Description         string          `json:"description"         keel:"description"`
	Amount              decimal.Decimal `json:"amount"              keel:"amount"`
	Version             int64           `json:"version"             keel:"version,version"`
	CreatedAt           int64           `json:"createdAt"           keel:"created_at,autocreate"`
	UpdatedAt           int64           `json:"updatedAt"           keel:"updated_at,autoupdate"`

	CarrierAssignment *CarrierAssignment                   `json:"carrierAssignment,omitempty" keel:"belongs:carrier_assignment_id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" keel:"belongs:accessorial_charge_id"`
}

func (caa *CarrierAssignmentAccessorial) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		caa,
		validation.Field(
			&caa.Description,
			validation.Required.Error("Description is required"),
			validation.Length(1, 255).Error("Description must be between 1 and 255 characters"),
		),
	))

	if caa.Amount.IsNegative() {
		multiErr.Add("amount", errortypes.ErrInvalid, "Amount cannot be negative")
	}
}

func (caa *CarrierAssignmentAccessorial) GetID() pulid.ID {
	return caa.ID
}

func (caa *CarrierAssignmentAccessorial) GetTableName() string {
	return "carrier_assignment_accessorials"
}

func (caa *CarrierAssignmentAccessorial) GetOrganizationID() pulid.ID {
	return caa.OrganizationID
}

func (caa *CarrierAssignmentAccessorial) GetBusinessUnitID() pulid.ID {
	return caa.BusinessUnitID
}

func (CarrierAssignmentAccessorial) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_carrier_assignment_accessorials_charge",
			Field:   "accessorials.accessorialChargeId",
			Message: "Accessorial charge does not exist in your organization",
		},
	}
}
