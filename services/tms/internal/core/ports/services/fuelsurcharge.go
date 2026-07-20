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

type ResolveShipmentChargeRequest struct {
	Shipment         *shipment.Shipment
	Linehaul         decimal.Decimal
	AccessorialTotal decimal.Decimal
}

type FuelSurchargeResolver interface {
	ResolveShipmentCharge(
		ctx context.Context,
		req *ResolveShipmentChargeRequest,
	) (*ResolvedFuelSurcharge, error)
}
