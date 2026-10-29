package factoringledgertools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/factoringkit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringremittance"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringremittanceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/jsonschemautils"
)

const (
	paramRemittanceID   = "remittanceId"
	paramRemittanceID2  = "id"
	paramLineID         = "lineId"
	paramLineAction     = "action"
	paramLineKind       = "kind"
	toolGetRemittance   = "factoring_get_remittance"
	toolListRemittances = "factoring_list_remittances"
	maxRemittanceReason = factoringremittance.MaxRemittanceReasonLength
)

func remittanceRecord(rem *factoringremittance.Remittance) toolpreview.Record {
	label := "Factor remittance"
	if rem.FileName != "" {
		label += " " + rem.FileName
	}
	return toolpreview.Record{
		Resource: permission.ResourceFactoringRemittance,
		ID:       rem.ID,
		Label:    label,
		Version:  changepreview.PreviewVersion(rem.Version),
	}
}

func remittancePreviewOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(
			"status", "matchedCount", "exceptionCount", "appliedCount", "discardReason",
		),
		toolpreview.Labels(map[string]string{
			"matchedCount":   "Matched lines",
			"exceptionCount": "Lines to resolve",
			"appliedCount":   "Booked lines",
			"discardReason":  "Why it was discarded",
		}),
	}
}

func remittanceLineRecord(
	rem *factoringremittance.Remittance,
	line *factoringremittance.RemittanceLine,
) toolpreview.Record {
	return toolpreview.Record{
		Resource: permission.ResourceFactoringRemittance,
		ID:       rem.ID,
		Label:    "Row " + strconv.Itoa(line.RowNumber) + " of " + remittanceRecord(rem).Label,
		Version:  changepreview.PreviewVersion(rem.Version),
	}
}

func remittanceLinePreviewOptions() []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(
			"kind", "matchStatus", "matchedBy", "submissionItemId", "ignoredReason",
		),
		toolpreview.Labels(map[string]string{
			"kind":             "Records",
			"matchStatus":      "Match",
			"matchedBy":        "Matched by",
			"submissionItemId": "Schedule item",
			"ignoredReason":    "Why it is not booked",
		}),
	}
}

func remittanceIDParam() map[string]any {
	return agenttoolschema.RecordIDText(
		permission.ResourceFactoringRemittance,
		"The remittance, from factoring_list_remittances.",
	)
}

func remittanceVersionParam() map[string]any {
	return jsonschemautils.Integer("The remittance's version, from factoring_get_remittance.")
}

func remittanceVersionRequest(
	params *agenttoolports.ToolExecuteParams,
	key string,
) (*factoringremittanceports.RemittanceVersionRequest, error) {
	id, err := toolparams.RequirePulid(params.Params, key)
	if err != nil {
		return nil, err
	}
	version, err := toolparams.RequireIntInRange(
		params.Params,
		factoringkit.ParamVersion,
		0,
		factoringkit.MaxVersion,
	)
	if err != nil {
		return nil, err
	}
	return &factoringremittanceports.RemittanceVersionRequest{
		TenantInfo: toolparams.TenantFrom(params),
		ID:         id,
		Version:    int64(version),
	}, nil
}

type ResolveFactoringRemittanceLineTool struct {
	remittances factoringremittanceports.RemittanceService
}

func NewResolveFactoringRemittanceLineTool(
	remittances factoringremittanceports.RemittanceService,
) agenttoolports.AgentTool {
	return &ResolveFactoringRemittanceLineTool{remittances: remittances}
}

func (t *ResolveFactoringRemittanceLineTool) Name() string {
	return "factoring_resolve_remittance_line"
}

func (t *ResolveFactoringRemittanceLineTool) Description() string {
	return "Resolve one line of a factor remittance so it can be applied, or set it aside. " +
		"Match it to a schedule item (a suggestion's itemId from factoring_get_remittance), " +
		"Unmatch it, Ignore it with why (a fee or adjustment booked by hand), Restore an " +
		"ignored line, or SetKind when the factor's report says what the line records " +
		"differently. Nothing is booked until the remittance is applied."
}

func (t *ResolveFactoringRemittanceLineTool) SearchTerms() []string {
	return []string{"remittance exception", "match remittance line", "ignore remittance line"}
}

func (t *ResolveFactoringRemittanceLineTool) Prerequisites() []string {
	return []string{toolGetRemittance}
}

func (t *ResolveFactoringRemittanceLineTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramRemittanceID:         remittanceIDParam(),
		factoringkit.ParamVersion: remittanceVersionParam(),
		paramLineID: jsonschemautils.Text(
			"The line's lineId, from factoring_get_remittance.",
		),
		paramLineAction: agenttoolschema.Enum(
			"What to do with the line.",
			agenttoolschema.FactoringRemittanceLineActions,
		),
		paramItemID: itemIDParam(
			"The schedule item the line pays, for Match: a suggestion's itemId from " +
				"factoring_get_remittance, or an item from factoring_get_submission.",
		),
		paramLineKind: agenttoolschema.Enum(
			"What the line records, for SetKind.",
			agenttoolschema.FactoringRemittanceLineKinds,
		),
		factoringkit.ParamReason: toolparams.StringProperty(
			"Why the line is not booked, for Ignore.",
			maxRemittanceReason,
		),
	}, paramRemittanceID, factoringkit.ParamVersion, paramLineID, paramLineAction)
}

func (t *ResolveFactoringRemittanceLineTool) Policy() agenttoolports.ToolPolicy {
	return factoringkit.Policy(t.Name(), &factoringkit.PolicySpec{
		Resource:    permission.ResourceFactoringRemittance,
		Operation:   permission.OpUpdate,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Rationale: "Changes which schedule item a remittance line pays before anything is " +
			"booked; it can be undone until the remittance is applied.",
	})
}

func (t *ResolveFactoringRemittanceLineTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramRemittanceID, permission.ResourceFactoringRemittance)
}

func (t *ResolveFactoringRemittanceLineTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringremittanceports.UpdateRemittanceLineRequest, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}
	base, err := remittanceVersionRequest(params, paramRemittanceID)
	if err != nil {
		return nil, err
	}
	lineID, err := toolparams.RequirePulid(params.Params, paramLineID)
	if err != nil {
		return nil, err
	}
	action := factoringremittanceports.RemittanceLineAction(
		strings.TrimSpace(toolparams.OptionalString(params.Params, paramLineAction)),
	)
	if !action.IsValid() {
		return nil, fmt.Errorf(
			"parameter %q must be Match, Unmatch, Ignore, Restore or SetKind",
			paramLineAction,
		)
	}
	req := &factoringremittanceports.UpdateRemittanceLineRequest{
		TenantInfo:   base.TenantInfo,
		RemittanceID: base.ID,
		Version:      base.Version,
		LineID:       lineID,
		Action:       action,
		Kind: factoringremittance.RemittanceLineKind(
			strings.TrimSpace(toolparams.OptionalString(params.Params, paramLineKind)),
		),
	}
	if action == factoringremittanceports.RemittanceLineActionMatch {
		if req.ItemID, err = toolparams.RequirePulid(params.Params, paramItemID); err != nil {
			return nil, err
		}
	}
	reason, err := toolparams.OptionalBoundedText(
		params.Params,
		factoringkit.ParamReason,
		maxRemittanceReason,
	)
	if err != nil {
		return nil, err
	}
	if reason != nil {
		req.Reason = *reason
	}
	return req, nil
}

func (t *ResolveFactoringRemittanceLineTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.remittances.PlanUpdateLine(ctx, req, params.Actor)
	return err
}

func (t *ResolveFactoringRemittanceLineTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.remittances.UpdateLine(ctx, req, params.Actor)
	return err
}

func (t *ResolveFactoringRemittanceLineTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const summary = "Would change how a remittance line is matched."
	req, err := t.request(&params)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}
	plan, err := t.remittances.PlanUpdateLine(ctx, req, params.Actor)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}
	lineChange, err := toolpreview.Changed(
		remittanceLineRecord(plan.Before, plan.LineBefore),
		plan.LineBefore,
		plan.LineAfter,
		remittanceLinePreviewOptions()...,
	)
	if err != nil {
		return nil, err
	}
	remChange, err := toolpreview.Changed(
		remittanceRecord(plan.Before),
		plan.Before,
		plan.After,
		remittancePreviewOptions()...,
	)
	if err != nil {
		return nil, err
	}
	return toolpreview.Build(
		"Would change it: "+plan.Comment+". Nothing is booked until the remittance is applied.",
		lineChange,
		remChange,
	), nil
}

type RematchFactoringRemittanceTool struct {
	remittances factoringremittanceports.RemittanceService
}

func NewRematchFactoringRemittanceTool(
	remittances factoringremittanceports.RemittanceService,
) agenttoolports.AgentTool {
	return &RematchFactoringRemittanceTool{remittances: remittances}
}

func (t *RematchFactoringRemittanceTool) Name() string { return "factoring_rematch_remittance" }

func (t *RematchFactoringRemittanceTool) Description() string {
	return "Match a remittance's open lines again against the factor's schedules as they are " +
		"now, after a schedule was sent or funded since the remittance was read. Lines a " +
		"person matched or ignored stay as they are."
}

func (t *RematchFactoringRemittanceTool) SearchTerms() []string {
	return []string{"rematch remittance", "match remittance again"}
}

func (t *RematchFactoringRemittanceTool) Prerequisites() []string {
	return []string{toolGetRemittance}
}

func (t *RematchFactoringRemittanceTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramRemittanceID2:        remittanceIDParam(),
		factoringkit.ParamVersion: remittanceVersionParam(),
	}, paramRemittanceID2, factoringkit.ParamVersion)
}

func (t *RematchFactoringRemittanceTool) Policy() agenttoolports.ToolPolicy {
	return factoringkit.Policy(t.Name(), &factoringkit.PolicySpec{
		Resource:    permission.ResourceFactoringRemittance,
		Operation:   permission.OpUpdate,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Idempotent:  true,
		Reversible:  true,
		Rationale: "Recomputes suggestions and automatic matches before anything is booked; " +
			"a person's own matches are kept.",
	})
}

func (t *RematchFactoringRemittanceTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramRemittanceID2, permission.ResourceFactoringRemittance)
}

func (t *RematchFactoringRemittanceTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringremittanceports.RemittanceVersionRequest, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}
	return remittanceVersionRequest(params, paramRemittanceID2)
}

func (t *RematchFactoringRemittanceTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	rem, err := t.remittances.Get(ctx, factoringremittanceports.GetRemittanceRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return err
	}
	return rem.RequireEditable()
}

func (t *RematchFactoringRemittanceTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.remittances.Rematch(ctx, req, params.Actor)
	return err
}

func (t *RematchFactoringRemittanceTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const summary = "Would match the remittance's open lines again."
	if err := t.Validate(ctx, params); err != nil {
		return factoringkit.Refused(summary, err)
	}
	return toolpreview.Build(summary + " Lines a person matched or ignored stay as they are."), nil
}

type ApplyFactoringRemittanceTool struct {
	ledgerTool
	remittances factoringremittanceports.RemittanceService
}

func NewApplyFactoringRemittanceTool(
	remittances factoringremittanceports.RemittanceService,
	ledger factoringledgerports.LedgerService,
	accounts glaccountports.GLAccountRepository,
) agenttoolports.AgentTool {
	return &ApplyFactoringRemittanceTool{
		ledgerTool:  ledgerTool{ledger: ledger, accounts: accounts},
		remittances: remittances,
	}
}

func (t *ApplyFactoringRemittanceTool) Name() string { return "factoring_apply_remittance" }

func (t *ApplyFactoringRemittanceTool) Description() string {
	return "Book every matched line of a factor remittance to the ledger in one go. Funded " +
		"and rejected invoices are recorded on their schedules, reserve releases and " +
		"chargebacks are booked with the line's cash, and each entry remembers its line. " +
		"Lines still to resolve stay open. A person approves every booking."
}

func (t *ApplyFactoringRemittanceTool) SearchTerms() []string {
	return []string{"apply remittance", "book remittance", "factor funding report"}
}

func (t *ApplyFactoringRemittanceTool) Prerequisites() []string {
	return []string{toolGetRemittance}
}

func (t *ApplyFactoringRemittanceTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramRemittanceID2:        remittanceIDParam(),
		factoringkit.ParamVersion: remittanceVersionParam(),
	}, paramRemittanceID2, factoringkit.ParamVersion)
}

func (t *ApplyFactoringRemittanceTool) Policy() agenttoolports.ToolPolicy {
	return factoringkit.Policy(t.Name(), &factoringkit.PolicySpec{
		Resource:    permission.ResourceFactoringRemittance,
		Operation:   permission.OpImport,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		Egress:      agent.EgressMoney,
		Idempotent:  true,
		Rationale: "Books the factor's advances, fees, reserve releases and chargebacks to the " +
			"general ledger; only a person's approval runs it.",
	})
}

func (t *ApplyFactoringRemittanceTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramRemittanceID2, permission.ResourceFactoringRemittance)
}

func (t *ApplyFactoringRemittanceTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringremittanceports.RemittanceVersionRequest, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}
	return remittanceVersionRequest(params, paramRemittanceID2)
}

func (t *ApplyFactoringRemittanceTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.remittances.PlanApply(ctx, req, params.Actor)
	return err
}

func (t *ApplyFactoringRemittanceTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	if err := requireApproval(t.Name(), &params); err != nil {
		return err
	}
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.remittances.Apply(ctx, req, params.Actor)
	return err
}

func (t *ApplyFactoringRemittanceTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const summary = "Would book the remittance's matched lines to the ledger."
	req, err := t.request(&params)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}
	plan, err := t.remittances.PlanApply(ctx, req, params.Actor)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}
	movements := make([]*factoringledgerports.PlannedMovement, 0, len(plan.Lines))
	for _, line := range plan.Lines {
		if line.Movement != nil {
			movements = append(movements, line.Movement)
		}
	}
	currency := plan.Remittance.CurrencyCode
	entries, err := t.movementChanges(ctx, req.TenantInfo, currency, movements)
	if err != nil {
		return nil, err
	}
	sentences := []string{fmt.Sprintf(
		"Would apply remittance %s: %s funded (advance %s, fee %s), %s rejected, %s released "+
			"(%s) and %s charged back (%s), booking %s.",
		remittanceRecord(plan.Remittance).Label,
		changepreview.CountOf(plan.FundedCount, "invoice"),
		formatMinor(currency, plan.AdvanceMinor),
		formatMinor(currency, plan.FeeMinor),
		changepreview.CountOf(plan.RejectedCount, "invoice"),
		changepreview.CountOf(plan.ReleasedCount, "reserve"),
		formatMinor(currency, plan.ReleasedMinor),
		changepreview.CountOf(plan.ChargebackCount, "invoice"),
		formatMinor(currency, plan.ChargebackMinor),
		changepreview.CountOf(len(entries), "journal entry"),
	)}
	if plan.LeftOpenCount > 0 {
		sentences = append(sentences, changepreview.CountOf(plan.LeftOpenCount, "line")+
			" still to resolve stay open.")
	}
	sentences = append(sentences, bookingSentences(movements)...)
	return toolpreview.Build(strings.Join(sentences, " "), entries...), nil
}

type DiscardFactoringRemittanceTool struct {
	remittances factoringremittanceports.RemittanceService
}

func NewDiscardFactoringRemittanceTool(
	remittances factoringremittanceports.RemittanceService,
) agenttoolports.AgentTool {
	return &DiscardFactoringRemittanceTool{remittances: remittances}
}

func (t *DiscardFactoringRemittanceTool) Name() string { return "factoring_discard_remittance" }

func (t *DiscardFactoringRemittanceTool) Description() string {
	return "Discard a factor remittance that will not be booked, such as a duplicate of one " +
		"already applied or a file that is not a remittance. Nothing on it can be booked " +
		"afterwards; a partly applied one cannot be discarded. Say why."
}

func (t *DiscardFactoringRemittanceTool) SearchTerms() []string {
	return []string{"discard remittance", "duplicate remittance"}
}

func (t *DiscardFactoringRemittanceTool) Prerequisites() []string {
	return []string{toolListRemittances}
}

func (t *DiscardFactoringRemittanceTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramRemittanceID2:        remittanceIDParam(),
		factoringkit.ParamVersion: remittanceVersionParam(),
		factoringkit.ParamReason: toolparams.StringProperty(
			"Why the remittance is discarded.",
			maxRemittanceReason,
		),
	}, paramRemittanceID2, factoringkit.ParamVersion, factoringkit.ParamReason)
}

func (t *DiscardFactoringRemittanceTool) Policy() agenttoolports.ToolPolicy {
	return factoringkit.Policy(t.Name(), &factoringkit.PolicySpec{
		Resource:    permission.ResourceFactoringRemittance,
		Operation:   permission.OpCancel,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Rationale:   "Sets a remittance aside for good before anything on it is booked.",
	})
}

func (t *DiscardFactoringRemittanceTool) Target(
	params map[string]any,
) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, paramRemittanceID2, permission.ResourceFactoringRemittance)
}

func (t *DiscardFactoringRemittanceTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*factoringremittanceports.DiscardRemittanceRequest, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}
	base, err := remittanceVersionRequest(params, paramRemittanceID2)
	if err != nil {
		return nil, err
	}
	reason, err := toolparams.RequireBoundedText(
		params.Params,
		factoringkit.ParamReason,
		maxRemittanceReason,
	)
	if err != nil {
		return nil, err
	}
	return &factoringremittanceports.DiscardRemittanceRequest{
		TenantInfo: base.TenantInfo,
		ID:         base.ID,
		Version:    base.Version,
		Reason:     reason,
	}, nil
}

func (t *DiscardFactoringRemittanceTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.remittances.PlanDiscard(ctx, req, params.Actor)
	return err
}

func (t *DiscardFactoringRemittanceTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, err := t.request(&params)
	if err != nil {
		return err
	}
	_, err = t.remittances.Discard(ctx, req, params.Actor)
	return err
}

func (t *DiscardFactoringRemittanceTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agentpreview.ToolPreview, error) {
	const summary = "Would discard the remittance."
	req, err := t.request(&params)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}
	plan, err := t.remittances.PlanDiscard(ctx, req, params.Actor)
	if err != nil {
		return factoringkit.Refused(summary, err)
	}
	change, err := toolpreview.Changed(
		remittanceRecord(plan.Before),
		plan.Before,
		plan.After,
		remittancePreviewOptions()...,
	)
	if err != nil {
		return nil, err
	}
	return toolpreview.Build(
		"Would discard "+remittanceRecord(plan.Before).Label+"; nothing on it can be booked "+
			"afterwards.",
		change,
	), nil
}
