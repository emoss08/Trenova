package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	syncStartDay   = int64(1_767_225_600)
	syncEnabledDay = int64(1_772_323_200)
)

type fakeSyncOperator struct {
	summary *serviceports.AccountingSyncSummary
	records map[pulid.ID]*accountingsync.AccountingSyncRecord

	retried    *serviceports.RetryAccountingSyncRequest
	skipped    *serviceports.SkipAccountingSyncRequest
	paused     *serviceports.PauseAccountingSyncRequest
	resumed    *serviceports.PauseAccountingSyncRequest
	backfilled *serviceports.RequestAccountingBackfillRequest
}

func (f *fakeSyncOperator) Summary(
	context.Context,
	pagination.TenantInfo,
	integration.Type,
) (*serviceports.AccountingSyncSummary, error) {
	return f.summary, nil
}

func (f *fakeSyncOperator) GetRecord(
	_ context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingSyncRecord, error) {
	record, ok := f.records[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("Accounting sync record not found")
	}
	out := *record
	return &out, nil
}

func (f *fakeSyncOperator) Retry(
	_ context.Context,
	req *serviceports.RetryAccountingSyncRequest,
) (int64, error) {
	f.retried = req
	return int64(len(req.IDs)), nil
}

func (f *fakeSyncOperator) Skip(
	_ context.Context,
	req *serviceports.SkipAccountingSyncRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	f.skipped = req
	return f.records[req.ID], nil
}

func (f *fakeSyncOperator) Pause(
	_ context.Context,
	req *serviceports.PauseAccountingSyncRequest,
) (*accountingsync.AccountingConnection, error) {
	f.paused = req
	return f.summary.Connection, nil
}

func (f *fakeSyncOperator) Resume(
	_ context.Context,
	req *serviceports.PauseAccountingSyncRequest,
) (*accountingsync.AccountingConnection, error) {
	f.resumed = req
	return f.summary.Connection, nil
}

func (f *fakeSyncOperator) RequestBackfill(
	_ context.Context,
	req *serviceports.RequestAccountingBackfillRequest,
) (*accountingsync.AccountingBackfill, error) {
	f.backfilled = req
	return &accountingsync.AccountingBackfill{ID: pulid.MustNew("acctbf_")}, nil
}

func syncingSummary() *serviceports.AccountingSyncSummary {
	start, enabled := syncStartDay, syncEnabledDay
	return &serviceports.AccountingSyncSummary{
		IntegrationType: integration.TypeQuickBooksOnline,
		ProviderName:    "QuickBooks Online",
		Connection: &accountingsync.AccountingConnection{
			ID:                  pulid.MustNew("acctc_"),
			IntegrationType:     integration.TypeQuickBooksOnline,
			Status:              accountingsync.ConnectionStatusConnected,
			SetupStep:           accountingsync.SetupStepComplete,
			ExternalCompanyName: "Acme Freight",
			SyncStartDate:       &start,
			SyncEnabledAt:       &enabled,
			AutoSync:            true,
		},
		Counts: []repositories.AccountingSyncStatusCount{
			{Status: accountingsync.SyncStatusQueued, Count: 7},
			{Status: accountingsync.SyncStatusRetrying, Count: 2},
			{Status: accountingsync.SyncStatusDeadLettered, Count: 5},
		},
		Attention: []repositories.AccountingSyncAttentionGroup{
			{
				Status:        accountingsync.SyncStatusDeadLettered,
				ErrorCategory: accountingsync.SyncErrorTransient,
				Count:         5,
			},
			{
				Status:        accountingsync.SyncStatusBlocked,
				ErrorCategory: accountingsync.SyncErrorMapping,
				Count:         3,
			},
		},
	}
}

func syncRecord(status accountingsync.SyncStatus) *accountingsync.AccountingSyncRecord {
	return &accountingsync.AccountingSyncRecord{
		ID:           pulid.MustNew("acctsr_"),
		ObjectType:   accountingsync.SyncObjectInvoice,
		ObjectID:     pulid.MustNew("inv_"),
		ObjectNumber: "INV-1042",
		Status:       status,
	}
}

func newSyncOperator(records ...*accountingsync.AccountingSyncRecord) *fakeSyncOperator {
	operator := &fakeSyncOperator{
		summary: syncingSummary(),
		records: make(map[pulid.ID]*accountingsync.AccountingSyncRecord, len(records)),
	}
	for _, record := range records {
		operator.records[record.ID] = record
	}
	return operator
}

func syncParams(params map[string]any) serviceports.ToolExecuteParams {
	out := executeParams(params)
	out.IdempotencyKey = "call-1"
	out.Actor.PrincipalType = serviceports.PrincipalTypeUser
	return out
}

