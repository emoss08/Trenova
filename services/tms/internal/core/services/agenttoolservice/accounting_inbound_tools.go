package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
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
		req *serviceports.GetAccountingInboundChangeRequest,
	) (*accountingsync.AccountingInboundChange, error)
	PreviewApply(
		ctx context.Context,
		req *serviceports.GetAccountingInboundChangeRequest,
	) (*serviceports.AccountingInboundApplyPreview, error)
	Apply(
		ctx context.Context,
		req *serviceports.DecideAccountingInboundChangeRequest,
		actor *serviceports.RequestActor,
	) (*accountingsync.AccountingInboundChange, error)
	Ignore(
		ctx context.Context,
		req *serviceports.DecideAccountingInboundChangeRequest,
		actor *serviceports.RequestActor,
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
	params *serviceports.ToolExecuteParams,
) (*serviceports.GetAccountingInboundChangeRequest, error) {
	id, err := requirePulid(params.Params, paramInboundChangeID)
	if err != nil {
		return nil, err
	}
	return &serviceports.GetAccountingInboundChangeRequest{
		TenantInfo: tenantFrom(*params),
		ID:         id,
	}, nil
}

type applyAccountingInboundChangeTool struct {
	inbound accountingInboundOperator
}

func newApplyAccountingInboundChangeTool(inbound accountingInboundOperator) serviceports.AgentTool {
	return &applyAccountingInboundChangeTool{inbound: inbound}
}

func provideApplyAccountingInboundChangeTool(
	inbound serviceports.AccountingInboundService,
) serviceports.AgentTool {
	return newApplyAccountingInboundChangeTool(inbound)
}

func (t *applyAccountingInboundChangeTool) Name() string {
	return "apply_accounting_inbound_change"
}

func (t *applyAccountingInboundChangeTool) Description() string {
	return "Post a payment recorded in the accounting system in Trenova. For a customer " +
		"payment it posts the payment against the invoices it pays and applies any credit " +
		"memos it used; for a bill payment it marks the settlement paid on the payment's " +
		"date. It only applies a Proposed payment that still matches what is open in " +
		"Trenova, and the preview shows exactly what would post."
}

func (t *applyAccountingInboundChangeTool) SearchTerms() []string {
	return []string{"apply payment", "record payment from the books", "mark paid"}
}

func (t *applyAccountingInboundChangeTool) Prerequisites() []string {
	return []string{toolListInbound}
}

func (t *applyAccountingInboundChangeTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramInboundChangeID: jsonschemautils.Text(inboundChangeParamHelp),
	}, paramInboundChangeID)
}

func (t *applyAccountingInboundChangeTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
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
) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramInboundChangeID, permission.ResourceAccountingSync)
}

func (t *applyAccountingInboundChangeTool) preview(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*serviceports.AccountingInboundApplyPreview, error) {
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
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, err := t.preview(ctx, &params)
	return err
}

func (t *applyAccountingInboundChangeTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if _, err := t.preview(ctx, &params); err != nil {
		return err
	}
	req, err := inboundRequest(&params)
	if err != nil {
		return err
	}
	_, err = t.inbound.Apply(ctx, &serviceports.DecideAccountingInboundChangeRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	}, params.Actor)
	return err
}

func (t *applyAccountingInboundChangeTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) (*agent.ToolSimulation, error) {
	preview, err := t.preview(ctx, &params)
	if err != nil {
		return nil, err
	}
	change := preview.Change
	currency := change.CurrencyCode
	changes := make([]agent.FieldChange, 0, len(preview.Lines)+1)
	changes = append(changes, agent.FieldChange{
		Field: fieldInboundStatus,
		From:  string(change.Status),
		To:    string(accountingsync.InboundStatusApplied),
	})
	for _, line := range preview.Lines {
		changes = append(changes, agent.FieldChange{
			Field: line.ObjectNumber,
			From:  "open " + money.FormatMinor(line.OpenMinor, currency),
			To:    "pays " + money.FormatMinor(line.AmountMinor, currency),
		})
	}

	summary := fmt.Sprintf(
		"Would post %s for %s on %s",
		inboundLabel(change),
		money.FormatMinor(change.AmountMinor, currency),
		timeutils.FormatCalendarDate(preview.PaidAt, nil),
	)
	if preview.UnappliedMinor > 0 {
		summary += fmt.Sprintf(", leaving %s as unapplied cash",
			money.FormatMinor(preview.UnappliedMinor, currency))
	}
	return &agent.ToolSimulation{
		Summary:   summary,
		Previewed: true,
		Changes:   changes,
	}, nil
}

type ignoreAccountingInboundChangeTool struct {
	inbound accountingInboundOperator
}

func newIgnoreAccountingInboundChangeTool(
	inbound accountingInboundOperator,
) serviceports.AgentTool {
	return &ignoreAccountingInboundChangeTool{inbound: inbound}
}

func provideIgnoreAccountingInboundChangeTool(
	inbound serviceports.AccountingInboundService,
) serviceports.AgentTool {
	return newIgnoreAccountingInboundChangeTool(inbound)
}

func (t *ignoreAccountingInboundChangeTool) Name() string {
	return "ignore_accounting_inbound_change"
}

func (t *ignoreAccountingInboundChangeTool) Description() string {
	return "Leave a payment recorded in the accounting system out of Trenova, with the " +
		"reason. Use it when a person says the payment was already entered in Trenova, or " +
		"belongs to something Trenova does not track. An ignored payment is never applied " +
		"afterwards."
}

func (t *ignoreAccountingInboundChangeTool) SearchTerms() []string {
	return []string{"ignore payment", "already entered", "duplicate payment"}
}

func (t *ignoreAccountingInboundChangeTool) Prerequisites() []string {
	return []string{toolListInbound}
}

func (t *ignoreAccountingInboundChangeTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramInboundChangeID: jsonschemautils.Text(inboundChangeParamHelp),
		paramInboundNote: jsonschemautils.Text(
			"Why it stays out of Trenova, in one sentence a bookkeeper would accept.",
		),
	}, paramInboundChangeID, paramInboundNote)
}

func (t *ignoreAccountingInboundChangeTool) Policy() serviceports.ToolPolicy {
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
) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramInboundChangeID, permission.ResourceAccountingSync)
}

func (t *ignoreAccountingInboundChangeTool) request(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*serviceports.DecideAccountingInboundChangeRequest, *accountingsync.AccountingInboundChange, error) {
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
	return &serviceports.DecideAccountingInboundChangeRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
		Note:       note,
	}, change, nil
}

func (t *ignoreAccountingInboundChangeTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.request(ctx, &params)
	return err
}

func (t *ignoreAccountingInboundChangeTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.request(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.inbound.Ignore(ctx, req, params.Actor)
	return err
}

func (t *ignoreAccountingInboundChangeTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) (*agent.ToolSimulation, error) {
	req, change, err := t.request(ctx, &params)
	if err != nil {
		return nil, err
	}
	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would leave %s for %s out of Trenova for good: %s",
			inboundLabel(change),
			money.FormatMinor(change.AmountMinor, change.CurrencyCode),
			req.Note,
		),
		Previewed: true,
		Changes: []agent.FieldChange{{
			Field: fieldInboundStatus,
			From:  string(change.Status),
			To:    string(accountingsync.InboundStatusIgnored),
		}},
	}, nil
}
