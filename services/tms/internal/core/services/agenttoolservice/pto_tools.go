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

func (t *approveWorkerPTOTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceWorkerPTO,
		Operation:     permission.OpApprove,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Books approved time off; the worker is told it was approved but reads no " +
			"text the model wrote.",
	}
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

func (t *rejectWorkerPTOTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceWorkerPTO,
		Operation:     permission.OpReject,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressDriverVisible},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "The worker is sent the rejection reason by push notice and text message.",
	}
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

func (t *cancelWorkerPTOTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceWorkerPTO,
		Operation:     permission.OpCancel,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressDriverVisible},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "The worker is sent the cancellation reason by push notice and text " +
			"message.",
	}
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

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *approveWorkerPTOTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "ptoId", permission.ResourceWorkerPTO)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *rejectWorkerPTOTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "ptoId", permission.ResourceWorkerPTO)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *cancelWorkerPTOTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "ptoId", permission.ResourceWorkerPTO)
}
