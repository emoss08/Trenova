package accountingdrifttools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDriftOperator struct {
	findings   map[pulid.ID]*accountingdrift.AccountingDriftFinding
	refuse     error
	resolved   *accountingdriftports.ResolveAccountingDriftRequest
	dismissed  *accountingdriftports.DismissAccountingDriftRequest
	checked    integration.Type
	lastActor  *permissionports.RequestActor
	overviewID pulid.ID
}

func (f *fakeDriftOperator) finding(id pulid.ID) (*accountingdrift.AccountingDriftFinding, error) {
	finding, ok := f.findings[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("Drift finding not found")
	}
	out := *finding
	return &out, nil
}

func (f *fakeDriftOperator) Get(
	_ context.Context,
	req *accountingdriftports.GetAccountingDriftFindingRequest,
) (*accountingdrift.AccountingDriftFinding, error) {
	return f.finding(req.ID)
}

func (f *fakeDriftOperator) PreviewResolve(
	_ context.Context,
	req *accountingdriftports.ResolveAccountingDriftRequest,
	actor *permissionports.RequestActor,
) (*accountingdriftports.AccountingDriftFixPreview, error) {
	f.lastActor = actor
	if f.refuse != nil {
		return nil, f.refuse
	}
	finding, err := f.finding(req.ID)
	if err != nil {
		return nil, err
	}
	return &accountingdriftports.AccountingDriftFixPreview{
		Finding:   finding,
		Direction: req.Direction,
		FixObject: accountingdrift.DriftFixCreditMemo,
		Summary:   "Post a credit memo of 50.00 USD against INV-1001.",
	}, nil
}

func (f *fakeDriftOperator) Resolve(
	_ context.Context,
	req *accountingdriftports.ResolveAccountingDriftRequest,
	actor *permissionports.RequestActor,
) (*accountingdrift.AccountingDriftFinding, error) {
	f.resolved = req
	f.lastActor = actor
	return f.finding(req.ID)
}

func (f *fakeDriftOperator) PreviewDismiss(
	_ context.Context,
	req *accountingdriftports.DismissAccountingDriftRequest,
	actor *permissionports.RequestActor,
) (*accountingdriftports.AccountingDriftFixPreview, error) {
	f.lastActor = actor
	if f.refuse != nil {
		return nil, f.refuse
	}
	finding, err := f.finding(req.ID)
	if err != nil {
		return nil, err
	}
	return &accountingdriftports.AccountingDriftFixPreview{
		Finding: finding,
		Summary: "Dismiss the difference in the total of INV-1001 and keep both sides as they are.",
	}, nil
}

func (f *fakeDriftOperator) Dismiss(
	_ context.Context,
	req *accountingdriftports.DismissAccountingDriftRequest,
	actor *permissionports.RequestActor,
) (*accountingdrift.AccountingDriftFinding, error) {
	f.dismissed = req
	f.lastActor = actor
	return f.finding(req.ID)
}

func (f *fakeDriftOperator) Overview(
	_ context.Context,
	_ *accountingdriftports.AccountingDriftOverviewRequest,
) (*accountingdriftports.AccountingDriftOverview, error) {
	return &accountingdriftports.AccountingDriftOverview{
		ConnectionID: f.overviewID,
		ProviderName: "QuickBooks Online",
	}, nil
}

func (f *fakeDriftOperator) CheckNow(
	_ context.Context,
	_ pagination.TenantInfo,
	integrationType integration.Type,
) (*accountingdriftports.AccountingDriftOverview, error) {
	f.checked = integrationType
	return &accountingdriftports.AccountingDriftOverview{ConnectionID: f.overviewID}, nil
}

func openDriftFinding() *accountingdrift.AccountingDriftFinding {
	trenova, provider := int64(125_000), int64(120_000)
	return accountingdrift.NewAccountingDriftFinding(&accountingdrift.DriftObservation{
		ConnectionID:  pulid.MustNew("acctc_"),
		ObjectType:    accountingsync.SyncObjectInvoice,
		ObjectID:      pulid.MustNew("inv_"),
		ObjectNumber:  "INV-1001",
		Kind:          accountingdrift.DriftAmountMismatch,
		CurrencyCode:  "USD",
		TrenovaMinor:  &trenova,
		ProviderMinor: &provider,
		At:            1_790_000_000,
	})
}