func TestRetryAccountingSync_RetriesNamedRecordsAsTheActor(t *testing.T) {
	t.Parallel()

	blocked := syncRecord(accountingsync.SyncStatusBlocked)
	dead := syncRecord(accountingsync.SyncStatusDeadLettered)
	operator := newSyncOperator(blocked, dead)
	tool := newRetryAccountingSyncTool(operator)
	params := syncParams(map[string]any{
		"system":        "QuickBooksOnline",
		"syncRecordIds": []any{blocked.ID.String(), dead.ID.String(), blocked.ID.String()},
	})

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.retried)
	assert.Equal(t, []pulid.ID{blocked.ID, dead.ID}, operator.retried.IDs)
	assert.Equal(t, params.Actor.UserID, operator.retried.UserID)
	assert.Equal(t, params.OrganizationID, operator.retried.TenantInfo.OrgID)
	assert.Equal(t, integration.TypeQuickBooksOnline, operator.retried.IntegrationType)

	sim, err := tool.(serviceports.ToolSimulator).Simulate(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, sim.Summary, "2 document(s)")
	require.Len(t, sim.Changes, 2)
	assert.Equal(t, "Invoice INV-1042", sim.Changes[0].Field)
	assert.Equal(t, "Queued", sim.Changes[0].To)
}

func TestRetryAccountingSync_RetriesByErrorCategory(t *testing.T) {
	t.Parallel()

	operator := newSyncOperator()
	tool := newRetryAccountingSyncTool(operator)
	params := syncParams(map[string]any{
		"system":          "QuickBooksOnline",
		"errorCategories": []any{"Transient", "RateLimited"},
	})

	sim, err := tool.(serviceports.ToolSimulator).Simulate(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, sim.Summary, "Transient or RateLimited")
	assert.Contains(t, sim.Summary, "5 are blocked")
	assert.Nil(t, operator.retried, "simulating changes nothing")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.retried)
	assert.Empty(t, operator.retried.IDs)
	assert.Equal(t, []accountingsync.SyncErrorCategory{
		accountingsync.SyncErrorTransient,
		accountingsync.SyncErrorRateLimited,
	}, operator.retried.ErrorCategories)
}

func TestRetryAccountingSync_RefusesWhatItCannotRetry(t *testing.T) {
	t.Parallel()

	synced := syncRecord(accountingsync.SyncStatusSynced)
	held := syncRecord(accountingsync.SyncStatusAwaitingApproval)
	operator := newSyncOperator(synced, held)
	tool := newRetryAccountingSyncTool(operator)

	tooMany := make([]any, 0, maxSyncRetryIDs+1)
	for range maxSyncRetryIDs + 1 {
		tooMany = append(tooMany, pulid.MustNew("acctsr_").String())
	}
	for _, params := range []map[string]any{
		{"system": "QuickBooksOnline"},
		{"system": "Xero", "errorCategories": []any{"Transient"}},
		{"system": "QuickBooksOnline", "errorCategories": []any{"Timeout"}},
		{"system": "QuickBooksOnline", "syncRecordIds": []any{"nope"}},
		{"system": "QuickBooksOnline", "syncRecordIds": tooMany},
		{"system": "QuickBooksOnline", "syncRecordIds": []any{synced.ID.String()}},
		{"system": "QuickBooksOnline", "syncRecordIds": []any{held.ID.String()}},
		{"system": "QuickBooksOnline", "syncRecordIds": []any{pulid.MustNew("acctsr_").String()}},
	} {
		require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), syncParams(params)), params)
	}

	operator.summary.Connection.SyncEnabledAt = nil
	require.Error(t, tool.Execute(t.Context(), syncParams(map[string]any{
		"system":          "QuickBooksOnline",
		"errorCategories": []any{"Transient"},
	})), "nothing is retried before sending has started")

	withoutKey := executeParams(map[string]any{
		"system":          "QuickBooksOnline",
		"errorCategories": []any{"Transient"},
	})
	require.ErrorIs(t, tool.Execute(t.Context(), withoutKey), ErrMissingIdempotencyKey)
	assert.Nil(t, operator.retried)
}

