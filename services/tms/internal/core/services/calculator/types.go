/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package calculator

import "github.com/shopspring/decimal"

type ShipmentTotalsResponse struct {
	BaseCharge        decimal.Decimal `json:"baseCharge"`
	OtherChargeAmount decimal.Decimal `json:"otherChargeAmount"`
	TotalChargeAmount decimal.Decimal `json:"totalChargeAmount"`
}

// StopState represents the possible states a stop can be in based on its data
type StopState int

const (
	StopStateNew StopState = iota
	StopStateInTransit
	StopStateCompleted
	StopStateCanceled
)

// MoveState represents the possible states a move can be in based on its data
type MoveState int

const (
	MoveStateNew MoveState = iota
	MoveStateAssigned
	MoveStateInTransit
	MoveStateCompleted
	MoveStateCanceled
)

// ShipmentState represents the possible states a shipment can be in based on its data
type ShipmentState int

const (
	ShipmentStateNew ShipmentState = iota
	ShipmentStatePartiallyAssigned
	ShipmentStateAssigned
	ShipmentStateInTransit
	ShipmentStateDelayed
	ShipmentStatePartiallyCompleted
	ShipmentStateCompleted
	ShipmentStateBilled
	ShipmentStateCanceled
)
