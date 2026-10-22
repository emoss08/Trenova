package rules

import (
	"context"

	"github.com/shopspring/decimal"
)

const (
	OpAllocationSplit       = "allocation.split"
	OpAllocationChargeTotal = "allocation.chargeTotal"
	OpAllocationRebalance   = "allocation.rebalance"

	AllocationKindFreight     = "Freight"
	AllocationKindAccessorial = "Accessorial"
	AllocationKindOrderCharge = "OrderCharge"
)

type AllocationRow struct {
	ID               string              `json:"id,omitempty"`
	BillToCustomerID string              `json:"billToCustomerId"`
	Method           string              `json:"method"`
	Percent          decimal.NullDecimal `json:"percent"`
	Amount           decimal.NullDecimal `json:"amount"`
	Sequence         int64               `json:"sequence"`
}

type AllocationCharge struct {
	Kind        string              `json:"kind,omitempty"`
	Total       decimal.NullDecimal `json:"total"`
	Method      string              `json:"method,omitempty"`
	Amount      decimal.NullDecimal `json:"amount"`
	Unit        int64               `json:"unit"`
	Allocations []AllocationRow     `json:"allocations"`
}

type AllocationSplitRequest struct {
	DefaultPayerID string             `json:"defaultPayerId"`
	Base           decimal.Decimal    `json:"base"`
	Charges        []AllocationCharge `json:"charges"`
}

type AllocationShare struct {
	Allocation       *int                `json:"allocation"`
	AllocationID     string              `json:"allocationId"`
	BillToCustomerID string              `json:"billToCustomerId"`
	Method           string              `json:"method"`
	Amount           decimal.Decimal     `json:"amount"`
	Percent          decimal.NullDecimal `json:"percent"`
	Partial          bool                `json:"partial"`
}

type AllocationChargeSplit struct {
	Total  decimal.Decimal   `json:"total"`
	Shares []AllocationShare `json:"shares"`
	Error  *Message          `json:"error"`
}

type AllocationPayer struct {
	PayerID           string          `json:"payerId"`
	FreightAmount     decimal.Decimal `json:"freightAmount"`
	AccessorialAmount decimal.Decimal `json:"accessorialAmount"`
	TotalAmount       decimal.Decimal `json:"totalAmount"`
}

type AllocationSplit struct {
	Charges []AllocationChargeSplit `json:"charges"`
	Payers  []AllocationPayer       `json:"payers"`
	IsSplit bool                    `json:"isSplit"`
}

func (s *AllocationSplit) FirstError() (int, *Message) {
	for i := range s.Charges {
		if s.Charges[i].Error != nil {
			return i, s.Charges[i].Error
		}
	}
	return -1, nil
}

type AllocationChargesRequest struct {
	Base    decimal.Decimal    `json:"base"`
	Charges []AllocationCharge `json:"charges"`
}

type AllocationTotals struct {
	Totals []decimal.Decimal `json:"totals"`
	Sum    decimal.Decimal   `json:"sum"`
}

type AllocationRebalanceCharge struct {
	Stale          bool                  `json:"stale"`
	AllocatedTotal decimal.NullDecimal   `json:"allocatedTotal"`
	ChargeTotal    decimal.NullDecimal   `json:"chargeTotal"`
	Percents       []decimal.NullDecimal `json:"percents"`
}

type AllocationRebalance struct {
	Charges   []AllocationRebalanceCharge `json:"charges"`
	Converted int                         `json:"converted"`
}

func SplitCharges(ctx context.Context, req *AllocationSplitRequest) (*AllocationSplit, error) {
	var out AllocationSplit
	if err := Call(ctx, OpAllocationSplit, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func ChargeTotals(ctx context.Context, req *AllocationChargesRequest) (*AllocationTotals, error) {
	var out AllocationTotals
	if err := Call(ctx, OpAllocationChargeTotal, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func RebalanceCharges(
	ctx context.Context,
	req *AllocationChargesRequest,
) (*AllocationRebalance, error) {
	var out AllocationRebalance
	if err := Call(ctx, OpAllocationRebalance, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