func newDriftOperator(finding *accountingdrift.AccountingDriftFinding) *fakeDriftOperator {
	return &fakeDriftOperator{
		findings:   map[pulid.ID]*accountingdrift.AccountingDriftFinding{finding.ID: finding},
		overviewID: finding.ConnectionID,
	}
}

func TestResolveAccountingDrift_PreviewsThenFixesAsTheApprover(t *testing.T) {
	t.Parallel()

	finding := openDriftFinding()
	operator := newDriftOperator(finding)
	tool := newResolveAccountingDriftTool(operator)
	params := agenttoolstest.AccountingSyncParams(map[string]any{
		"findingId": finding.ID.String(),
		"direction": "AdjustTrenova",
	})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Would post a credit memo of 50.00 USD")
	require.Len(t, preview.Changes, 1)
	assert.Equal(t, finding.ID, preview.Changes[0].EntityID)
	assert.Nil(t, operator.resolved, "previewing fixes nothing")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.resolved)
	assert.Equal(t, accountingdrift.DriftAdjustTrenova, operator.resolved.Direction)
	assert.Equal(t, params.Actor.UserID, operator.lastActor.UserID)

	target, ok := tool.(agenttoolports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(
		t,
		agenttoolports.ToolTarget{Resource: permission.ResourceAccountingSync, ID: finding.ID},
		target,
	)
}

func TestResolveAccountingDrift_RefusesAnUnknownDirectionOrARefusedFix(t *testing.T) {
	t.Parallel()

	finding := openDriftFinding()
	operator := newDriftOperator(finding)
	tool := newResolveAccountingDriftTool(operator)

	err := tool.Execute(t.Context(), agenttoolstest.AccountingSyncParams(map[string]any{
		"findingId": finding.ID.String(),
		"direction": "Sideways",
	}))
	require.Error(t, err)

	operator.refuse = errortypes.NewBusinessError("INV-1001 already has a change on its way")
	params := agenttoolstest.AccountingSyncParams(map[string]any{
		"findingId": finding.ID.String(),
		"direction": "PushTrenovaValue",
	})
	require.Error(t, tool.Execute(t.Context(), params))
	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err, "a refusal previews as a warning")
	assert.NotEmpty(t, preview.Warnings)
	assert.Nil(t, operator.resolved)
}

func TestDismissAccountingDrift_DismissesWithTheNoteAsTheAgent(t *testing.T) {
	t.Parallel()

	finding := openDriftFinding()
	operator := newDriftOperator(finding)
	tool := newDismissAccountingDriftTool(operator)
	params := agenttoolstest.AccountingSyncParams(map[string]any{
		"findingId": finding.ID.String(),
		"note":      "  Rounding on the\n customer's side.  ",
	})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Rounding on the customer's side.")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.dismissed)
	assert.Equal(t, "Rounding on the customer's side.", operator.dismissed.Note)
	assert.Same(
		t,
		params.Actor,
		operator.lastActor,
		"the service decides what an agent may dismiss",
	)

	require.Error(t, tool.Execute(t.Context(), agenttoolstest.AccountingSyncParams(map[string]any{
		"findingId": finding.ID.String(),
	})))
}

func TestCheckAccountingDrift_StartsTheCheck(t *testing.T) {
	t.Parallel()

	finding := openDriftFinding()
	operator := newDriftOperator(finding)
	tool := newCheckAccountingDriftTool(operator)
	params := agenttoolstest.AccountingSyncParams(map[string]any{"system": "QuickBooksOnline"})

	preview, err := tool.(agenttoolports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "QuickBooks Online")
	assert.Empty(t, operator.checked)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, integration.TypeQuickBooksOnline, operator.checked)
}

func TestAccountingDriftTools_AskAPersonFirst(t *testing.T) {
	t.Parallel()

	operator := &fakeDriftOperator{}
	resolve := newResolveAccountingDriftTool(operator).(agenttoolports.ToolPolicyDeclarer).Policy()
	assert.Equal(t, agent.TierActWithApproval, resolve.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, resolve.Egress)
	for _, tool := range []agenttoolports.AgentTool{
		newDismissAccountingDriftTool(operator),
		newCheckAccountingDriftTool(operator),
	} {
		policy := tool.(agenttoolports.ToolPolicyDeclarer).Policy()
		assert.Equal(t, agent.TierActWithApproval, policy.MaxTier, policy.Name)
		assert.Equal(t, permission.ResourceAccountingSync, policy.Resource, policy.Name)
	}
}
