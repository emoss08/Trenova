package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ResolvedFuelSurcharge struct {
	ProgramID           pulid.ID
	AccessorialChargeID pulid.ID
	Amount              decimal.Decimal
	Detail              *shipment.FuelSurchargeDetail
}

// FuelProgramOverride carries the fuel terms a rate agreement negotiated, which
// take precedence over the customer's billing profile.
//
// Fuel is where invoice disputes concentrate, and the reason is nearly always
// that the contract said something the system did not know: a different peg, a
// cap, or an all-in rate with no surcharge at all. Those three are stated here
// rather than left to whoever reads the contract.
//
// It is a neutral value type rather than the agreement's own binding so the
// fuel service stays independent of the rate agreement domain — the same reason
// the resolver takes a shipment and not a rating context.
type FuelProgramOverride struct {
	AgreementID   pulid.ID
	ProgramID     pulid.ID
	Waived        bool
	PegPrice      decimal.NullDecimal
	IncrementRate decimal.NullDecimal
	CapAmount     decimal.NullDecimal
}

// AppliesTerms reports whether the override changes the program's own terms, as
// opposed to only selecting which program to run.
func (o *FuelProgramOverride) AppliesTerms() bool {
	return o != nil &&
		(o.PegPrice.Valid || o.IncrementRate.Valid || o.CapAmount.Valid)
}

type ResolveShipmentChargeRequest struct {
	Shipment         *shipment.Shipment
	Linehaul         decimal.Decimal
	AccessorialTotal decimal.Decimal

	// Override is the contract's fuel binding, when the shipment rated against
	// an agreement that carries one. Nil means the customer's billing profile
	// decides, which is what happened before rate agreements existed.
	Override *FuelProgramOverride
}

type FuelSurchargeResolver interface {
	ResolveShipmentCharge(
		ctx context.Context,
		req *ResolveShipmentChargeRequest,
	) (*ResolvedFuelSurcharge, error)
}
