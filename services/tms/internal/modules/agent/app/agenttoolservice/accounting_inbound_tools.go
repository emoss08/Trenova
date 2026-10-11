package agenttoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	paramInboundChangeID   = "inboundChangeId"
	paramInboundNote       = "note"
	toolListInbound        = "list_accounting_inbound_changes"
	fieldInboundStatus     = "status"
	maxInboundNoteRunes    = 500
	inboundChangeParamHelp = "The payment's id, from list_accounting_inbound_changes."
)

type accountingInboundOperator interface {
	Get(
		ctx context.Context,
		req *accountingsyncports.GetAccountingInboundChangeRequest,
	) (*accountingsync.AccountingInboundChange, error)
	PreviewApply(
		ctx context.Context,
		req *accountingsyncports.GetAccountingInboundChangeRequest,
	) (*accountingsyncports.AccountingInboundApplyPreview, error)
	Apply(
		ctx context.Context,
		req *accountingsyncports.DecideAccountingInboundChangeRequest,
		actor *permissionports.RequestActor,
	) (*accountingsync.AccountingInboundChange, error)
	Ignore(
		ctx context.Context,
		req *accountingsyncports.DecideAccountingInboundChangeRequest,
		actor *permissionports.RequestActor,
	) (*accountingsync.AccountingInboundChange, error)
}

func inboundLabel(change *accountingsync.AccountingInboundChange) string {
	noun := "Payment"
	if change.Kind == accountingsync.InboundBillPayment {
		noun = "Bill payment"
	}
	parts := []string{noun}
	if change.ExternalNumber != "" {
		parts = append(parts, change.ExternalNumber)
	}
	if change.PartyName != "" {
		parts = append(parts, "from "+change.PartyName)
	}
	return strings.Join(parts, " ")
}

func inboundRequest(
	params *agentports.ToolExecuteParams,
) (*accountingsyncports.GetAccountingInboundChangeRequest, error) {
	id, err := requirePulid(params.Params, paramInboundChangeID)
	if err != nil {
		return nil, err
	}
	return &accountingsyncports.GetAccountingInboundChangeRequest{
		TenantInfo: tenantFrom(*params),
		ID:         id,
	}, nil
}

type applyAccountingInboundChangeTool struct {
	inbound accountingInboundOperator
}

func newApplyAccountingInboundChangeTool(inbound accountingInboundOperator) agentports.AgentTool {
	return &applyAccountingInboundChangeTool{inbound: inbound}
}

func provideApplyAccountingInboundChangeTool(
	inbound accountingsyncports.AccountingInboundService,
) agentports.AgentTool {
	return newApplyAccountingInboundChangeTool(inbound)
}

func (t *applyAccountingInboundChangeTool) Name() string {
	return "apply_accounting_inbound_change"
}

func (t *applyAccountingInboundChangeTool) Recipe() []string {
	return []string{"list_accounting_inbound_changes", "apply_accounting_inbound_change"}
}

func (t *applyAccountingInboundChangeTool) Description() string {
	return "Accept an inbound change: one payment the accounting system recorded against " +
		"documents Trenova sent, shown as Proposed by list_accounting_inbound_changes. " +
		"Accepting brings it into Trenova on the day it was made, with any credits it used, " +
		"or marks the settlement it pays as paid, and only while it still matches what is " +
		"open. The preview shows exactly what would post."
}

func (t *applyAccountingInboundChangeTool) SearchTerms() []string {
	return []string{"accept inbound change", "customer payment from accounting"}
}

func (t *applyAccountingInboundChangeTool) Prerequisites() []string {
	return []string{toolListInbound}
}

func (t *applyAccountingInboundChangeTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramInboundChangeID: agenttoolschema.KindID(
			inboundChangeParamHelp,
			permission.KindAccountingInboundChange,
		),
	}, paramInboundChangeID)
}

func (t *applyAccountingInboundChangeTool) Policy() agentports.ToolPolicy {
	return agentports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceAccountingSync,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Posts money in Trenova: a customer payment or a settlement payment, so " +
			"a person approves each one.",
	}
}

// Target names the payment this call applies, so a proposal is refused when the
// payment was applied, ignored or voided before it was approved.
func (t *applyAccountingInboundChangeTool) Target(
	params map[string]any,
) (agentports.ToolTarget, bool) {
	return targetOf(params, paramInboundChangeID, permission.ResourceAccountingSync)
}

