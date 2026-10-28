package operationstools

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
)

const paramPTOID = "ptoId"

type ApproveWorkerPTOTool struct {
	pto common.PtoApprover
}

func NewApproveWorkerPTOTool(pto common.PtoApprover) agenttoolports.AgentTool {
	return &ApproveWorkerPTOTool{pto: pto}
}

func (t *ApproveWorkerPTOTool) Name() string { return "worker_approve_pto" }

func (t *ApproveWorkerPTOTool) Recipe() []string {
	return []string{"worker_list_time_off", "worker_approve_pto"}
}

func (t *ApproveWorkerPTOTool) Description() string {
	return "Approve a worker's time-off request. Approving books the days against " +
		"their balance and takes them off the board for those dates. When you hold " +
		"worker_get_schedule, say what they are covering on those days beside the " +
		"proposal; when you do not, propose it anyway and say coverage was not checked, " +
		"since the person approving weighs it. Only a request still awaiting a decision " +
		"can be approved; for several, propose each."
}

func (t *ApproveWorkerPTOTool) SearchTerms() []string {
	return []string{"approve time off", "approve vacation", "approve pto request"}
}

func (t *ApproveWorkerPTOTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramPTOID: agenttoolschema.RecordIDText(
				permission.ResourceWorkerPTO,
				common.PtoIDNote,
			),
			common.ParamReason: map[string]any{
				"type": "string",
				"description": "An optional note recorded with the decision. " +
					"Leave it out unless you were given one.",
			},
		},
		"required":             []string{paramPTOID},
		"additionalProperties": false,
	}
}

func (t *ApproveWorkerPTOTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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

func (t *ApproveWorkerPTOTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if err := common.GuardExecute(t, params); err != nil {
		return err
	}

	request, err := ptoStatusRequest(params)
	if err != nil {
		return err
	}

	_, err = t.pto.Approve(ctx, request)

	return err
}

type RejectWorkerPTOTool struct {
	pto ptoRejecter
}

func NewRejectWorkerPTOTool(pto ptoRejecter) agenttoolports.AgentTool {
	return &RejectWorkerPTOTool{pto: pto}
}

func (t *RejectWorkerPTOTool) Name() string { return "worker_reject_pto" }

func (t *RejectWorkerPTOTool) Description() string {
	return "Decline a worker's time-off request. A rejection is final — the request " +
		"cannot be reopened, and the worker is told why — so the reason must be the " +
		"one you were actually given, not one you composed."
}

func (t *RejectWorkerPTOTool) SearchTerms() []string {
	return []string{"reject time off", "deny vacation", "decline pto request"}
}

func (t *RejectWorkerPTOTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramPTOID: agenttoolschema.RecordIDText(
				permission.ResourceWorkerPTO,
				common.PtoIDNote,
			),
			common.ParamReason: map[string]any{
				"type": "string",
				"description": "Why it is being declined. Required, and shown to the " +
					"worker. If nobody told you why, ask rather than guessing.",
			},
		},
		"required":             []string{paramPTOID, common.ParamReason},
		"additionalProperties": false,
	}
}

func (t *RejectWorkerPTOTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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

func (t *RejectWorkerPTOTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.pto.Reject(ctx, request)

	return err
}

func (t *RejectWorkerPTOTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*workerptoports.UpdatePTOStatusRequest, error) {
	return ptoDecisionRequest(t, params)
}

type CancelWorkerPTOTool struct {
	pto ptoRejecter
}

func NewCancelWorkerPTOTool(pto ptoRejecter) agenttoolports.AgentTool {
	return &CancelWorkerPTOTool{pto: pto}
}

func (t *CancelWorkerPTOTool) Name() string { return "worker_cancel_pto" }

func (t *CancelWorkerPTOTool) SearchTerms() []string {
	return []string{"cancel vacation", "cancel time off", "cancel approved vacation"}
}

func (t *CancelWorkerPTOTool) Description() string {
	return "Cancel a worker's time off, whether it was still awaiting a decision or " +
		"already approved. Cancelling approved time off returns the days to their " +
		"balance and puts them back on the board for those dates."
}

func (t *CancelWorkerPTOTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			paramPTOID: agenttoolschema.RecordIDText(
				permission.ResourceWorkerPTO,
				common.PtoIDNote,
			),
			common.ParamReason: map[string]any{
				"type":        "string",
				"description": "Why it is being cancelled. Recorded with the change.",
			},
		},
		"required":             []string{paramPTOID, common.ParamReason},
		"additionalProperties": false,
	}
}

func (t *CancelWorkerPTOTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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

func (t *CancelWorkerPTOTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.pto.Cancel(ctx, request)

	return err
}

func (t *CancelWorkerPTOTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*workerptoports.UpdatePTOStatusRequest, error) {
	return ptoDecisionRequest(t, params)
}

// ptoDecisionRequest is a decision the driver is told the reason for. The
// service refuses an empty reason too; failing here names the parameter the
// model left out, which is the correction it can act on.
func ptoDecisionRequest(
	tool agenttoolports.AgentTool,
	params *agenttoolports.ToolExecuteParams,
) (*workerptoports.UpdatePTOStatusRequest, error) {
	if err := common.GuardExecute(tool, *params); err != nil {
		return nil, err
	}

	request, err := ptoStatusRequest(*params)
	if err != nil {
		return nil, err
	}

	if _, err = common.RequireString(params.Params, common.ParamReason); err != nil {
		return nil, err
	}

	return request, nil
}

// ptoStatusRequest builds the decision envelope the three transitions share.
//
// ExpectedVersion is deliberately left at zero: the service reads the record's
// current version when the caller does not supply one, and a model has no way
// to know a version. Leaving it out is the honest thing; inventing one would
// defeat the concurrency check it exists to run.
func ptoStatusRequest(
	params agenttoolports.ToolExecuteParams,
) (*workerptoports.UpdatePTOStatusRequest, error) {
	id, err := common.RequirePulid(params.Params, paramPTOID)
	if err != nil {
		return nil, err
	}

	return &workerptoports.UpdatePTOStatusRequest{
		ID:         id,
		TenantInfo: common.TenantFrom(params),
		UserID:     params.Actor.UserID,
		Reason:     common.OptionalString(params.Params, common.ParamReason),
	}, nil
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *ApproveWorkerPTOTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, paramPTOID, permission.ResourceWorkerPTO)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *RejectWorkerPTOTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, paramPTOID, permission.ResourceWorkerPTO)
}

// Target names the record this call would change, so a proposal to change it
// can be checked against the record's version before it runs.
func (t *CancelWorkerPTOTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, paramPTOID, permission.ResourceWorkerPTO)
}
