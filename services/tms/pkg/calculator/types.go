package calculator

import "github.com/shopspring/decimal"

type ShipmentTotalsResponse struct {
	BaseCharge        decimal.Decimal `json:"baseCharge"`
	OtherChargeAmount decimal.Decimal `json:"otherChargeAmount"`
	TotalChargeAmount decimal.Decimal `json:"totalChargeAmount"`
}
