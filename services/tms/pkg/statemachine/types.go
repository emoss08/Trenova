package statemachine

import (
	"fmt"
)

type TransitionEvent interface {
	EventType() string
}

type StopTransitionEvent string

const (
	EventStopArrived  = StopTransitionEvent("StopArrived")
	EventStopDeparted = StopTransitionEvent("StopDeparted")
	EventStopCanceled = StopTransitionEvent("StopCanceled")
)

func (e StopTransitionEvent) EventType() string {
	return string(e)
}

type MoveTransitionEvent string

const (
	EventMoveAssigned  = MoveTransitionEvent("MoveAssigned")
	EventMoveStarted   = MoveTransitionEvent("MoveStarted")
	EventMoveCompleted = MoveTransitionEvent("MoveCompleted")
	EventMoveCanceled  = MoveTransitionEvent("MoveCanceled")
)

func (e MoveTransitionEvent) EventType() string {
	return string(e)
}

type ShipmentTransitionEvent string

const (
	EventShipmentAssigned          = ShipmentTransitionEvent("ShipmentAssigned")
	EventShipmentPartiallyAssigned = ShipmentTransitionEvent("ShipmentPartiallyAssigned")
	EventShipmentInTransit         = ShipmentTransitionEvent("ShipmentInTransit")
	EventShipmentCompleted         = ShipmentTransitionEvent("ShipmentCompleted")
	EventShipmentCanceled          = ShipmentTransitionEvent("ShipmentCanceled")
	EventShipmentReadyToBill       = ShipmentTransitionEvent("ShipmentReadyToBill")
	EventShipmentReviewRequired    = ShipmentTransitionEvent("ShipmentReviewRequired")
	EventShipmentBilled            = ShipmentTransitionEvent("ShipmentBilled")
	EventShipmentDelayed           = ShipmentTransitionEvent("ShipmentDelayed")
	EventShipmentPartialCompleted  = ShipmentTransitionEvent("ShipmentPartialCompleted")
)

func (e ShipmentTransitionEvent) EventType() string {
	return string(e)
}

type BillingTransitionEvent string

const (
	EventBillingDraftCreated = BillingTransitionEvent("BillingDraftCreated")
	EventBillingBilled       = BillingTransitionEvent("BillingBilled")
	EventBillingCanceled     = BillingTransitionEvent("BillingCanceled")
)

func (e BillingTransitionEvent) EventType() string {
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

func NewTransitionError(current string, event TransitionEvent, msg string) *TransitionError {
	return &TransitionError{
		CurrentState: current,
		TargetState:  "<unknown>",
		Event:        event,
		Message:      msg,
	}
}

type StateMachine interface {
	CurrentState() string
	CanTransition(event TransitionEvent) bool
	Transition(event TransitionEvent) error
	IsInTerminalState() bool
}
