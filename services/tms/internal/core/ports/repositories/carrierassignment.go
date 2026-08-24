package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type CarrierAccessorialInput struct {
	AccessorialChargeID *pulid.ID       `json:"accessorialChargeId"`
	Description         string          `json:"description"`
	Amount              decimal.Decimal `json:"amount"`
}

type AssignMoveToCarrierRequest struct {
	TenantInfo               pagination.TenantInfo      `json:"-"`
	ShipmentMoveID           pulid.ID                   `json:"shipmentMoveId"`
	CarrierID                pulid.ID                   `json:"carrierId"`
	RateMethod               shipment.CarrierRateMethod `json:"rateMethod"`
	BaseRate                 decimal.Decimal            `json:"baseRate"`
	FuelSurcharge            decimal.Decimal            `json:"fuelSurcharge"`
	Accessorials             []CarrierAccessorialInput  `json:"accessorials"`
	ProNumber                string                     `json:"proNumber"`
	ExternalDriverName       string                     `json:"externalDriverName"`
	ExternalDriverPhone      string                     `json:"externalDriverPhone"`
	ExternalTractorNumber    string                     `json:"externalTractorNumber"`
	ExternalTrailerNumber    string                     `json:"externalTrailerNumber"`
	Replace                  bool                       `json:"replace"`
	OverrideInsuranceWarning bool                       `json:"overrideInsuranceWarning"`

	// AutoRate asks the carrier's contract to price this assignment instead of
	// taking the rate from the request. A rate typed alongside it wins: that is
	// a negotiated number, and overruling it is what makes people switch
	// auto-rating off.
	AutoRate bool `json:"autoRate"`

	// OverrideMarginFloor lets somebody assign a carrier the contract's margin
	// terms would otherwise refuse, the same escape the insurance warning has.
	OverrideMarginFloor bool `json:"overrideMarginFloor"`

	// RateQuoteID is filled in by auto-rating rather than sent by a client. It
	// names the quote that decided the pay.
	RateQuoteID *pulid.ID `json:"-"`
}

func (r *AssignMoveToCarrierRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if r.ShipmentMoveID.IsNil() {
		multiErr.Add("shipmentMoveId", errortypes.ErrRequired, "Shipment move is required")
	}
	if r.CarrierID.IsNil() {
		multiErr.Add("carrierId", errortypes.ErrRequired, "Carrier is required")
	}
	if r.RateMethod != "" && !r.RateMethod.IsValid() {
		multiErr.Add("rateMethod", errortypes.ErrInvalid, "Rate method is invalid")
	}
	if r.BaseRate.IsNegative() {
		multiErr.Add("baseRate", errortypes.ErrInvalid, "Base rate cannot be negative")
	}
	if r.FuelSurcharge.IsNegative() {
		multiErr.Add("fuelSurcharge", errortypes.ErrInvalid, "Fuel surcharge cannot be negative")
	}
	for idx, acc := range r.Accessorials {
		if acc.Description == "" {
			multiErr.WithIndex("accessorials", idx).
				Add("description", errortypes.ErrRequired, "Description is required")
		}
		if acc.Amount.IsNegative() {
			multiErr.WithIndex("accessorials", idx).
				Add("amount", errortypes.ErrInvalid, "Amount cannot be negative")
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type CancelCarrierAssignmentRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ShipmentMoveID pulid.ID              `json:"shipmentMoveId"`
	Reason         string                `json:"reason"`
}

func (r *CancelCarrierAssignmentRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if r.ShipmentMoveID.IsNil() {
		multiErr.Add("shipmentMoveId", errortypes.ErrRequired, "Shipment move is required")
	}
	if r.Reason == "" {
		multiErr.Add("reason", errortypes.ErrRequired, "Cancellation reason is required")
	}

	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

type GetCarrierAssignmentByIDRequest struct {
	TenantInfo          pagination.TenantInfo `json:"-"`
	CarrierAssignmentID pulid.ID              `json:"carrierAssignmentId"`
}

type FindCarrierAssignmentForMatchingRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	CarrierID  pulid.ID              `json:"carrierId"`
	ProNumber  string                `json:"proNumber"`
	ShipmentID pulid.ID              `json:"shipmentId"`
}

type CarrierAssignmentRepository interface {
	GetByID(
		ctx context.Context,
		req *GetCarrierAssignmentByIDRequest,
	) (*shipment.CarrierAssignment, error)
	FindForMatching(
		ctx context.Context,
		req *FindCarrierAssignmentForMatchingRequest,
	) (*shipment.CarrierAssignment, error)
	GetActiveByMoveID(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveID pulid.ID,
	) (*shipment.CarrierAssignment, error)
	Create(
		ctx context.Context,
		entity *shipment.CarrierAssignment,
	) (*shipment.CarrierAssignment, error)
	Update(
		ctx context.Context,
		entity *shipment.CarrierAssignment,
	) (*shipment.CarrierAssignment, error)
}
