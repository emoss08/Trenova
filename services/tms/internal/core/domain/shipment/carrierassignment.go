package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type CarrierAssignmentStatus string

const (
	CarrierAssignmentStatusPending   = CarrierAssignmentStatus("Pending")
	CarrierAssignmentStatusConfirmed = CarrierAssignmentStatus("Confirmed")
	CarrierAssignmentStatusCanceled  = CarrierAssignmentStatus("Canceled")
)

func (s CarrierAssignmentStatus) String() string {
	return string(s)
}

func (s CarrierAssignmentStatus) IsValid() bool {
	switch s {
	case CarrierAssignmentStatusPending,
		CarrierAssignmentStatusConfirmed,
		CarrierAssignmentStatusCanceled:
		return true
	}
	return false
}

type CarrierRateMethod string

const (
	CarrierRateMethodFlat    = CarrierRateMethod("Flat")
	CarrierRateMethodPerMile = CarrierRateMethod("PerMile")
)

func (m CarrierRateMethod) String() string {
	return string(m)
}

func (m CarrierRateMethod) Label() string {
	switch m {
	case CarrierRateMethodPerMile:
		return "Per Mile"
	case CarrierRateMethodFlat:
		return "Flat"
	default:
		return string(m)
	}
}

func (m CarrierRateMethod) IsValid() bool {
	switch m {
	case CarrierRateMethodFlat, CarrierRateMethodPerMile:
		return true
	}
	return false
}

// CarrierBaseAmount extends a buy rate into the base amount: per-mile rates
// multiply by the move distance, flat rates pass through. Every surface that
// prices a carrier offer or assignment must share this math so documents and
// settlements cannot diverge on rounding.
func CarrierBaseAmount(
	method CarrierRateMethod,
	rate decimal.Decimal,
	distance *float64,
) decimal.Decimal {
	if method == CarrierRateMethodPerMile {
		miles := decimal.Zero
		if distance != nil {
			miles = decimal.NewFromFloat(*distance)
		}
		return rate.Mul(miles).Round(4)
	}
	return rate
}

var _ bun.BeforeAppendModelHook = (*CarrierAssignment)(nil)

