package statemachine

import (
	"github.com/emoss08/trenova/internal/core/domain/billinglog"
	"github.com/rs/zerolog/log"
)

type BillingStateMachine struct {
	log *billinglog.Log
}

func NewBillingStateMachine(log *billinglog.Log) StateMachine {
	return &BillingStateMachine{
		log: log,
	}
}

func (bs *BillingStateMachine) CurrentState() string {
	return string(bs.log.Status)
}

func (bs *BillingStateMachine) CanTransition(event TransitionEvent) bool {
	currentState := bs.CurrentState()

	validTransitions := map[billinglog.Status]map[TransitionEvent]bool{
		billinglog.StatusDraft: {
			EventBillingBilled:   true,
			EventBillingCanceled: true,
		},
		// Terminal States
		billinglog.StatusCanceled: {}, // * A new invoice will be need to be created
		billinglog.StatusBilled:   {}, // * If the invoice is sent, then a credit memo will be created along with a new invoice
	}

	if transitions, exists := validTransitions[billinglog.Status(currentState)]; exists {
		return transitions[event]
	}

	log.Debug().
		Str("billingID", bs.log.ID.String()).
		Str("event", event.EventType()).
		Str("currentState", currentState).
		Msg("billing transition not allowed")

	return false
}

func (bs *BillingStateMachine) Transition(event TransitionEvent) error {
	if !bs.CanTransition(event) {
		return newTransitionError(
			string(bs.log.Status),
			event,
			"transition not allowed",
		)
	}

	var newStatus billinglog.Status
	switch event {
	case EventBillingDraftCreated:
		newStatus = billinglog.StatusDraft
	case EventBillingBilled:
		newStatus = billinglog.StatusBilled
	case EventBillingCanceled:
		newStatus = billinglog.StatusCanceled
	default:
		return newTransitionError(
			string(bs.log.Status),
			event,
			"unsupported event",
		)
	}

	// Update the status
	bs.log.Status = newStatus

	return nil
}

func (bs *BillingStateMachine) IsInTerminalState() bool {
	return bs.log.Status == billinglog.StatusCanceled || bs.log.Status == billinglog.StatusBilled
}
