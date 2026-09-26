package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDriftOperator struct {
	findings   map[pulid.ID]*accountingsync.AccountingDriftFinding
	refuse     error
	resolved   *serviceports.ResolveAccountingDriftRequest
	dismissed  *serviceports.DismissAccountingDriftRequest
	checked    integration.Type
	lastActor  *serviceports.RequestActor
	overviewID pulid.ID
}

func (f *fakeDriftOperator) finding(id pulid.ID) (*accountingsync.AccountingDriftFinding, error) {
	finding, ok := f.findings[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("Drift finding not found")
	}
	out := *finding
	return &out, nil
}

func (f *fakeDriftOperator) Get(
	_ context.Context,
	req *serviceports.GetAccountingDriftFindingRequest,
) (*accountingsync.AccountingDriftFinding, error) {
	return f.finding(req.ID)
}

func (f *fakeDriftOperator) PreviewResolve(
	_ context.Context,
	req *serviceports.ResolveAccountingDriftRequest,
	actor *serviceports.RequestActor,
) (*serviceports.AccountingDriftFixPreview, error) {
	f.lastActor = actor
	if f.refuse != nil {
		return nil, f.refuse
	}
	finding, err := f.finding(req.ID)
	if err != nil {
		return nil, err
	}
	return &serviceports.AccountingDriftFixPreview{
		Finding:   finding,
		Direction: req.Direction,
		FixObject: accountingsync.DriftFixCreditMemo,
		Summary:   "Post a credit memo of 50.00 USD against INV-1001.",
	}, nil
}

func (f *fakeDriftOperator) Resolve(
	_ context.Context,
	req *serviceports.ResolveAccountingDriftRequest,
	actor *serviceports.RequestActor,
) (*accountingsync.AccountingDriftFinding, error) {
	f.resolved = req
	f.lastActor = actor
	return f.finding(req.ID)
}

func (f *fakeDriftOperator) PreviewDismiss(
	_ context.Context,
	req *serviceports.DismissAccountingDriftRequest,
	actor *serviceports.RequestActor,
) (*serviceports.AccountingDriftFixPreview, error) {
	f.lastActor = actor
	if f.refuse != nil {
		return nil, f.refuse
	}
	finding, err := f.finding(req.ID)
	if err != nil {
		return nil, err
	}
	return &serviceports.AccountingDriftFixPreview{
		Finding: finding,
		Summary: "Dismiss the difference in the total of INV-1001 and keep both sides as they are.",
	}, nil
}

func (f *fakeDriftOperator) Dismiss(
	_ context.Context,
	req *serviceports.DismissAccountingDriftRequest,
	actor *serviceports.RequestActor,
) (*accountingsync.AccountingDriftFinding, error) {
	f.dismissed = req
	f.lastActor = actor
	return f.finding(req.ID)
}

func (f *fakeDriftOperator) Overview(
	_ context.Context,
	_ *serviceports.AccountingDriftOverviewRequest,
) (*serviceports.AccountingDriftOverview, error) {
	return &serviceports.AccountingDriftOverview{
		ConnectionID: f.overviewID,
		ProviderName: "QuickBooks Online",
	}, nil
}

func (f *fakeDriftOperator) CheckNow(
	_ context.Context,
	_ pagination.TenantInfo,
	integrationType integration.Type,
) (*serviceports.AccountingDriftOverview, error) {
	f.checked = integrationType
	return &serviceports.AccountingDriftOverview{ConnectionID: f.overviewID}, nil
}

func openDriftFinding() *accountingsync.AccountingDriftFinding {
	trenova, provider := int64(125_000), int64(120_000)
	return accountingsync.NewAccountingDriftFinding(&accountingsync.DriftObservation{
		ConnectionID:  pulid.MustNew("acctc_"),
		ObjectType:    accountingsync.SyncObjectInvoice,
		ObjectID:      pulid.MustNew("inv_"),
		ObjectNumber:  "INV-1001",
		Kind:          accountingsync.DriftAmountMismatch,
		CurrencyCode:  "USD",
		TrenovaMinor:  &trenova,
		ProviderMinor: &provider,
		At:            1_790_000_000,
	})
}

