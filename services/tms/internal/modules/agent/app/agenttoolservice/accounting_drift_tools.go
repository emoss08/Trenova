package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	paramDriftFindingID = "findingId"
	paramDriftDirection = "direction"
	paramDriftNote      = "note"
	toolListDrift       = "list_accounting_drift_findings"
	maxDriftNoteRunes   = 500
	driftFindingHelp    = "The finding's id, from list_accounting_drift_findings."
)

type accountingDriftOperator interface {
	Get(
		ctx context.Context,
		req *accountingsyncports.GetAccountingDriftFindingRequest,
	) (*accountingsync.AccountingDriftFinding, error)
	PreviewResolve(
		ctx context.Context,
		req *accountingsyncports.ResolveAccountingDriftRequest,
		actor *permissionports.RequestActor,
	) (*accountingsyncports.AccountingDriftFixPreview, error)
	Resolve(
		ctx context.Context,
		req *accountingsyncports.ResolveAccountingDriftRequest,
		actor *permissionports.RequestActor,
	) (*accountingsync.AccountingDriftFinding, error)
	PreviewDismiss(
		ctx context.Context,
		req *accountingsyncports.DismissAccountingDriftRequest,
		actor *permissionports.RequestActor,
	) (*accountingsyncports.AccountingDriftFixPreview, error)
	Dismiss(
		ctx context.Context,
		req *accountingsyncports.DismissAccountingDriftRequest,
		actor *permissionports.RequestActor,
	) (*accountingsync.AccountingDriftFinding, error)
	Overview(
		ctx context.Context,
		req *accountingsyncports.AccountingDriftOverviewRequest,
	) (*accountingsyncports.AccountingDriftOverview, error)
	CheckNow(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		integrationType integration.Type,
	) (*accountingsyncports.AccountingDriftOverview, error)
}

func driftLabel(finding *accountingsync.AccountingDriftFinding) string {
	label := finding.ObjectNumber
	if label == "" {
		label = finding.PartyName
	}
	return string(finding.ObjectType) + " " + label
}

func driftFindingID(params *agentports.ToolExecuteParams) (pulid.ID, error) {
	return requirePulid(params.Params, paramDriftFindingID)
}

type resolveAccountingDriftTool struct {
	drift accountingDriftOperator
}

func newResolveAccountingDriftTool(drift accountingDriftOperator) agentports.AgentTool {
	return &resolveAccountingDriftTool{drift: drift}
}

func provideResolveAccountingDriftTool(
	drift accountingsyncports.AccountingDriftService,
) agentports.AgentTool {
	return newResolveAccountingDriftTool(drift)
}

func (t *resolveAccountingDriftTool) Name() string {
	return "resolve_accounting_drift"
}

func (t *resolveAccountingDriftTool) Recipe() []string {
	return []string{"list_accounting_drift_findings", "resolve_accounting_drift"}
}

func (t *resolveAccountingDriftTool) Description() string {
	return "Fix an open drift finding in one direction. PushTrenovaValue sends Trenova's " +
		"document to the accounting system again: an update, a void, or a new copy of a " +
		"document deleted there. AdjustTrenova changes Trenova to match the accounting system: " +
		"a credit or debit memo against the invoice, a void of an invoice gone from it, or the " +
		"reversal of a payment voided there, none of which is sent back. Only the directions " +
		"the finding lists under fixes are offered. The preview says exactly what happens."
}

func (t *resolveAccountingDriftTool) SearchTerms() []string {
	return []string{"fix drift", "send trenova's total again", "match the accounting system"}
}

func (t *resolveAccountingDriftTool) Prerequisites() []string {
	return []string{toolListDrift}
}

func (t *resolveAccountingDriftTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramDriftFindingID: agenttoolschema.KindID(
			driftFindingHelp,
			permission.KindAccountingDriftFinding,
		),
		paramDriftDirection: agenttoolschema.Enum(
			"Which side to change: PushTrenovaValue changes the accounting system, AdjustTrenova changes Trenova.",
			agenttoolschema.DriftDirections,
		),
	}, paramDriftFindingID, paramDriftDirection)
}

