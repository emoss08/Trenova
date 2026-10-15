package tenderjobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs/tendercontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
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
	input    *tendercontract.WorkflowInput
	plan     *tenderports.TenderPlan
	offerOne pulid.ID
	offerTwo pulid.ID
}

func newPlanFixture(mode tender.Mode) *testPlanFixture {
	tenderID := pulid.MustNew("tnd_")
	offerOne := pulid.MustNew("tof_")
	offerTwo := pulid.MustNew("tof_")

	return &testPlanFixture{
		input: &tendercontract.WorkflowInput{
			TenderID:       tenderID,
			OrganizationID: pulid.MustNew("org_"),
			BusinessUnitID: pulid.MustNew("bu_"),
			Mode:           mode,
		},
		plan: &tenderports.TenderPlan{
			TenderID: tenderID,
			Status:   tender.StatusActive,
			Mode:     mode,
			Offers: []tenderports.TenderPlanOffer{
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

func dispatchResultIn(ttl time.Duration) *tenderports.TenderDispatchResult {
	return &tenderports.TenderDispatchResult{
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
		Return(&tenderports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:    tendercontract.SignalKindResponse,
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
		Return(&tenderports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:    tendercontract.SignalKindResponse,
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
		Return(&tenderports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.Anything).
		Return(errors.New("issuance is down"))

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:    tendercontract.SignalKindResponse,
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
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:          tendercontract.SignalKindResponse,
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
		Return(&tenderports.TenderAcceptResult{
			NeedsReview:  true,
			ReviewReason: "Carrier insurance expired",
		}, nil).Once()
	env.OnActivity(a.MarkNeedsReviewActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.Reason == "Carrier insurance expired" },
	)).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:    tendercontract.SignalKindResponse,
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
	)).Return(&tenderports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	)).Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:    tendercontract.SignalKindResponse,
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
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:   tendercontract.SignalKindCancel,
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
	)).Return(&tenderports.TenderDispatchResult{
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
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:          tendercontract.SignalKindResponse,
			OfferID:       fixture.offerOne,
			Action:        tender.ResponseActionDecline,
			Source:        tender.ResponseSourceEmail,
			DeclineReason: "No capacity",
		})
	}, time.Minute)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:    tendercontract.SignalKindResponse,
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

func TestCarrierTenderWorkflow_RoundAfterABounceStartsAtTheNextRank(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	reofferThree := pulid.MustNew("tof_")
	reofferFour := pulid.MustNew("tof_")
	fixture.plan.Offers = []tenderports.TenderPlanOffer{
		{OfferID: pulid.MustNew("tof_"), Rank: 1, Status: tender.OfferStatusDeclined},
		{OfferID: pulid.MustNew("tof_"), Rank: 2, Status: tender.OfferStatusRevoked},
		{OfferID: pulid.MustNew("tof_"), Rank: 3, Status: tender.OfferStatusSkipped},
		{OfferID: reofferThree, Rank: 3, Status: tender.OfferStatusPending, OfferTTLSeconds: 600},
		{OfferID: pulid.MustNew("tof_"), Rank: 4, Status: tender.OfferStatusSkipped},
		{OfferID: reofferFour, Rank: 4, Status: tender.OfferStatusPending, OfferTTLSeconds: 600},
	}
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == reofferThree },
	)).Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.DeclineOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == reofferThree },
	)).Return(nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == reofferFour },
	)).Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.AcceptOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == reofferFour },
	)).Return(&tenderports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:    tendercontract.SignalKindResponse,
			OfferID: reofferThree,
			Action:  tender.ResponseActionDecline,
			Source:  tender.ResponseSourceEmail,
		})
	}, time.Minute)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:    tendercontract.SignalKindResponse,
			OfferID: reofferFour,
			Action:  tender.ResponseActionAccept,
			Source:  tender.ResponseSourceEmail,
		})
	}, 2*time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "MarkExhaustedActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}

func counterSignal(offerID pulid.ID) tendercontract.Signal {
	return tendercontract.Signal{
		Kind:        tendercontract.SignalKindResponse,
		OfferID:     offerID,
		Action:      tender.ResponseActionCounter,
		Source:      tender.ResponseSourceEmail,
		CounterRate: decimal.NewFromInt(1850),
		CounterNote: "Fuel is up",
	}
}