func (t *applyAccountingInboundChangeTool) preview(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*accountingsyncports.AccountingInboundApplyPreview, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}
	req, err := inboundRequest(params)
	if err != nil {
		return nil, err
	}
	preview, err := t.inbound.PreviewApply(ctx, req)
	if err != nil {
		return nil, err
	}
	if !preview.CanApply {
		return nil, errortypes.NewBusinessError(
			"{0} cannot be applied: {1}",
			inboundLabel(preview.Change),
			preview.Blocker,
		)
	}
	return preview, nil
}

func (t *applyAccountingInboundChangeTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.preview(ctx, &params)
	return err
}

func (t *applyAccountingInboundChangeTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if _, err := t.preview(ctx, &params); err != nil {
		return err
	}
	req, err := inboundRequest(&params)
	if err != nil {
		return err
	}
	_, err = t.inbound.Apply(ctx, &accountingsyncports.DecideAccountingInboundChangeRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	}, params.Actor)
	return err
}

type ignoreAccountingInboundChangeTool struct {
	inbound accountingInboundOperator
}

func newIgnoreAccountingInboundChangeTool(
	inbound accountingInboundOperator,
) agentports.AgentTool {
	return &ignoreAccountingInboundChangeTool{inbound: inbound}
}

func provideIgnoreAccountingInboundChangeTool(
	inbound accountingsyncports.AccountingInboundService,
) agentports.AgentTool {
	return newIgnoreAccountingInboundChangeTool(inbound)
}

func (t *ignoreAccountingInboundChangeTool) Name() string {
	return "ignore_accounting_inbound_change"
}

func (t *ignoreAccountingInboundChangeTool) Description() string {
	return "Dismiss an inbound change, with the reason, so the accounting system's payment " +
		"is never brought into Trenova. Use it when a person says the payment was already " +
		"keyed in Trenova, or belongs to something Trenova does not track."
}

func (t *ignoreAccountingInboundChangeTool) SearchTerms() []string {
	return []string{"dismiss inbound change", "already keyed"}
}

func (t *ignoreAccountingInboundChangeTool) Prerequisites() []string {
	return []string{toolListInbound}
}

func (t *ignoreAccountingInboundChangeTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramInboundChangeID: agenttoolschema.KindID(
			inboundChangeParamHelp,
			permission.KindAccountingInboundChange,
		),
		paramInboundNote: jsonschemautils.Text(
			"Why it stays out of Trenova, in one sentence a bookkeeper would accept.",
		),
	}, paramInboundChangeID, paramInboundNote)
}

func (t *ignoreAccountingInboundChangeTool) Policy() agentports.ToolPolicy {
	return accountingSyncPolicy(t.Name(), accountingSyncPolicySpec{
		resource:  permission.ResourceAccountingSync,
		operation: permission.OpUpdate,
		tier:      agent.TierActWithApproval,
		rationale: "Keeps a payment the books hold out of Trenova for good, so a person " +
			"approves it.",
	})
}

func (t *ignoreAccountingInboundChangeTool) Target(
	params map[string]any,
) (agentports.ToolTarget, bool) {
	return targetOf(params, paramInboundChangeID, permission.ResourceAccountingSync)
}

func (t *ignoreAccountingInboundChangeTool) request(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*accountingsyncports.DecideAccountingInboundChangeRequest, *accountingsync.AccountingInboundChange, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, nil, err
	}
	req, err := inboundRequest(params)
	if err != nil {
		return nil, nil, err
	}
	raw, err := requireString(params.Params, paramInboundNote)
	if err != nil {
		return nil, nil, errortypes.NewValidationError(
			paramInboundNote,
			errortypes.ErrRequired,
			"Say why this payment stays out of Trenova",
		)
	}
	note := stringutils.TruncateRunes(
		stringutils.OneLine(raw, maxInboundNoteRunes),
		maxInboundNoteRunes,
	)
	change, err := t.inbound.Get(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	if !change.Status.IsOpen() {
		return nil, nil, errortypes.NewBusinessError(
			"{0} is {1} and cannot be ignored",
			inboundLabel(change),
			string(change.Status),
		)
	}
	return &accountingsyncports.DecideAccountingInboundChangeRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
		Note:       note,
	}, change, nil
}

func (t *ignoreAccountingInboundChangeTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.request(ctx, &params)
	return err
}

func (t *ignoreAccountingInboundChangeTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.inbound.Ignore(ctx, req, params.Actor)
	return err
}
