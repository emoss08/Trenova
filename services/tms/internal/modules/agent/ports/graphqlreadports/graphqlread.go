package graphqlreadports

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// ErrUnknownReadOperation is a read naming an operation no tool is bound to.
// The reader runs only the persisted operations its bindings name, never
// query text, so the safelist holds in every environment.
var ErrUnknownReadOperation = errors.New("no agent read is bound to this operation")

// GraphQLReader runs a persisted GraphQL read as the actor, through the
// same executor, extensions, permission checks, field guard, row-level
// security and cost budget the screen's request goes through. The agent
// module declares it; the API package provides it.
type GraphQLReader interface {
	Read(ctx context.Context, req *GraphQLReadRequest) (*GraphQLReadResult, error)
}

type GraphQLReadRequest struct {
	Actor     *permissionports.RequestActor
	Operation string
	Variables map[string]any
	// DataAccessCeiling is the agent definition's data access; the reader
	// lowers it to the person's role ceiling per resource when a person is
	// the actor.
	DataAccessCeiling permission.FieldSensitivity
	RequestID         string
}

type GraphQLReadResult struct {
	Data   map[string]any
	Errors []GraphQLReadError
	// Withheld is every field the read did not show, by its path without
	// list indices.
	Withheld []string
	// Cost is the operation's complexity, which the caller's budget was
	// charged.
	Cost int
	// RetryAfter is how long the caller's budget needs before a read of
	// this cost would be admitted, when the budget refused it.
	RetryAfter time.Duration
}

type GraphQLReadError struct {
	Message string
	Path    string
	Code    string
}

const (
	GraphQLCodeForbidden      = string(errortypes.ErrForbidden)
	GraphQLCodeUnauthorized   = string(errortypes.ErrUnauthorized)
	GraphQLCodeNotFound       = string(errortypes.ErrNotFound)
	GraphQLCodeBudgetExceeded = "OPERATION_COST_BUDGET_EXCEEDED"
)

// ReadRefusedError is a read the resolver refused: the same refusal the
// person's screen would show. dispatch reports it as denied, with the advice
// it gives for a permission the person lacks, never as a failed call.
type ReadRefusedError struct {
	Resource permission.Resource
	Message  string
}

func (e *ReadRefusedError) Error() string {
	return "the read was refused: " + e.Message
}