func newDriftOperator(finding *accountingsync.AccountingDriftFinding) *fakeDriftOperator {
	return &fakeDriftOperator{
		findings:   map[pulid.ID]*accountingsync.AccountingDriftFinding{finding.ID: finding},
		overviewID: finding.ConnectionID,
	}
}

func TestResolveAccountingDrift_PreviewsThenFixesAsTheApprover(t *testing.T) {
	t.Parallel()

	finding := openDriftFinding()
	operator := newDriftOperator(finding)
	tool := newResolveAccountingDriftTool(operator)
	params := syncParams(map[string]any{
		"findingId": finding.ID.String(),
		"direction": "AdjustTrenova",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Would post a credit memo of 50.00 USD")
	require.Len(t, preview.Changes, 1)
	assert.Equal(t, finding.ID, preview.Changes[0].EntityID)
	assert.Nil(t, operator.resolved, "previewing fixes nothing")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.resolved)
	assert.Equal(t, accountingsync.DriftAdjustTrenova, operator.resolved.Direction)
	assert.Equal(t, params.Actor.UserID, operator.lastActor.UserID)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, serviceports.ToolTarget{Resource: permission.ResourceAccountingSync, ID: finding.ID}, target)
}

func TestResolveAccountingDrift_RefusesAnUnknownDirectionOrARefusedFix(t *testing.T) {
	t.Parallel()

	finding := openDriftFinding()
	operator := newDriftOperator(finding)
	tool := newResolveAccountingDriftTool(operator)

	err := tool.Execute(t.Context(), syncParams(map[string]any{
		"findingId": finding.ID.String(),
		"direction": "Sideways",
	}))
	require.Error(t, err)

	operator.refuse = errortypes.NewBusinessError("INV-1001 already has a change on its way")
	params := syncParams(map[string]any{
		"findingId": finding.ID.String(),
		"direction": "PushTrenovaValue",
	})
	require.Error(t, tool.Execute(t.Context(), params))
	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err, "a refusal previews as a warning")
	assert.NotEmpty(t, preview.Warnings)
	assert.Nil(t, operator.resolved)
}

func TestDismissAccountingDrift_DismissesWithTheNoteAsTheAgent(t *testing.T) {
	t.Parallel()

	finding := openDriftFinding()
	operator := newDriftOperator(finding)
	tool := newDismissAccountingDriftTool(operator)
	params := syncParams(map[string]any{
		"findingId": finding.ID.String(),
		"note":      "  Rounding on the\n customer's side.  ",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "Rounding on the customer's side.")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.dismissed)
	assert.Equal(t, "Rounding on the customer's side.", operator.dismissed.Note)
	assert.Same(t, params.Actor, operator.lastActor, "the service decides what an agent may dismiss")

	require.Error(t, tool.Execute(t.Context(), syncParams(map[string]any{
		"findingId": finding.ID.String(),
	})))
}

func TestCheckAccountingDrift_StartsTheCheck(t *testing.T) {
	t.Parallel()

	finding := openDriftFinding()
	operator := newDriftOperator(finding)
	tool := newCheckAccountingDriftTool(operator)
	params := syncParams(map[string]any{"system": "QuickBooksOnline"})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, preview.Summary, "QuickBooks Online")
	assert.Empty(t, operator.checked)

	require.NoError(t, tool.Execute(t.Context(), params))
	assert.Equal(t, integration.TypeQuickBooksOnline, operator.checked)
}

func TestAccountingDriftTools_AskAPersonFirst(t *testing.T) {
	t.Parallel()

	operator := &fakeDriftOperator{}
	resolve := newResolveAccountingDriftTool(operator).(serviceports.ToolPolicyDeclarer).Policy()
	assert.Equal(t, agent.TierActWithApproval, resolve.MaxTier)
	assert.Equal(t, []agent.EgressClass{agent.EgressMoney}, resolve.Egress)
	for _, tool := range []serviceports.AgentTool{
		newDismissAccountingDriftTool(operator),
		newCheckAccountingDriftTool(operator),
	} {
		policy := tool.(serviceports.ToolPolicyDeclarer).Policy()
		assert.Equal(t, agent.TierActWithApproval, policy.MaxTier, policy.Name)
		assert.Equal(t, permission.ResourceAccountingSync, policy.Resource, policy.Name)
	}
}
