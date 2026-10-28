package accountingtools

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const paramInboundChangeID = "inboundChangeId"

const paramInboundNote = "note"

const toolListInbound = "accounting_list_inbound_changes"

const fieldInboundStatus = "status"

const maxInboundNoteRunes = 500

const inboundChangeParamHelp = "The payment's id, from accounting_list_inbound_changes."

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
	params *agenttoolports.ToolExecuteParams,
) (*accountingsyncports.GetAccountingInboundChangeRequest, error) {
	id, err := common.RequirePulid(params.Params, paramInboundChangeID)
	if err != nil {
		return nil, err
	}
	return &accountingsyncports.GetAccountingInboundChangeRequest{
		TenantInfo: common.TenantFrom(*params),
		ID:         id,
	}, nil
}

type ApplyAccountingInboundChangeTool struct {
	inbound accountingInboundOperator
}

func newApplyAccountingInboundChangeTool(
	inbound accountingInboundOperator,
) agenttoolports.AgentTool {
	return &ApplyAccountingInboundChangeTool{inbound: inbound}
}

func ProvideApplyAccountingInboundChangeTool(
	inbound accountingsyncports.AccountingInboundService,
) agenttoolports.AgentTool {
	return newApplyAccountingInboundChangeTool(inbound)
}

func (t *ApplyAccountingInboundChangeTool) Name() string {
	return "accounting_apply_inbound_change"
}

func (t *ApplyAccountingInboundChangeTool) Recipe() []string {
	return []string{"accounting_list_inbound_changes", "accounting_apply_inbound_change"}
}

func (t *ApplyAccountingInboundChangeTool) Description() string {
	return "Accept an inbound change: one payment the accounting system recorded against " +
		"documents Trenova sent, shown as Proposed by accounting_list_inbound_changes. " +
		"Accepting brings it into Trenova on the day it was made, with any credits it used, " +
		"or marks the settlement it pays as paid, and only while it still matches what is " +
		"open. The preview shows exactly what would post."
}

func (t *ApplyAccountingInboundChangeTool) SearchTerms() []string {
	return []string{"accept inbound change", "customer payment from accounting"}
}

func (t *ApplyAccountingInboundChangeTool) Prerequisites() []string {
	return []string{toolListInbound}
}

func (t *ApplyAccountingInboundChangeTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramInboundChangeID: agenttoolschema.KindID(
			inboundChangeParamHelp,
			permission.KindAccountingInboundChange,
		),
	}, paramInboundChangeID)
}

func (t *ApplyAccountingInboundChangeTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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
func (t *ApplyAccountingInboundChangeTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, paramInboundChangeID, permission.ResourceAccountingSync)
}

func (t *ApplyAccountingInboundChangeTool) PlanPreview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingsyncports.AccountingInboundApplyPreview, error) {
	if err := common.GuardExecute(t, *params); err != nil {
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

func (t *ApplyAccountingInboundChangeTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.PlanPreview(ctx, &params)
	return err
}

func (t *ApplyAccountingInboundChangeTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if _, err := t.PlanPreview(ctx, &params); err != nil {
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

type IgnoreAccountingInboundChangeTool struct {
	inbound accountingInboundOperator
}

func newIgnoreAccountingInboundChangeTool(
	inbound accountingInboundOperator,
) agenttoolports.AgentTool {
	return &IgnoreAccountingInboundChangeTool{inbound: inbound}
}

func ProvideIgnoreAccountingInboundChangeTool(
	inbound accountingsyncports.AccountingInboundService,
) agenttoolports.AgentTool {
	return newIgnoreAccountingInboundChangeTool(inbound)
}

func (t *IgnoreAccountingInboundChangeTool) Name() string {
	return "accounting_ignore_inbound_change"
}

func (t *IgnoreAccountingInboundChangeTool) Description() string {
	return "Dismiss an inbound change, with the reason, so the accounting system's payment " +
		"is never brought into Trenova. Use it when a person says the payment was already " +
		"keyed in Trenova, or belongs to something Trenova does not track."
}

func (t *IgnoreAccountingInboundChangeTool) SearchTerms() []string {
	return []string{"dismiss inbound change", "already keyed"}
}

func (t *IgnoreAccountingInboundChangeTool) Prerequisites() []string {
	return []string{toolListInbound}
}

func (t *IgnoreAccountingInboundChangeTool) ParamSchema() map[string]any {
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

func (t *IgnoreAccountingInboundChangeTool) Policy() agenttoolports.ToolPolicy {
	return accountingSyncPolicy(t.Name(), accountingSyncPolicySpec{
		resource:  permission.ResourceAccountingSync,
		operation: permission.OpUpdate,
		tier:      agent.TierActWithApproval,
		rationale: "Keeps a payment the books hold out of Trenova for good, so a person " +
			"approves it.",
	})
}

func (t *IgnoreAccountingInboundChangeTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return common.TargetOf(params, paramInboundChangeID, permission.ResourceAccountingSync)
}

func (t *IgnoreAccountingInboundChangeTool) request(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*accountingsyncports.DecideAccountingInboundChangeRequest, *accountingsync.AccountingInboundChange, error) {
	if err := common.GuardExecute(t, *params); err != nil {
		return nil, nil, err
	}
	req, err := inboundRequest(params)
	if err != nil {
		return nil, nil, err
	}
	raw, err := common.RequireString(params.Params, paramInboundNote)
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

func (t *IgnoreAccountingInboundChangeTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.request(ctx, &params)
	return err
}

func (t *IgnoreAccountingInboundChangeTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.inbound.Ignore(ctx, req, params.Actor)
	return err
}
