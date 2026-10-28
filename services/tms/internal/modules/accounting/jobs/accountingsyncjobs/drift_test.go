package accountingsyncjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/jobs/accountingsyncjobs/accountingsynccontract"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/testutil/agenteventstest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type DriftWorkflowSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func TestDriftWorkflowSuite(t *testing.T) {
	suite.Run(t, new(DriftWorkflowSuite))
}

func (s *DriftWorkflowSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *DriftWorkflowSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func driftPayload() *accountingsynccontract.DriftPayload {
	return &accountingsynccontract.DriftPayload{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ConnectionID:   pulid.MustNew("acctc_"),
	}
}

func (s *DriftWorkflowSuite) TestComparesEveryBatchThenBalancesThenFinishes() {
	var a *Activities
	first, second := pulid.MustNew("acctsr_"), pulid.MustNew("acctsr_")
	customer := pulid.MustNew("cus_")
	var seen []*accountingsynccontract.DriftPayload
	record := func(args mock.Arguments) {
		copied := *args.Get(1).(*accountingsynccontract.DriftPayload)
		seen = append(seen, &copied)
	}
	s.env.OnActivity(a.ReconcileAccountingDriftActivity, mock.Anything, mock.Anything).
		Return(&accountingdriftports.AccountingDriftBatchResult{
			LastID: first, More: true, Compared: 200, Opened: 15, Events: 15,
		}, nil).Once().Run(record)
	s.env.OnActivity(a.ReconcileAccountingDriftActivity, mock.Anything, mock.Anything).
		Return(&accountingdriftports.AccountingDriftBatchResult{
			LastID: second, Compared: 40, Opened: 9, Events: 5, Resolved: 2,
		}, nil).Once().Run(record)
	s.env.OnActivity(a.ReconcileAccountingDriftBalancesActivity, mock.Anything, mock.Anything).
		Return(&accountingdriftports.AccountingDriftBalanceResult{
			LastCustomerID: customer, More: true, Customers: 50, Opened: 1,
		}, nil).Once().Run(record)
	s.env.OnActivity(a.ReconcileAccountingDriftBalancesActivity, mock.Anything, mock.Anything).
		Return(&accountingdriftports.AccountingDriftBalanceResult{Customers: 3}, nil).
		Once().Run(record)
	s.env.OnActivity(a.FinishAccountingDriftCheckActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	s.env.ExecuteWorkflow(ReconcileAccountingDriftWorkflow, driftPayload())

	s.Require().True(s.env.IsWorkflowCompleted())
	s.Require().NoError(s.env.GetWorkflowError())
	var result accountingsynccontract.DriftRunResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.Equal(4, result.Batches)
	s.Equal(240, result.Compared)
	s.Equal(53, result.Customers)
	s.Equal(25, result.Opened)
	s.Equal(2, result.Resolved)
	s.Equal(20, result.Events)

	s.Require().Len(seen, 4)
	s.True(seen[0].AfterID.IsNil())
	s.Equal(driftEventsPerRun, seen[0].EventsLeft)
	s.Equal(first, seen[1].AfterID, "the next batch starts after the last record")
	s.Equal(5, seen[1].EventsLeft, "events are capped across the whole run")
	s.True(seen[2].AfterCustomerID.IsNil())
	s.Equal(0, seen[2].EventsLeft)
	s.Equal(customer, seen[3].AfterCustomerID)
}

func (s *DriftWorkflowSuite) TestStopsWithoutFinishingWhenTheConnectionHolds() {
	var a *Activities
	s.env.OnActivity(a.ReconcileAccountingDriftActivity, mock.Anything, mock.Anything).
		Return(&accountingdriftports.AccountingDriftBatchResult{Held: true}, nil).Once()

	s.env.ExecuteWorkflow(ReconcileAccountingDriftWorkflow, driftPayload())

	s.Require().NoError(s.env.GetWorkflowError())
	var result accountingsynccontract.DriftRunResult
	s.Require().NoError(s.env.GetWorkflowResult(&result))
	s.True(result.Held)
}

func (s *DriftWorkflowSuite) TestAFailedBatchFailsTheRunWithoutStampingIt() {
	var a *Activities
	s.env.OnActivity(a.ReconcileAccountingDriftActivity, mock.Anything, mock.Anything).
		Return(nil, errors.New("QuickBooks did not answer"))

	s.env.ExecuteWorkflow(ReconcileAccountingDriftWorkflow, driftPayload())

	s.Require().Error(s.env.GetWorkflowError())
}

func (s *DriftWorkflowSuite) TestContinuesAsNewAfterTheBatchLimitKeepingItsPlace() {
	var a *Activities
	last := pulid.MustNew("acctsr_")
	s.env.OnActivity(a.ReconcileAccountingDriftActivity, mock.Anything, mock.Anything).
		Return(&accountingdriftports.AccountingDriftBatchResult{LastID: last, More: true}, nil).
		Times(driftBatchesPerRun)

	s.env.ExecuteWorkflow(ReconcileAccountingDriftWorkflow, driftPayload())

	err := s.env.GetWorkflowError()
	s.Require().Error(err)
	s.True(workflow.IsContinueAsNewError(err))
}

func TestDriftWorkflowsAndSchedulesAreRegistered(t *testing.T) {
	t.Parallel()
	names := map[string]bool{}
	for _, def := range RegisterWorkflows() {
		names[def.Name] = true
	}
	require.True(t, names[accountingsynccontract.ReconcileAccountingDriftWorkflowName])
	require.True(t, names[KickAccountingDriftWorkflowName])
	require.True(t, names[AnnounceAccountingReconciliationDueWorkflowName])

	schedules := map[string]bool{}
	for _, sched := range NewScheduleProvider().GetSchedules() {
		schedules[sched.ID] = true
	}
	assert.True(t, schedules["accounting-drift-reconcile"])
	assert.True(t, schedules["accounting-drift-weekly"])
}

type fakeActiveConnections struct {
	accountingconnectionports.AccountingConnectionRepository
	rows []*accountingconnection.AccountingConnection
}

func (f *fakeActiveConnections) ListActive(
	_ context.Context,
	req accountingconnectionports.ListActiveAccountingConnectionsRequest,
) ([]*accountingconnection.AccountingConnection, error) {
	out := []*accountingconnection.AccountingConnection{}
	for _, row := range f.rows {
		if row.ID.String() > req.AfterID.String() && len(out) < req.Limit {
			out = append(out, row)
		}
	}
	return out, nil
}

type fakeDriftChecker struct {
	checked []pulid.ID
}

func (f *fakeDriftChecker) CheckNow(_ context.Context, _ pagination.TenantInfo, id pulid.ID) error {
	f.checked = append(f.checked, id)
	return nil
}

func driftConnections() (syncing, paused *accountingconnection.AccountingConnection) {
	start, enabled, pausedAt := int64(1), int64(1), int64(2)
	syncing = &accountingconnection.AccountingConnection{
		ID:             pulid.MustNew("acctc_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Status:         accountingconnection.ConnectionStatusConnected,
		SetupStep:      accountingconnection.SetupStepComplete,
		SyncStartDate:  &start,
		SyncEnabledAt:  &enabled,
	}
	copied := *syncing
	paused = &copied
	paused.ID = pulid.MustNew("acctc_")
	paused.PausedAt = &pausedAt
	return syncing, paused
}

func TestKickAndWeeklyVisitOnlyConnectionsThatAreSyncing(t *testing.T) {
	t.Parallel()
	syncing, paused := driftConnections()
	checker := &fakeDriftChecker{}
	events := &agenteventstest.Recorder{}
	a := NewActivities(ActivitiesParams{
		ConnectionsRepository: &fakeActiveConnections{
			rows: []*accountingconnection.AccountingConnection{syncing, paused},
		},
		DriftChecker: checker,
		Publisher:    events,
		Logger:       zap.NewNop(),
	})

	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	env.RegisterActivity(a)

	value, err := env.ExecuteActivity(a.KickAccountingDriftActivity)
	require.NoError(t, err)
	var kicked KickDriftResult
	require.NoError(t, value.Get(&kicked))
	assert.Equal(t, 1, kicked.Started)
	assert.Equal(t, 1, kicked.Connections)
	assert.Equal(t, []pulid.ID{syncing.ID}, checker.checked)

	value, err = env.ExecuteActivity(a.AnnounceAccountingReconciliationActivity)
	require.NoError(t, err)
	var announced AnnounceReconciliationResult
	require.NoError(t, value.Get(&announced))
	assert.Equal(t, 1, announced.Connections)
	published := events.Published()
	require.Len(t, published, 1)
	assert.Equal(t, agent.EventAccountingReconciliationDue, published[0].Kind)
	assert.Equal(t, syncing.ID, published[0].SubjectID)
	assert.Equal(t, syncing.OrganizationID, published[0].TenantInfo.OrgID)
}