func (t *resolveAccountingDriftTool) Policy() agentports.ToolPolicy {
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
		Rationale: "Changes money on one side of the books: it posts a memo, void or " +
			"reversal in Trenova, or sends a document to the accounting system, so a person " +
			"approves each fix.",
	}
}

func (t *resolveAccountingDriftTool) Target(
	params map[string]any,
) (agentports.ToolTarget, bool) {
	return targetOf(params, paramDriftFindingID, permission.ResourceAccountingSync)
}

func (t *resolveAccountingDriftTool) request(
	params *agentports.ToolExecuteParams,
) (*accountingsyncports.ResolveAccountingDriftRequest, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, err
	}
	id, err := driftFindingID(params)
	if err != nil {
		return nil, err
	}
	raw, err := requireString(params.Params, paramDriftDirection)
	if err != nil {
		return nil, err
	}
	direction := accountingsync.DriftDirection(raw)
	if !direction.IsValid() {
		return nil, errortypes.NewValidationError(
			paramDriftDirection,
			errortypes.ErrInvalid,
			"Direction must be PushTrenovaValue or AdjustTrenova",
		)
	}
	return &accountingsyncports.ResolveAccountingDriftRequest{
		TenantInfo: tenantFrom(*params),
		ID:         id,
		Direction:  direction,
	}, nil
}

func (t *resolveAccountingDriftTool) plan(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*accountingsyncports.ResolveAccountingDriftRequest, *accountingsyncports.AccountingDriftFixPreview, error) {
	req, err := t.request(params)
	if err != nil {
		return nil, nil, err
	}
	preview, err := t.drift.PreviewResolve(ctx, req, params.Actor)
	if err != nil {
		return nil, nil, err
	}
	return req, preview, nil
}

func (t *resolveAccountingDriftTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.plan(ctx, &params)
	return err
}

func (t *resolveAccountingDriftTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.drift.Resolve(ctx, req, params.Actor)
	return err
}

type dismissAccountingDriftTool struct {
	drift accountingDriftOperator
}

func newDismissAccountingDriftTool(drift accountingDriftOperator) agentports.AgentTool {
	return &dismissAccountingDriftTool{drift: drift}
}

func provideDismissAccountingDriftTool(
	drift accountingsyncports.AccountingDriftService,
) agentports.AgentTool {
	return newDismissAccountingDriftTool(drift)
}

func (t *dismissAccountingDriftTool) Name() string {
	return "dismiss_accounting_drift"
}

func (t *dismissAccountingDriftTool) Description() string {
	return "Dismiss an open drift finding, with the reason, keeping both sides as they are. " +
		"An agent may dismiss only an amount difference within the reconciliation tolerance, " +
		"which list_accounting_drift_findings marks as withinTolerance. Anything larger, and " +
		"any deleted, voided or status difference, is a person's decision."
}

func (t *dismissAccountingDriftTool) SearchTerms() []string {
	return []string{"dismiss drift", "rounding difference"}
}

func (t *dismissAccountingDriftTool) Prerequisites() []string {
	return []string{toolListDrift}
}

func (t *dismissAccountingDriftTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramDriftFindingID: agenttoolschema.KindID(
			driftFindingHelp,
			permission.KindAccountingDriftFinding,
		),
		paramDriftNote: jsonschemautils.Text(
			"Why both sides stay as they are, in one sentence a bookkeeper would accept.",
		),
	}, paramDriftFindingID, paramDriftNote)
}

func (t *dismissAccountingDriftTool) Policy() agentports.ToolPolicy {
	return accountingSyncPolicy(t.Name(), accountingSyncPolicySpec{
		resource:  permission.ResourceAccountingSync,
		operation: permission.OpUpdate,
		tier:      agent.TierActWithApproval,
		rationale: "Closes a difference with the books without fixing it, so a person " +
			"approves it.",
	})
}

func (t *dismissAccountingDriftTool) Target(
	params map[string]any,
) (agentports.ToolTarget, bool) {
	return targetOf(params, paramDriftFindingID, permission.ResourceAccountingSync)
}

