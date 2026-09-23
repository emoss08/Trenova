package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ErrRunStepClaimed reports that the step key already belongs to another
// attempt. It is not a failure: it is how a retry learns the work was already
// begun.
var ErrRunStepClaimed = errors.New("agent run step is already claimed")

// ListAgentRunStepsRequest reads one run's ledger, oldest first.
type ListAgentRunStepsRequest struct {
	TenantInfo pagination.TenantInfo
	OwnerKind  string
	OwnerID    pulid.ID
}

// SettleAgentRunStepRequest closes a claimed step.
type SettleAgentRunStepRequest struct {
	TenantInfo pagination.TenantInfo
	OwnerID    pulid.ID
	StepKey    string
	Status     string
	Outcome    map[string]any
}

// PruneAgentRunStepsRequest drops steps older than Before, up to Limit rows.
// The caller loops until a batch comes back short, so no single statement
// holds a long transaction.
type PruneAgentRunStepsRequest struct {
	Before int64
	Limit  int
}

type AgentRunStepRepository interface {
	// Claim inserts a Started step. A key that is already taken returns
	// ErrRunStepClaimed together with the row that holds it, so the caller can
	// tell a finished step from one that was only begun.
	Claim(ctx context.Context, step *agent.AgentRunStep) (*agent.AgentRunStep, error)
	Settle(ctx context.Context, req SettleAgentRunStepRequest) error
	// Record files a step that needs no claim, such as a model reply.
	Record(ctx context.Context, step *agent.AgentRunStep) error
	List(ctx context.Context, req ListAgentRunStepsRequest) ([]*agent.AgentRunStep, error)
	Prune(ctx context.Context, req PruneAgentRunStepsRequest) (int, error)
}
