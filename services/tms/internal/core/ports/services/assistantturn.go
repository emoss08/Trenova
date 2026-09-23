package services

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ErrNoTurnExecution is what cancelling a turn reports when no execution
// carries it: it was never handed to a worker, or its execution ended without
// closing the record. The turn then has to be closed by whoever asked.
var ErrNoTurnExecution = errors.New("no execution carries this turn")

// AssistantTurnCanceller cancels the execution carrying an assistant turn.
//
// It is the one place that knows how the engine running turns says an
// execution is gone, so a stop pressed in the conversation and a stop made by
// signing out agree on when a turn must be closed by hand.
type AssistantTurnCanceller interface {
	// CancelTurn asks the execution named workflowID to stop. It reports
	// ErrNoTurnExecution when there is no such execution.
	CancelTurn(ctx context.Context, workflowID string) error
}

// AssistantTurnCancellerFunc adapts a function to an AssistantTurnCanceller.
type AssistantTurnCancellerFunc func(ctx context.Context, workflowID string) error

func (f AssistantTurnCancellerFunc) CancelTurn(ctx context.Context, workflowID string) error {
	return f(ctx, workflowID)
}

// StopUserTurnsRequest names a person whose replies are no longer wanted, in
// one tenant.
type StopUserTurnsRequest struct {
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// AssistantTurnStopper ends every reply a person has in flight.
//
// Signing out is the case it exists for: a reply keeps running on a worker, and
// billing, after the tab that asked for it is gone, and once the person has
// signed out nobody is left to read it.
type AssistantTurnStopper interface {
	StopAllForUser(ctx context.Context, req StopUserTurnsRequest) error
}
