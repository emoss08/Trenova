package statemachine

import (
	"fmt"
)

type StopTransitionEvent string

const (
	EventStopArrived  = StopTransitionEvent("StopArrived")
	EventStopDeparted = StopTransitionEvent("StopDeparted")
	EventStopCanceled = StopTransitionEvent("StopCanceled")
)

type MoveTransitionEvent string

const (
	EventMoveAssigned  = MoveTransitionEvent("MoveAssigned")
	EventMoveStarted   = MoveTransitionEvent("MoveStarted")
	EventMoveCompleted = MoveTransitionEvent("MoveCompleted")
	EventMoveCanceled  = MoveTransitionEvent("MoveCanceled")
)

type ShipmentTransitionEvent string

const (
	EventShipmentAssigned          = ShipmentTransitionEvent("ShipmentAssigned")
	EventShipmentPartiallyAssigned = ShipmentTransitionEvent("ShipmentPartiallyAssigned")
	EventShipmentInTransit         = ShipmentTransitionEvent("ShipmentInTransit")
	EventShipmentCompleted         = ShipmentTransitionEvent("ShipmentCompleted")
	EventShipmentCanceled          = ShipmentTransitionEvent("ShipmentCanceled")
	EventShipmentDelayed           = ShipmentTransitionEvent("ShipmentDelayed")
	EventShipmentPartialCompleted  = ShipmentTransitionEvent("ShipmentPartialCompleted")
	EventShipmentMarkedForBilling  = ShipmentTransitionEvent("ShipmentMarkedForBilling")
)

// TransitionEvent is a common interface for all transition event types
type TransitionEvent interface {
	EventType() string
}

// Implement the TransitionEvent interface for each event type
func (e StopTransitionEvent) EventType() string {
	return string(e)
}

func (e MoveTransitionEvent) EventType() string {
	return string(e)
}

func (e ShipmentTransitionEvent) EventType() string {
	return string(e)
}

type TransitionError struct {
	CurrentState string `json:"currentState"`
	TargetState  string `json:"targetState"`
	Event        TransitionEvent
	Message      string `json:"message"`
}

func (e *TransitionError) Error() string {
	return fmt.Sprintf("invalid transition from %s --[%s]--> %s: %s",
		e.CurrentState, e.Event.EventType(), e.TargetState, e.Message)
}

func newTransitionError(current string, event TransitionEvent, msg string) *TransitionError {
	return &TransitionError{
		CurrentState: current,
		TargetState:  "<unknown>",
		Event:        event,
		Message:      msg,
	}
}

type StateMachine interface {
	// CurrentState returns the current state of the entity
	CurrentState() string

	// CanTransition checks if a transition is possible given an event
	CanTransition(event TransitionEvent) bool

	// Transition attempts to transition to a new state based on an event
	Transition(event TransitionEvent) error

	// IsInTerminalState returns true if current is terminal
	IsInTerminalState() bool
}
