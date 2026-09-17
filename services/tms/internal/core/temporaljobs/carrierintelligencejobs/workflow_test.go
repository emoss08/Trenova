package carrierintelligencejobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/services/carrierintelservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type WorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func TestWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}

func (s *WorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *WorkflowTestSuite) AfterTest(_, _ string) {
	s.env.AssertExpectations(s.T())
}

func tenants(n int) []temporaljobs.TenantWorkItem {
	out := make([]temporaljobs.TenantWorkItem, 0, n)
	for range n {
		out = append(out, temporaljobs.TenantWorkItem{
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
		})
	}
	return out
}

type pagedTenants struct {
	mu     sync.Mutex
	pages  [][]temporaljobs.TenantWorkItem
	afters []*temporaljobs.TenantWorkItem
	always bool
}

func (p *pagedTenants) list(
	_ context.Context,
	payload *ListTenantsPayload,
) (*temporaljobs.TenantPage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.afters = append(p.afters, payload.After)
	idx := len(p.afters) - 1
	if p.always {
		return &temporaljobs.TenantPage{Tenants: tenants(1), HasMore: true}, nil
	}
	if idx >= len(p.pages) {
		return &temporaljobs.TenantPage{}, nil
	}
	return &temporaljobs.TenantPage{
		Tenants: p.pages[idx],
		HasMore: idx < len(p.pages)-1,
	}, nil
}

func (s *WorkflowTestSuite) TestSweepPagesTenantsAndIsolatesFailures() {
	var a *Activities
	first, second := tenants(2), tenants(1)
	failing := first[1].OrganizationID
	pager := &pagedTenants{pages: [][]temporaljobs.TenantWorkItem{first, second}}

	s.env.OnActivity(a.ListTenantsActivity, mock.Anything, mock.Anything).Return(pager.list)
	s.env.OnActivity(a.SweepTenantActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, payload *TenantPayload) (*SweepTenantResult, error) {
			if payload.OrganizationID == failing {
				return nil, temporal.NewNonRetryableApplicationError(
					"provider rejected key", "BusinessError", errors.New("unauthorized"),
				)
			}
			return &SweepTenantResult{Recomputed: 2, Enrolled: 1, Changed: 5, Refreshed: 3}, nil
		},
	)

	s.env.ExecuteWorkflow(CarrierIntelSweepWorkflow, &FanOutInput{})

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())

	var result temporaljobs.TenantRunResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(3, result.TenantsScanned)
	s.Equal(2, result.TenantsProcessed)
	s.Equal(12, result.RecordsProcessed)
	s.Equal(1, result.FailureCount)

	s.Require().Len(pager.afters, 2)
	s.Nil(pager.afters[0])
	s.Require().NotNil(pager.afters[1])
	s.Equal(first[1].OrganizationID, pager.afters[1].OrganizationID)
}

func (s *WorkflowTestSuite) TestSweepContinuesAsNewAfterPageLimit() {
	var a *Activities
	pager := &pagedTenants{always: true}

	s.env.OnActivity(a.ListTenantsActivity, mock.Anything, mock.Anything).Return(pager.list)
	s.env.OnActivity(a.SweepTenantActivity, mock.Anything, mock.Anything).
		Return(&SweepTenantResult{}, nil)

	s.env.ExecuteWorkflow(CarrierIntelSweepWorkflow, &FanOutInput{})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Require().Error(err)
	s.True(workflow.IsContinueAsNewError(err))
	s.Len(pager.afters, 5)
}

func (s *WorkflowTestSuite) TestSweepResumesFromContinuationCursor() {
	var a *Activities
	cursor := tenants(1)[0]
	pager := &pagedTenants{pages: [][]temporaljobs.TenantWorkItem{tenants(1)}}

	s.env.OnActivity(a.ListTenantsActivity, mock.Anything, mock.Anything).Return(pager.list)
	s.env.OnActivity(a.SweepTenantActivity, mock.Anything, mock.Anything).
		Return(&SweepTenantResult{}, nil)

	s.env.ExecuteWorkflow(CarrierIntelSweepWorkflow, &FanOutInput{After: &cursor})

	s.NoError(s.env.GetWorkflowError())
	s.Require().NotEmpty(pager.afters)
	s.Require().NotNil(pager.afters[0])
	s.Equal(cursor.OrganizationID, pager.afters[0].OrganizationID)
}

