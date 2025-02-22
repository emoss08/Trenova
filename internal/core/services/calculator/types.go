package calculator

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
