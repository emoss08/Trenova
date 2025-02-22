package statemachine

import (
	"context"
	"fmt"
)

type TransitionEvent string

const (
	// * Stop Events
	EventStopArrived  = TransitionEvent("StopArrived")
	EventStopDeparted = TransitionEvent("StopDeparted")
	EventStopCanceled = TransitionEvent("StopCanceled")

	// * Move Events
	EventMoveAssigned  = TransitionEvent("MoveAssigned")
	EventMoveStarted   = TransitionEvent("MoveStarted")
	EventMoveCompleted = TransitionEvent("MoveCompleted")
	EventMoveCanceled  = TransitionEvent("MoveCanceled")

	// * Shipment Events
	EventShipmentAssigned          = TransitionEvent("ShipmentAssigned")
	EventShipmentPartiallyAssigned = TransitionEvent("ShipmentPartiallyAssigned")
	EventShipmentInTransit         = TransitionEvent("ShipmentInTransit")
	EventShipmentCompleted         = TransitionEvent("ShipmentCompleted")
	EventShipmentCanceled          = TransitionEvent("ShipmentCanceled")
	EventShipmentDelayed           = TransitionEvent("ShipmentDelayed")
	EventShipmentPartialCompleted  = TransitionEvent("ShipmentPartialCompleted")
	EventShipmentMarkedForBilling  = TransitionEvent("ShipmentMarkedForBilling")
)

type TransitionError struct {
	CurrentState string `json:"currentState"`
	TargetState  string `json:"targetState"`
	Event        TransitionEvent
	Message      string `json:"message"`
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("invalid transition from %s --[%s]--> %s: %s",
		e.CurrentState, string(e.Event), e.TargetState, e.Message)
}

func newTransitionError(current, target string, event TransitionEvent, msg string) *TransitionError {
	return &TransitionError{
		CurrentState: current,
		TargetState:  target,
		Event:        event,
		Message:      msg,
	}
}

type StateMachine interface {
	// CurrentState returns the current state of the entity
	CurrentState() string

	// CanTransition checks if a transition is possible given an event
	CanTransition(ctx context.Context, event TransitionEvent) bool

	// Transition attempts to transition to a new state based on an event
	Transition(ctx context.Context, event TransitionEvent) error

	// IsInTerminalState returns true if current is terminal
	IsInTerminalState() bool
}