func TestSkipAccountingSync_SkipsWithTheReasonAndNamesItsTarget(t *testing.T) {
	t.Parallel()

	blocked := syncRecord(accountingsync.SyncStatusBlocked)
	operator := newSyncOperator(blocked)
	tool := newSkipAccountingSyncTool(operator)
	params := syncParams(map[string]any{
		"syncRecordId": blocked.ID.String(),
		"reason":       "  Entered by hand in QuickBooks\n on the 3rd.  ",
	})

	sim, err := tool.(serviceports.ToolSimulator).Simulate(t.Context(), params)
	require.NoError(t, err)
	assert.Equal(t, []agent.FieldChange{{Field: "status", From: "Blocked", To: "Skipped"}}, sim.Changes)
	assert.Nil(t, operator.skipped)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.skipped)
	assert.Equal(t, blocked.ID, operator.skipped.ID)
	assert.Equal(t, "Entered by hand in QuickBooks on the 3rd.", operator.skipped.Reason)
	assert.Equal(t, params.Actor.UserID, operator.skipped.UserID)

	target, ok := tool.(serviceports.TargetedTool).Target(params.Params)
	require.True(t, ok)
	assert.Equal(t, serviceports.ToolTarget{
		Resource: permission.ResourceAccountingSync,
		ID:       blocked.ID,
	}, target)
}

func TestSkipAccountingSync_NeedsAReasonAndAnUnsentRecord(t *testing.T) {
	t.Parallel()

	synced := syncRecord(accountingsync.SyncStatusSynced)
	sending := syncRecord(accountingsync.SyncStatusInFlight)
	blocked := syncRecord(accountingsync.SyncStatusBlocked)
	operator := newSyncOperator(synced, sending, blocked)
	tool := newSkipAccountingSyncTool(operator)

	for _, params := range []map[string]any{
		{"syncRecordId": blocked.ID.String()},
		{"syncRecordId": blocked.ID.String(), "reason": "   "},
		{"syncRecordId": "nope", "reason": "duplicate"},
		{"syncRecordId": synced.ID.String(), "reason": "duplicate"},
		{"syncRecordId": sending.ID.String(), "reason": "duplicate"},
	} {
		require.Error(t, tool.Execute(t.Context(), syncParams(params)), params)
	}
	assert.Nil(t, operator.skipped)
}

func TestPauseAccountingSync_PausesWithAReason(t *testing.T) {
	t.Parallel()

	operator := newSyncOperator()
	tool := newPauseAccountingSyncTool(operator)

	require.Error(t, tool.Execute(t.Context(), syncParams(map[string]any{"system": "QuickBooksOnline"})))
	assert.Nil(t, operator.paused)

	params := syncParams(map[string]any{
		"system": "QuickBooksOnline",
		"reason": "QuickBooks is down for maintenance until 6pm.",
	})
	sim, err := tool.(serviceports.ToolSimulator).Simulate(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, sim.Summary, "9 document(s) waiting")
	assert.Equal(t, "Paused", sim.Changes[0].To)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.paused)
	assert.Equal(t, "QuickBooks is down for maintenance until 6pm.", operator.paused.Reason)

	paused := int64(1_790_000_000)
	operator.summary.Connection.PausedAt = &paused
	operator.summary.Connection.PausedReason = "Month-end close"
	operator.paused = nil
	err = tool.Execute(t.Context(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Month-end close")
	assert.Nil(t, operator.paused)
}

func TestResumeAccountingSync_OnlyResumesWhatIsPaused(t *testing.T) {
	t.Parallel()

	operator := newSyncOperator()
	tool := newResumeAccountingSyncTool(operator)
	params := syncParams(map[string]any{"system": "QuickBooksOnline"})

	require.Error(t, tool.Execute(t.Context(), params))
	assert.Nil(t, operator.resumed)

	paused := int64(1_790_000_000)
	operator.summary.Connection.PausedAt = &paused
	operator.summary.Connection.PausedReason = "Month-end close"

	sim, err := tool.(serviceports.ToolSimulator).Simulate(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, sim.Summary, "9 queued document(s)")
	assert.Contains(t, sim.Summary, "Month-end close")

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.resumed)
	assert.Equal(t, params.Actor.UserID, operator.resumed.UserID)
}

func TestRequestAccountingBackfill_SendsTheRangeAPersonAskedFor(t *testing.T) {
	t.Parallel()

	operator := newSyncOperator()
	tool := newRequestAccountingBackfillTool(operator)
	params := syncParams(map[string]any{
		"system":        "QuickBooksOnline",
		"from":          "2026-01-15",
		"to":            "2026-02-10",
		"documentTypes": []any{"Invoice", "CustomerPayment"},
	})

	sim, err := tool.(serviceports.ToolSimulator).Simulate(t.Context(), params)
	require.NoError(t, err)
	assert.Contains(t, sim.Summary, "2026-01-15 to 2026-02-10")
	assert.Contains(t, sim.Summary, "arrive twice")
	assert.Nil(t, operator.backfilled)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, operator.backfilled)
	assert.Equal(t, int64(1_768_435_200), *operator.backfilled.RangeStart)
	assert.Equal(t, int64(1_770_767_999), *operator.backfilled.RangeEnd)
	assert.Equal(t, []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectCustomerPayment,
	}, operator.backfilled.ObjectTypes)
	assert.Equal(t, params.Actor.UserID, operator.backfilled.UserID)
}

