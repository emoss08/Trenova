package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// ptoDecider is the slice of the PTO service these tools need. The service
// itself owns the ledger, the overlap check and the driver notification, none
// of which a tool has any business reimplementing.
type ptoDecider interface {
	Approve(
		ctx context.Context,
		req *repositories.UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
	Reject(
		ctx context.Context,
		req *repositories.UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
	Cancel(
		ctx context.Context,
		req *repositories.UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
}

const ptoIDNote = "The id of the time-off request, from list_time_off. " +
	"A worker id is not a request id."

type approveWorkerPTOTool struct {
	pto ptoDecider
}

func newApproveWorkerPTOTool(pto ptoDecider) serviceports.AgentTool {
	return &approveWorkerPTOTool{pto: pto}
}

func (t *approveWorkerPTOTool) Name() string { return "approve_worker_pto" }

func (t *approveWorkerPTOTool) Description() string {
	return "Approve a worker's time-off request. Approving books the days against " +
		"their balance and takes them off the board for those dates, so check what " +
		"they are covering before you propose it. Only a request that is still " +
		"awaiting a decision can be approved."
}

func (t *approveWorkerPTOTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ptoId": map[string]any{"type": "string", "description": ptoIDNote},
			"reason": map[string]any{
				"type": "string",
				"description": "An optional note recorded with the decision. " +
					"Leave it out unless you were given one.",
			},
		},
		"required":             []string{"ptoId"},
		"additionalProperties": false,
	}
}

// Approved time off can be cancelled, which releases the booked days.
func (t *approveWorkerPTOTool) Reversible() bool { return true }

func (t *approveWorkerPTOTool) PermissionResource() permission.Resource {
	return permission.ResourceWorkerPTO
}

// Approve, not update. The guard refuses an approval to an agent principal, so
// naming the operation honestly is also what keeps a scheduled agent from
// deciding a person's leave on its own.
func (t *approveWorkerPTOTool) PermissionOperation() permission.Operation {
	return permission.OpApprove
}

func (t *approveWorkerPTOTool) RequiresIdempotencyKey() bool { return false }

func (t *approveWorkerPTOTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *approveWorkerPTOTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	request, err := ptoStatusRequest(params)
	if err != nil {
		return err
	}

	_, err = t.pto.Approve(ctx, request)

	return err
}

type rejectWorkerPTOTool struct {
	pto ptoDecider
}

func newRejectWorkerPTOTool(pto ptoDecider) serviceports.AgentTool {
	return &rejectWorkerPTOTool{pto: pto}
}

func (t *rejectWorkerPTOTool) Name() string { return "reject_worker_pto" }

func (t *rejectWorkerPTOTool) Description() string {
	return "Decline a worker's time-off request. A rejection is final — the request " +
		"cannot be reopened, and the worker is told why — so the reason must be the " +
		"one you were actually given, not one you composed."
}

func (t *rejectWorkerPTOTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ptoId": map[string]any{"type": "string", "description": ptoIDNote},
			"reason": map[string]any{
				"type": "string",
				"description": "Why it is being declined. Required, and shown to the " +
					"worker. If nobody told you why, ask rather than guessing.",
			},
		},
		"required":             []string{"ptoId", "reason"},
		"additionalProperties": false,
	}
}

// Rejected is a terminal state. The worker has to file again.
func (t *rejectWorkerPTOTool) Reversible() bool { return false }

func (t *rejectWorkerPTOTool) PermissionResource() permission.Resource {
	return permission.ResourceWorkerPTO
}

func (t *rejectWorkerPTOTool) PermissionOperation() permission.Operation {
	return permission.OpReject
}

func (t *rejectWorkerPTOTool) RequiresIdempotencyKey() bool { return false }

func (t *rejectWorkerPTOTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *rejectWorkerPTOTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	request, err := ptoStatusRequest(params)
	if err != nil {
		return err
	}

	// The service rejects an empty reason too. Failing here names the parameter
	// the model left out, which is the correction it can act on.
	if _, err = requireString(params.Params, "reason"); err != nil {
		return err
	}

	_, err = t.pto.Reject(ctx, request)

	return err
}

type cancelWorkerPTOTool struct {
	pto ptoDecider
}

func newCancelWorkerPTOTool(pto ptoDecider) serviceports.AgentTool {
	return &cancelWorkerPTOTool{pto: pto}
}

func (t *cancelWorkerPTOTool) Name() string { return "cancel_worker_pto" }

func (t *cancelWorkerPTOTool) Description() string {
	return "Cancel a worker's time off, whether it was still awaiting a decision or " +
		"already approved. Cancelling approved time off returns the days to their " +
		"balance and puts them back on the board for those dates."
}

func (t *cancelWorkerPTOTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"ptoId": map[string]any{"type": "string", "description": ptoIDNote},
			"reason": map[string]any{
				"type":        "string",
				"description": "Why it is being cancelled. Recorded with the change.",
			},
		},
		"required":             []string{"ptoId", "reason"},
		"additionalProperties": false,
	}
}

// Cancelled is terminal; the days come back but the request does not.
func (t *cancelWorkerPTOTool) Reversible() bool { return false }

func (t *cancelWorkerPTOTool) PermissionResource() permission.Resource {
	return permission.ResourceWorkerPTO
}

func (t *cancelWorkerPTOTool) PermissionOperation() permission.Operation {
	return permission.OpCancel
}

func (t *cancelWorkerPTOTool) RequiresIdempotencyKey() bool { return false }

func (t *cancelWorkerPTOTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *cancelWorkerPTOTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	request, err := ptoStatusRequest(params)
	if err != nil {
		return err
	}

	if _, err = requireString(params.Params, "reason"); err != nil {
		return err
	}

	_, err = t.pto.Cancel(ctx, request)

	return err
}

// ptoStatusRequest builds the decision envelope the three transitions share.
//
// ExpectedVersion is deliberately left at zero: the service reads the record's
// current version when the caller does not supply one, and a model has no way
// to know a version. Leaving it out is the honest thing; inventing one would
// defeat the concurrency check it exists to run.
func ptoStatusRequest(
	params serviceports.ToolExecuteParams,
) (*repositories.UpdatePTOStatusRequest, error) {
	id, err := requirePulid(params.Params, "ptoId")
	if err != nil {
		return nil, err
	}

	return &repositories.UpdatePTOStatusRequest{
		ID:         id,
		TenantInfo: tenantFrom(params),
		UserID:     params.Actor.UserID,
		Reason:     optionalString(params.Params, "reason"),
	}, nil
}
