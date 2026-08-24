package tenderjobs

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func newTenderWorkflowTestEnv(t *testing.T) *testsuite.TestWorkflowEnvironment {
	t.Helper()
	suite := &testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterWorkflow(CarrierTenderWorkflow)
	return env
}

type testPlanFixture struct {
	input    *WorkflowInput
	plan     *serviceports.TenderPlan
	offerOne pulid.ID
	offerTwo pulid.ID
}

func newPlanFixture(mode tender.Mode) *testPlanFixture {
	tenderID := pulid.MustNew("tnd_")
	offerOne := pulid.MustNew("tof_")
	offerTwo := pulid.MustNew("tof_")

	return &testPlanFixture{
		input: &WorkflowInput{
			TenderID:       tenderID,
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			Mode:           mode,
		},
		plan: &serviceports.TenderPlan{
			TenderID: tenderID,
			Status:   tender.StatusActive,
			Mode:     mode,
			Offers: []serviceports.TenderPlanOffer{
				{
					OfferID:         offerOne,
					Rank:            1,
					Status:          tender.OfferStatusPending,
					OfferTTLSeconds: 600,
				},
				{
					OfferID:         offerTwo,
					Rank:            2,
					Status:          tender.OfferStatusPending,
					OfferTTLSeconds: 600,
				},
			},
		},
		offerOne: offerOne,
		offerTwo: offerTwo,
	}
}

func dispatchResultIn(ttl time.Duration) *serviceports.TenderDispatchResult {
	return &serviceports.TenderDispatchResult{
		Delivered: true,
		ExpiresAt: time.Now().Add(ttl).Unix(),
	}
}

func TestCarrierTenderWorkflow_WaterfallExpiresThroughAllRanks(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.Anything).
		Return(dispatchResultIn(10*time.Minute), nil).Twice()
	env.OnActivity(a.ExpireOfferActivity, mock.Anything, mock.Anything).
		Return(nil).Twice()
	env.OnActivity(a.MarkExhaustedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_WaterfallAcceptShortCircuits(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.AcceptOfferActivity, mock.Anything, mock.Anything).
		Return(&serviceports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(TenderSignalName, Signal{
			Kind:    SignalKindResponse,
			OfferID: fixture.offerOne,
			Action:  tender.ResponseActionAccept,
			Source:  tender.ResponseSourceEmail,
		})
	}, time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "DispatchOfferActivity", mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	))
	env.AssertNotCalled(t, "MarkExhaustedActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_AcceptIssuesRateConfirmation(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.AcceptOfferActivity, mock.Anything, mock.Anything).
		Return(&serviceports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(TenderSignalName, Signal{
			Kind:    SignalKindResponse,
			OfferID: fixture.offerOne,
			Action:  tender.ResponseActionAccept,
			Source:  tender.ResponseSourceEmail,
		})
	}, time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_RateConIssueFailureDoesNotFailWorkflow(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.AcceptOfferActivity, mock.Anything, mock.Anything).
		Return(&serviceports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.Anything).
		Return(errors.New("issuance is down"))

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(TenderSignalName, Signal{
			Kind:    SignalKindResponse,
			OfferID: fixture.offerOne,
			Action:  tender.ResponseActionAccept,
			Source:  tender.ResponseSourceEmail,
		})
	}, time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError(),
		"a failed issuance must never fail the committed acceptance")
	env.AssertNotCalled(t, "MarkNeedsReviewActivity", mock.Anything, mock.Anything)
	env.AssertNotCalled(t, "MarkExhaustedActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_WaterfallDeclineAdvances(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.Anything).
		Return(dispatchResultIn(10*time.Minute), nil).Twice()
	env.OnActivity(a.DeclineOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(nil).Once()
	env.OnActivity(a.ExpireOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	)).Return(nil).Once()
	env.OnActivity(a.MarkExhaustedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(TenderSignalName, Signal{
			Kind:          SignalKindResponse,
			OfferID:       fixture.offerOne,
			Action:        tender.ResponseActionDecline,
			Source:        tender.ResponseSourceEmail,
			DeclineReason: "No capacity",
		})
	}, time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_NeedsReviewHalts(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.Anything).
		Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.AcceptOfferActivity, mock.Anything, mock.Anything).
		Return(&serviceports.TenderAcceptResult{
			NeedsReview:  true,
			ReviewReason: "Carrier insurance expired",
		}, nil).Once()
	env.OnActivity(a.MarkNeedsReviewActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.Reason == "Carrier insurance expired" },
	)).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(TenderSignalName, Signal{
			Kind:    SignalKindResponse,
			OfferID: fixture.offerOne,
			Action:  tender.ResponseActionAccept,
			Source:  tender.ResponseSourceEDI,
		})
	}, time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "FinalizeAcceptedActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_BroadcastFirstAcceptWins(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeSpotBroadcast)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.Anything).
		Return(dispatchResultIn(10*time.Minute), nil).Twice()
	env.OnActivity(a.AcceptOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	)).Return(&serviceports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	)).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(TenderSignalName, Signal{
			Kind:    SignalKindResponse,
			OfferID: fixture.offerTwo,
			Action:  tender.ResponseActionAccept,
			Source:  tender.ResponseSourceEmail,
		})
	}, time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "ExpireOfferActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_CancelSignalWithdraws(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.Anything).
		Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.CancelTenderActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.Reason == "Shipment was canceled" },
	)).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(TenderSignalName, Signal{
			Kind:   SignalKindCancel,
			Reason: "Shipment was canceled",
		})
	}, time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "ExpireOfferActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_DeliveryFailureAdvances(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(&serviceports.TenderDispatchResult{
		Delivered: false,
		Reason:    "Email delivery is not configured",
	}, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	)).Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.ExpireOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	)).Return(nil).Once()
	env.OnActivity(a.MarkExhaustedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_StaleResponseRecordedAsLate(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.Anything).
		Return(dispatchResultIn(10*time.Minute), nil).Twice()
	env.OnActivity(a.DeclineOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(nil).Once()
	env.OnActivity(a.RecordLateResponseActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool {
			return in.OfferID == fixture.offerOne &&
				in.Action == tender.ResponseActionAccept &&
				in.Source == tender.ResponseSourceEmail
		},
	)).Return(nil).Once()
	env.OnActivity(a.ExpireOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	)).Return(nil).Once()
	env.OnActivity(a.MarkExhaustedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(TenderSignalName, Signal{
			Kind:          SignalKindResponse,
			OfferID:       fixture.offerOne,
			Action:        tender.ResponseActionDecline,
			Source:        tender.ResponseSourceEmail,
			DeclineReason: "No capacity",
		})
	}, time.Minute)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(TenderSignalName, Signal{
			Kind:    SignalKindResponse,
			OfferID: fixture.offerOne,
			Action:  tender.ResponseActionAccept,
			Source:  tender.ResponseSourceEmail,
		})
	}, 2*time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_TerminalPlanNoOps(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	fixture.plan.Status = tender.StatusCanceled
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "DispatchOfferActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}