type CarrierAssignment struct {
	bun.BaseModel `bun:"table:carrier_assignments,alias:casn" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	ShipmentMoveID pulid.ID `json:"shipmentMoveId" bun:"shipment_move_id,type:VARCHAR(100),notnull"`
	CarrierID      pulid.ID `json:"carrierId"      bun:"carrier_id,type:VARCHAR(100),notnull"`

	Status                CarrierAssignmentStatus `json:"status"                bun:"status,type:VARCHAR(50),notnull,default:'Pending'"`
	RateMethod            CarrierRateMethod       `json:"rateMethod"            bun:"rate_method,type:VARCHAR(50),notnull,default:'Flat'"`
	BaseRate              decimal.Decimal         `json:"baseRate"              bun:"base_rate,type:NUMERIC(19,4),notnull"`
	BaseAmount            decimal.Decimal         `json:"baseAmount"            bun:"base_amount,type:NUMERIC(19,4),notnull"`
	FuelSurcharge         decimal.Decimal         `json:"fuelSurcharge"         bun:"fuel_surcharge,type:NUMERIC(19,4),notnull,default:0"`
	AccessorialTotal      decimal.Decimal         `json:"accessorialTotal"      bun:"accessorial_total,type:NUMERIC(19,4),notnull,default:0"`
	TotalCost             decimal.Decimal         `json:"totalCost"             bun:"total_cost,type:NUMERIC(19,4),notnull"`
	CurrencyCode          string                  `json:"currencyCode"          bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	ProNumber             string                  `json:"proNumber"             bun:"pro_number,type:VARCHAR(50),nullzero"`
	ExternalDriverName    string                  `json:"externalDriverName"    bun:"external_driver_name,type:VARCHAR(255),nullzero"`
	ExternalDriverPhone   string                  `json:"externalDriverPhone"   bun:"external_driver_phone,type:VARCHAR(20),nullzero"`
	ExternalTractorNumber string                  `json:"externalTractorNumber" bun:"external_tractor_number,type:VARCHAR(50),nullzero"`
	ExternalTrailerNumber string                  `json:"externalTrailerNumber" bun:"external_trailer_number,type:VARCHAR(50),nullzero"`
	AssignedByID          *pulid.ID               `json:"assignedById"          bun:"assigned_by_id,type:VARCHAR(100),nullzero"`
	// RateQuoteID names the buy side quote that decided what this carrier is
	// paid. Absent when somebody typed the rate, which stays the ordinary case
	// until an organization writes carrier contracts.
	RateQuoteID        *pulid.ID `json:"rateQuoteId"        bun:"rate_quote_id,type:VARCHAR(100),nullzero"`
	ConfirmedAt        *int64    `json:"confirmedAt"        bun:"confirmed_at,type:BIGINT,nullzero"`
	CanceledAt         *int64    `json:"canceledAt"         bun:"canceled_at,type:BIGINT,nullzero"`
	CancellationReason string    `json:"cancellationReason" bun:"cancellation_reason,type:TEXT,nullzero"`
	Version            int64     `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt          int64     `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64     `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	ShipmentMove *ShipmentMove                   `json:"shipmentMove,omitempty" bun:"rel:belongs-to,join:shipment_move_id=id"`
	Carrier      *carrier.Carrier                `json:"carrier,omitempty"      bun:"rel:belongs-to,join:carrier_id=id"`
	Accessorials []*CarrierAssignmentAccessorial `json:"accessorials,omitempty" bun:"rel:has-many,join:id=carrier_assignment_id"`
}

func (ca *CarrierAssignment) applyDefaults() {
	if ca.Status == "" {
		ca.Status = CarrierAssignmentStatusPending
	}
	if ca.RateMethod == "" {
		ca.RateMethod = CarrierRateMethodFlat
	}
	if ca.CurrencyCode == "" {
		ca.CurrencyCode = "USD"
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
	return ca != nil && ca.Status != CarrierAssignmentStatusCanceled
}

func (ca *CarrierAssignment) Validate(multiErr *errortypes.MultiError) {
	ca.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(ca,
		validation.Field(&ca.CarrierID,
			validation.Required.Error("Carrier is required"),
		),
		validation.Field(&ca.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[CarrierAssignmentStatus]("Status is invalid"),
		),
		validation.Field(&ca.RateMethod,
			validation.Required.Error("Rate method is required"),
			domainvalidation.ValidEnum[CarrierRateMethod]("Rate method is invalid"),
		),
		validation.Field(&ca.CancellationReason,
			validation.When(
				ca.Status == CarrierAssignmentStatusCanceled,
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

func (ca *CarrierAssignment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ca.ID.IsNil() {
			ca.ID = pulid.MustNew("casn_")
		}
		ca.CreatedAt = now
	case *bun.UpdateQuery:
		ca.UpdatedAt = now
	}

	return nil
}

var _ bun.BeforeAppendModelHook = (*CarrierAssignmentAccessorial)(nil)

type CarrierAssignmentAccessorial struct {
	bun.BaseModel `bun:"table:carrier_assignment_accessorials,alias:casna" json:"-"`

	ID                  pulid.ID        `json:"id"                  bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID      pulid.ID        `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID      pulid.ID        `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	CarrierAssignmentID pulid.ID        `json:"carrierAssignmentId" bun:"carrier_assignment_id,type:VARCHAR(100),notnull"`
	AccessorialChargeID *pulid.ID       `json:"accessorialChargeId" bun:"accessorial_charge_id,type:VARCHAR(100),nullzero"`
	Description         string          `json:"description"         bun:"description,type:VARCHAR(255),notnull"`
	Amount              decimal.Decimal `json:"amount"              bun:"amount,type:NUMERIC(19,4),notnull"`
	Version             int64           `json:"version"             bun:"version,type:BIGINT"`
	CreatedAt           int64           `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64           `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	CarrierAssignment *CarrierAssignment                   `json:"carrierAssignment,omitempty" bun:"rel:belongs-to,join:carrier_assignment_id=id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" bun:"rel:belongs-to,join:accessorial_charge_id=id"`
}

func (caa *CarrierAssignmentAccessorial) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(caa,
		validation.Field(&caa.Description,
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

func (caa *CarrierAssignmentAccessorial) BeforeAppendModel(
	_ context.Context,
	query bun.Query,
) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if caa.ID.IsNil() {
			caa.ID = pulid.MustNew("casna_")
		}
		caa.CreatedAt = now
	case *bun.UpdateQuery:
		caa.UpdatedAt = now
	}

	return nil
}
