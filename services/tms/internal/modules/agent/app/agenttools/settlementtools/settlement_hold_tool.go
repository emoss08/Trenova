package settlementtools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/settlementpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/driversettlementservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/pkg/toolschema"
)

const HoldDriverSettlementToolName = "settlement_hold_driver"

const paramRelease = "release"

const paramHoldNote = "note"

const fieldHoldReason = "holdReason"

const fieldHoldNote = "holdNote"

const fieldHeldAt = "heldAt"

const fieldHoldReleasedAt = "holdReleasedAt"

var driverHoldReasons = agenttoolschema.Source(
	"driverSettlement.holdReason",
	driversettlement.HoldReasons(),
)

var holdReasonWords = map[driversettlement.HoldReason]string{
	driversettlement.HoldReasonRateDispute:      "a rate dispute",
	driversettlement.HoldReasonMissingPaperwork: "missing paperwork",
	driversettlement.HoldReasonNegativeNet:      "a negative net",
	driversettlement.HoldReasonDeductionReview:  "a deduction under review",
	driversettlement.HoldReasonOther:            "another reason",
}

type settlementHolder interface {
	PlanHoldChange(
		ctx context.Context,
		req *driversettlementservice.HoldRequest,
	) (*driversettlementservice.ActionPlan, error)
	ChangeHold(
		ctx context.Context,
		req *driversettlementservice.HoldRequest,
		actor *permissionports.RequestActor,
	) (*driversettlement.Settlement, error)
}

type SettlementHoldTool struct {
	settlements settlementHolder
}

func ProvideHoldDriverSettlementTool(
	s *driversettlementservice.Service,
) agenttoolports.AgentTool {
	return &SettlementHoldTool{settlements: s}
}

func (t *SettlementHoldTool) Name() string { return HoldDriverSettlementToolName }

func (t *SettlementHoldTool) Description() string {
	return "Propose holding one driver settlement out of the pay run, with the reason and a " +
		"note saying what it waits on, or releasing a held one (release: true). A held " +
		"settlement keeps its status, and approving, posting and paying it are refused until " +
		"it is released; a posted settlement cannot be held. A person always decides. Read it " +
		"with settlement_get_driver first."
}

func (t *SettlementHoldTool) SearchTerms() []string {
	return []string{"hold settlement", "keep out of pay run", "release hold", "unhold"}
}

func (t *SettlementHoldTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			settlementpreview.ParamSettlementID: agenttoolschema.KindID(
				"The driver settlement, from "+settlementpreview.DriverSettlementReads+
					", settlement_list_batch_blockers or this run's subject. Never guess one.",
				recordkind.Kind(permission.ResourceDriverSettlement),
			),
			paramRelease: toolparams.BooleanProperty(
				"True to release a held settlement back into the pay run; leave it out to hold.",
			),
			toolparams.ParamReason: agenttoolschema.Enum(
				"Why it is held; required to hold, left out to release.",
				driverHoldReasons,
			),
			paramHoldNote: toolparams.StringProperty(
				"What the hold waits on, in a sentence payroll can act on; required when the "+
					"reason is Other.",
				driversettlement.MaxHoldNoteLength,
			),
		},
		toolschema.KeyRequired:             []string{settlementpreview.ParamSettlementID},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *SettlementHoldTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceDriverSettlement,
		Operation:     permission.OpHold,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Keeps a driver's settlement out of the pay run, or lets it back in, inside " +
			"Trenova; whether a driver's pay waits is a person's decision.",
	}
}

func (t *SettlementHoldTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(
		params,
		settlementpreview.ParamSettlementID,
		permission.ResourceDriverSettlement,
	)
}

func (t *SettlementHoldTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*driversettlementservice.HoldRequest, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}
	settlementID, err := toolparams.RequirePulid(params.Params, settlementpreview.ParamSettlementID)
	if err != nil {
		return nil, err
	}
	req := &driversettlementservice.HoldRequest{
		TenantInfo:   toolparams.TenantFrom(params),
		SettlementID: settlementID,
		Release:      toolparams.OptionalBool(params.Params, paramRelease),
	}
	if req.Release {
		return req, nil
	}

	reason := toolparams.OptionalString(params.Params, toolparams.ParamReason)
	if reason == "" {
		return nil, fmt.Errorf("parameter %q is required to hold a settlement; "+
			"set %q to true to release one", toolparams.ParamReason, paramRelease)
	}
	req.Reason = driversettlement.HoldReason(reason)
	if !req.Reason.IsValid() {
		return nil, toolparams.ErrUnknownValue(toolparams.ParamReason, reason,
			driverHoldReasons.Names())
	}
	if req.Note, err = toolparams.BoundedString(
		params.Params,
		paramHoldNote,
		driversettlement.MaxHoldNoteLength,
		false,
	); err != nil {
		return nil, err
	}

	return req, nil
}

func (t *SettlementHoldTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*driversettlementservice.ActionPlan, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}

	return t.settlements.PlanHoldChange(ctx, req)
}

func (t *SettlementHoldTool) Validate(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	plan, err := t.plan(ctx, params)
	if err != nil {
		return err
	}

	return plan.Refusal
}

func (t *SettlementHoldTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return settlementpreview.ErrSettlementNeedsAPerson
	}
	req, err := t.request(params)
	if err != nil {
		return err
	}
	_, err = t.settlements.ChangeHold(ctx, req, params.Actor)

	return err
}

func (t *SettlementHoldTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, err
	}
	plan, err := t.settlements.PlanHoldChange(ctx, req)
	if err != nil {
		return nil, err
	}

	before := settlementpreview.DriverSettlementFacts(plan.Before)
	summary := fmt.Sprintf(
		"Would hold %s %s%s (now %s) out of the pay run for %s.",
		settlementpreview.NounDriverSettlement,
		before.Number,
		settlementpreview.PayeeClause(before.Payee),
		before.Status,
		holdReasonWords[req.Reason],
	)
	fields := []string{fieldHoldReason, fieldHoldNote, fieldHeldAt}
	volatile := []string{fieldHeldAt}
	if req.Release {
		summary = fmt.Sprintf(
			"Would release the hold on %s %s%s (now %s), so it can be approved, posted and "+
				"paid again.",
			settlementpreview.NounDriverSettlement,
			before.Number,
			settlementpreview.PayeeClause(before.Payee),
			before.Status,
		)
		fields = append(fields, fieldHoldReleasedAt)
		volatile = append(volatile, fieldHoldReleasedAt)
	}
	if plan.Refused() {
		return changepreview.WouldFail(toolpreview.Build(summary), plan.Refusal)
	}

	change, err := toolpreview.Changed(toolpreview.Record{
		Resource: permission.ResourceDriverSettlement,
		ID:       before.ID,
		Label:    before.Number,
		Version:  changepreview.PinnedVersion(before.Version),
	}, plan.Before, plan.After, toolpreview.Only(fields...), toolpreview.Volatile(volatile...))
	if err != nil {
		return nil, err
	}

	return toolpreview.Build(summary, change), nil
}