func TestCarrierTenderWorkflow_TakenCounterAcceptsTheOffer(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.RecordCounterActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool {
			return in.OfferID == fixture.offerOne &&
				in.CounterRate.Equal(decimal.NewFromInt(1850)) &&
				in.CounterNote == "Fuel is up"
		},
	)).Return(&tenderports.TenderCounterResult{
		DecisionDeadline: time.Now().Add(time.Hour).Unix(),
	}, nil).Once()
	env.OnActivity(a.AcceptOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool {
			return in.OfferID == fixture.offerOne && in.Source == tender.ResponseSourceManual
		},
	)).Return(&tenderports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, counterSignal(fixture.offerOne))
	}, time.Minute)
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, tendercontract.Signal{
			Kind:    tendercontract.SignalKindResponse,
			OfferID: fixture.offerOne,
			Action:  tender.ResponseActionAccept,
			Source:  tender.ResponseSourceManual,
		})
	}, 30*time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "ExpireOfferActivity", mock.Anything, mock.Anything)
	env.AssertNotCalled(t, "DispatchOfferActivity", mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	))
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_CounterHoldsPastTheOfferExpiryThenLapses(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeWaterfall)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.RecordCounterActivity, mock.Anything, mock.Anything).
		Return(&tenderports.TenderCounterResult{
			DecisionDeadline: time.Now().Add(time.Hour).Unix(),
		}, nil).Once()
	env.OnActivity(a.ExpireOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerOne },
	)).Return(nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	)).Return(dispatchResultIn(10*time.Minute), nil).Once()
	env.OnActivity(a.ExpireOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	)).Return(nil).Once()
	env.OnActivity(a.MarkExhaustedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	var secondDispatchedAt time.Time
	env.SetOnActivityStartedListener(func(info *activity.Info, _ context.Context, args converter.EncodedValues) {
		if info.ActivityType.Name != "DispatchOfferActivity" {
			return
		}
		var in TenderActivityInput
		if err := args.Get(&in); err == nil && in.OfferID == fixture.offerTwo {
			secondDispatchedAt = env.Now()
		}
	})
	start := env.Now()
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, counterSignal(fixture.offerOne))
	}, time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.False(t, secondDispatchedAt.IsZero())
	require.Greater(t, secondDispatchedAt.Sub(start), 30*time.Minute,
		"a countered offer waits for its decision deadline, not the offer's own expiry")
	env.AssertExpectations(t)
}

func TestCarrierTenderWorkflow_AutoAcceptedCounterAcceptsAtOnce(t *testing.T) {
	env := newTenderWorkflowTestEnv(t)
	fixture := newPlanFixture(tender.ModeSpotBroadcast)
	var a *Activities

	env.OnActivity(a.LoadPlanActivity, mock.Anything, mock.Anything).
		Return(fixture.plan, nil).Once()
	env.OnActivity(a.DispatchOfferActivity, mock.Anything, mock.Anything).
		Return(dispatchResultIn(10*time.Minute), nil).Twice()
	env.OnActivity(a.RecordCounterActivity, mock.Anything, mock.Anything).
		Return(&tenderports.TenderCounterResult{AutoAccept: true}, nil).Once()
	env.OnActivity(a.AcceptOfferActivity, mock.Anything, mock.MatchedBy(
		func(in *TenderActivityInput) bool { return in.OfferID == fixture.offerTwo },
	)).Return(&tenderports.TenderAcceptResult{}, nil).Once()
	env.OnActivity(a.FinalizeAcceptedActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	env.OnActivity(a.IssueRateConfirmationActivity, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(tendercontract.TenderSignalName, counterSignal(fixture.offerTwo))
	}, time.Minute)

	env.ExecuteWorkflow(CarrierTenderWorkflow, fixture.input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	env.AssertNotCalled(t, "ExpireOfferActivity", mock.Anything, mock.Anything)
	env.AssertExpectations(t)
}