func (s *WorkflowTestSuite) runReconcileOn(start time.Time) []bool {
	var a *Activities
	pager := &pagedTenants{pages: [][]temporaljobs.TenantWorkItem{tenants(2)}}
	var mu sync.Mutex
	drift := make([]bool, 0, 2)

	s.env.SetStartTime(start)
	s.env.OnActivity(a.ListTenantsActivity, mock.Anything, mock.Anything).Return(pager.list)
	s.env.OnActivity(a.ReconcileTenantActivity, mock.Anything, mock.Anything).Return(
		func(_ context.Context, payload *ReconcileTenantPayload) (*carrierintelservice.ReconcileResult, error) {
			mu.Lock()
			drift = append(drift, payload.DriftCheck)
			mu.Unlock()
			return &carrierintelservice.ReconcileResult{Desired: 4}, nil
		},
	)

	s.env.ExecuteWorkflow(CarrierIntelReconcileWorkflow, &FanOutInput{})
	s.NoError(s.env.GetWorkflowError())

	var result temporaljobs.TenantRunResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(8, result.RecordsProcessed)
	return drift
}

func (s *WorkflowTestSuite) TestReconcileChecksDriftOnSundays() {
	drift := s.runReconcileOn(time.Date(2026, time.September, 20, 6, 0, 0, 0, time.UTC))
	s.Equal([]bool{true, true}, drift)
}

func (s *WorkflowTestSuite) TestReconcileSkipsDriftOnWeekdays() {
	drift := s.runReconcileOn(time.Date(2026, time.September, 21, 6, 0, 0, 0, time.UTC))
	s.Equal([]bool{false, false}, drift)
}

func (s *WorkflowTestSuite) TestMaintenanceSurvivesGlobalFailure() {
	var a *Activities
	pager := &pagedTenants{pages: [][]temporaljobs.TenantWorkItem{tenants(2)}}

	s.env.OnActivity(a.GlobalMaintenanceActivity, mock.Anything).Return(
		(*carrierintelservice.MaintenanceResult)(nil),
		temporal.NewNonRetryableApplicationError("rollup failed", "BusinessError", nil),
	).Once()
	s.env.OnActivity(a.ListTenantsActivity, mock.Anything, mock.Anything).Return(pager.list)
	s.env.OnActivity(a.PruneTenantActivity, mock.Anything, mock.Anything).Return(7, nil)

	s.env.ExecuteWorkflow(CarrierIntelMaintenanceWorkflow, &FanOutInput{})

	s.NoError(s.env.GetWorkflowError())
	var result temporaljobs.TenantRunResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(2, result.TenantsProcessed)
	s.Equal(14, result.RecordsProcessed)
}

func (s *WorkflowTestSuite) TestMaintenanceContinuationSkipsGlobalWork() {
	var a *Activities
	cursor := tenants(1)[0]
	pager := &pagedTenants{pages: [][]temporaljobs.TenantWorkItem{tenants(1)}}

	s.env.OnActivity(a.ListTenantsActivity, mock.Anything, mock.Anything).Return(pager.list)
	s.env.OnActivity(a.PruneTenantActivity, mock.Anything, mock.Anything).Return(1, nil)

	s.env.ExecuteWorkflow(CarrierIntelMaintenanceWorkflow, &FanOutInput{After: &cursor})

	s.NoError(s.env.GetWorkflowError())
	var result temporaljobs.TenantRunResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(1, result.TenantsProcessed)
}

func (s *WorkflowTestSuite) TestDigestCountsEvents() {
	var a *Activities
	pager := &pagedTenants{pages: [][]temporaljobs.TenantWorkItem{tenants(3)}}

	s.env.OnActivity(a.ListTenantsActivity, mock.Anything, mock.Anything).Return(pager.list)
	s.env.OnActivity(a.DigestTenantActivity, mock.Anything, mock.Anything).
		Return(&carrierintelservice.DigestResult{Events: 2}, nil)

	s.env.ExecuteWorkflow(CarrierIntelDigestWorkflow, &FanOutInput{})

	s.NoError(s.env.GetWorkflowError())
	var result temporaljobs.TenantRunResult
	s.NoError(s.env.GetWorkflowResult(&result))
	s.Equal(3, result.TenantsProcessed)
	s.Equal(6, result.RecordsProcessed)
}
