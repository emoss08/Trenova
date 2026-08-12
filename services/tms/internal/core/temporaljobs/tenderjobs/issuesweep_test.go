package tenderjobs

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

type issueCall struct {
	tenantInfo pagination.TenantInfo
	offerID    pulid.ID
}

type fakeIssueLifecycle struct {
	serviceports.TenderLifecycle

	calls []issueCall
	errOn map[pulid.ID]error
}

func (f *fakeIssueLifecycle) IssueRateConfirmation(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	offerID pulid.ID,
) error {
	f.calls = append(f.calls, issueCall{tenantInfo: tenantInfo, offerID: offerID})
	return f.errOn[offerID]
}

type fakeSweepTenderRepo struct {
	repositories.TenderRepository

	req      repositories.ListAcceptedMissingRateConfirmationRequest
	tenders  []*tender.Tender
	listErr  error
	callSeen bool
}

func (f *fakeSweepTenderRepo) ListAcceptedMissingRateConfirmation(
	_ context.Context,
	req repositories.ListAcceptedMissingRateConfirmationRequest,
) ([]*tender.Tender, error) {
	f.callSeen = true
	f.req = req
	return f.tenders, f.listErr
}

func acceptedTenderRow(offerID pulid.ID) *tender.Tender {
	return &tender.Tender{
		ID:             pulid.MustNew("tnd_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ShipmentMoveID: pulid.MustNew("smv_"),
		Status:         tender.StatusAccepted,
		AcceptedOfferID: func() *pulid.ID {
			id := offerID
			return &id
		}(),
	}
}

func TestSweepMissingRateConfirmationsActivity_IssuesPerRow(t *testing.T) {
	firstOffer := pulid.MustNew("tof_")
	secondOffer := pulid.MustNew("tof_")
	first := acceptedTenderRow(firstOffer)
	second := acceptedTenderRow(secondOffer)

	repo := &fakeSweepTenderRepo{tenders: []*tender.Tender{first, second}}
	lifecycle := &fakeIssueLifecycle{}
	a := &Activities{repo: repo, lifecycle: lifecycle, l: zap.NewNop()}

	result, err := a.SweepMissingRateConfirmationsActivity(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 2, result.Checked)
	assert.Equal(t, 2, result.Issued)
	assert.Equal(t, 0, result.Failed)

	require.Len(t, lifecycle.calls, 2)
	assert.Equal(t, firstOffer, lifecycle.calls[0].offerID)
	assert.Equal(t, first.OrganizationID, lifecycle.calls[0].tenantInfo.OrgID)
	assert.Equal(t, first.BusinessUnitID, lifecycle.calls[0].tenantInfo.BuID)
	assert.Equal(t, secondOffer, lifecycle.calls[1].offerID)

	assert.True(t, repo.callSeen)
	assert.Equal(t, issueSweepBatchLimit, repo.req.Limit)
	assert.Equal(t,
		int64(issueSweepLookbackSeconds-issueSweepGraceSeconds),
		repo.req.AcceptedBefore-repo.req.AcceptedAfter,
	)
	assert.Less(t, repo.req.AcceptedAfter, repo.req.AcceptedBefore)
}

func TestSweepMissingRateConfirmationsActivity_LogsAndContinuesOnFailure(t *testing.T) {
	failingOffer := pulid.MustNew("tof_")
	healthyOffer := pulid.MustNew("tof_")
	failing := acceptedTenderRow(failingOffer)
	healthy := acceptedTenderRow(healthyOffer)

	lifecycle := &fakeIssueLifecycle{
		errOn: map[pulid.ID]error{failingOffer: errors.New("templates unavailable")},
	}
	a := &Activities{
		repo:      &fakeSweepTenderRepo{tenders: []*tender.Tender{failing, healthy}},
		lifecycle: lifecycle,
		l:         zap.NewNop(),
	}

	result, err := a.SweepMissingRateConfirmationsActivity(t.Context())

	require.NoError(t, err, "one bad row must never abort the sweep")
	assert.Equal(t, 2, result.Checked)
	assert.Equal(t, 1, result.Issued)
	assert.Equal(t, 1, result.Failed)
	require.Len(t, lifecycle.calls, 2)
}

func TestSweepMissingRateConfirmationsActivity_SkipsRowsWithoutAcceptedOffer(t *testing.T) {
	orphan := acceptedTenderRow(pulid.MustNew("tof_"))
	orphan.AcceptedOfferID = nil

	lifecycle := &fakeIssueLifecycle{}
	a := &Activities{
		repo:      &fakeSweepTenderRepo{tenders: []*tender.Tender{orphan, nil}},
		lifecycle: lifecycle,
		l:         zap.NewNop(),
	}

	result, err := a.SweepMissingRateConfirmationsActivity(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 0, result.Checked)
	assert.Empty(t, lifecycle.calls)
}

func TestSweepMissingRateConfirmationsActivity_PropagatesListFailure(t *testing.T) {
	a := &Activities{
		repo:      &fakeSweepTenderRepo{listErr: errors.New("connection reset")},
		lifecycle: &fakeIssueLifecycle{},
		l:         zap.NewNop(),
	}

	result, err := a.SweepMissingRateConfirmationsActivity(t.Context())

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestRateConfirmationIssueSweepWorkflow_RunsTheSweepActivity(t *testing.T) {
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(RateConfirmationIssueSweepWorkflow)

	var a *Activities
	env.OnActivity(a.SweepMissingRateConfirmationsActivity, mock.Anything).
		Return(&RateConIssueSweepResult{Checked: 3, Issued: 2, Failed: 1}, nil).
		Once()

	env.ExecuteWorkflow(RateConfirmationIssueSweepWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	result := new(RateConIssueSweepResult)
	require.NoError(t, env.GetWorkflowResult(result))
	assert.Equal(t, 3, result.Checked)
	assert.Equal(t, 2, result.Issued)
	assert.Equal(t, 1, result.Failed)
	env.AssertExpectations(t)
}