func TestRequestAccountingBackfill_DefaultsToTheWholeGap(t *testing.T) {
	t.Parallel()

	operator := newSyncOperator()
	tool := newRequestAccountingBackfillTool(operator)

	require.NoError(t, tool.Execute(t.Context(), syncParams(map[string]any{"system": "QuickBooksOnline"})))
	require.NotNil(t, operator.backfilled)
	assert.Equal(t, syncStartDay, *operator.backfilled.RangeStart)
	assert.Equal(t, syncEnabledDay, *operator.backfilled.RangeEnd)
	assert.Equal(t, accountingsync.BackfillObjectTypes(), operator.backfilled.ObjectTypes)
}

func TestRequestAccountingBackfill_RefusesWhatCannotBeBackfilled(t *testing.T) {
	t.Parallel()

	operator := newSyncOperator()
	tool := newRequestAccountingBackfillTool(operator)

	for _, params := range []map[string]any{
		{"system": "QuickBooksOnline", "from": "2025-12-01"},
		{"system": "QuickBooksOnline", "from": "2026-02-10", "to": "2026-01-15"},
		{"system": "QuickBooksOnline", "from": "15/01/2026"},
		{"system": "QuickBooksOnline", "documentTypes": []any{"Customer"}},
	} {
		require.Error(t, tool.Execute(t.Context(), syncParams(params)), params)
	}

	operator.summary.ActiveBackfill = &accountingsync.AccountingBackfill{
		Status: accountingsync.BackfillStatusRunning,
	}
	require.Error(t, tool.Execute(t.Context(), syncParams(map[string]any{"system": "QuickBooksOnline"})))
	operator.summary.ActiveBackfill = nil

	agentParams := syncParams(map[string]any{"system": "QuickBooksOnline"})
	agentParams.Actor.PrincipalType = serviceports.PrincipalTypeAgent
	require.Error(t, tool.Execute(t.Context(), agentParams), "a backfill is a person's to start")
	assert.Nil(t, operator.backfilled)
}

func TestAccountingSyncTools_CarryTheirSafetyTiers(t *testing.T) {
	t.Parallel()

	operator := newSyncOperator()
	cases := []struct {
		tool       serviceports.AgentTool
		resource   permission.Resource
		operation  permission.Operation
		tier       agent.AutonomyTier
		reversible bool
	}{
		{
			newRetryAccountingSyncTool(operator),
			permission.ResourceAccountingSync, permission.OpUpdate, agent.TierAutoExecute, false,
		},
		{
			newSkipAccountingSyncTool(operator),
			permission.ResourceAccountingSync, permission.OpUpdate, agent.TierActWithApproval, false,
		},
		{
			newPauseAccountingSyncTool(operator),
			permission.ResourceAccountingSync, permission.OpUpdate, agent.TierAutoExecute, true,
		},
		{
			newResumeAccountingSyncTool(operator),
			permission.ResourceAccountingSync, permission.OpUpdate, agent.TierActWithApproval, false,
		},
		{
			newRequestAccountingBackfillTool(operator),
			permission.ResourceAccountingIntegration, permission.OpManage, agent.TierActWithApproval, false,
		},
	}

	for _, tc := range cases {
		policy := tc.tool.Policy()
		assert.Equal(t, tc.resource, policy.Resource, tc.tool.Name())
		assert.Equal(t, tc.operation, policy.Operation, tc.tool.Name())
		assert.Equal(t, tc.tier, policy.DefaultTier, tc.tool.Name())
		assert.Equal(t, tc.tier, policy.MaxTier, tc.tool.Name())
		assert.Equal(t, tc.reversible, policy.Reversible, tc.tool.Name())
		assert.Equal(t, []agent.EgressClass{agent.EgressInternal}, policy.Egress, tc.tool.Name())
	}
	assert.True(t, newRetryAccountingSyncTool(operator).Policy().Idempotent)
	assert.True(t, permission.IsAgentAllowed(permission.ResourceAccountingSync, permission.OpUpdate))
	assert.False(t,
		permission.IsAgentAllowed(permission.ResourceAccountingIntegration, permission.OpManage),
		"backfilling stays with a person who manages the integration",
	)
}