func (t *dismissAccountingDriftTool) plan(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (*accountingsyncports.DismissAccountingDriftRequest, *accountingsyncports.AccountingDriftFixPreview, error) {
	if err := guardExecute(t, *params); err != nil {
		return nil, nil, err
	}
	id, err := driftFindingID(params)
	if err != nil {
		return nil, nil, err
	}
	raw, err := requireString(params.Params, paramDriftNote)
	if err != nil {
		return nil, nil, errortypes.NewValidationError(
			paramDriftNote,
			errortypes.ErrRequired,
			"Say why both sides stay as they are",
		)
	}
	req := &accountingsyncports.DismissAccountingDriftRequest{
		TenantInfo: tenantFrom(*params),
		ID:         id,
		Note: stringutils.TruncateRunes(
			stringutils.OneLine(raw, maxDriftNoteRunes),
			maxDriftNoteRunes,
		),
	}
	preview, err := t.drift.PreviewDismiss(ctx, req, params.Actor)
	if err != nil {
		return nil, nil, err
	}
	return req, preview, nil
}

func (t *dismissAccountingDriftTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.plan(ctx, &params)
	return err
}

func (t *dismissAccountingDriftTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	req, _, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.drift.Dismiss(ctx, req, params.Actor)
	return err
}

type checkAccountingDriftTool struct {
	drift accountingDriftOperator
}

func newCheckAccountingDriftTool(drift accountingDriftOperator) agentports.AgentTool {
	return &checkAccountingDriftTool{drift: drift}
}

func provideCheckAccountingDriftTool(
	drift accountingsyncports.AccountingDriftService,
) agentports.AgentTool {
	return newCheckAccountingDriftTool(drift)
}

func (t *checkAccountingDriftTool) Name() string {
	return "check_accounting_drift"
}

func (t *checkAccountingDriftTool) Description() string {
	return "Compare the accounting system with Trenova immediately instead of waiting for the " +
		"nightly check. It runs in the background and changes nothing on either side; read " +
		"list_accounting_drift_findings once it has finished."
}

func (t *checkAccountingDriftTool) SearchTerms() []string {
	return []string{"check drift", "reconcile with the accounting system"}
}

func (t *checkAccountingDriftTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(
		map[string]any{paramAccountingSystem: accountingSystemSchema()},
		paramAccountingSystem,
	)
}

func (t *checkAccountingDriftTool) Policy() agentports.ToolPolicy {
	return accountingSyncPolicy(t.Name(), accountingSyncPolicySpec{
		resource:   permission.ResourceAccountingSync,
		operation:  permission.OpUpdate,
		tier:       agent.TierActWithApproval,
		idempotent: true,
		rationale: "Reads every synced document back from the accounting system, which " +
			"spends the provider's call allowance, so a person approves it.",
	})
}

func (t *checkAccountingDriftTool) setup(
	ctx context.Context,
	params *agentports.ToolExecuteParams,
) (integration.Type, *accountingsyncports.AccountingDriftOverview, error) {
	if err := guardExecute(t, *params); err != nil {
		return "", nil, err
	}
	system, err := accountingSystemFrom(params.Params)
	if err != nil {
		return "", nil, err
	}
	overview, err := t.drift.Overview(ctx, &accountingsyncports.AccountingDriftOverviewRequest{
		TenantInfo:      tenantFrom(*params),
		IntegrationType: system,
	})
	if err != nil {
		return "", nil, err
	}
	return system, overview, nil
}

func (t *checkAccountingDriftTool) Validate(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	_, _, err := t.setup(ctx, &params)
	return err
}

func (t *checkAccountingDriftTool) Execute(
	ctx context.Context,
	params agentports.ToolExecuteParams, //nolint:gocritic // the AgentTool interface passes params by value
) error {
	system, _, err := t.setup(ctx, &params)
	if err != nil {
		return err
	}
	_, err = t.drift.CheckNow(ctx, tenantFrom(params), system)
	return err
}
