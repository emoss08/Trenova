package tenderjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/tenderjobs/tendercontract"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var tenderRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    30 * time.Second,
	MaximumAttempts:    5,
}

var tenderActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy:         tenderRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        tendercontract.CarrierTenderWorkflowName,
			Fn:          CarrierTenderWorkflow,
			TaskQueue:   temporaltype.TaskQueueDispatch.String(),
			Description: "Drive one carrier tender through dispatch, expiry, and response",
		},
	}
}

// CarrierTenderWorkflow owns every state transition for one tender. Responses
// arrive as signals from the single service funnel; expiry is the workflow's
// own timer. Activities are all CAS-idempotent, so the workflow restarts
// cleanly from whatever the rows say.
func CarrierTenderWorkflow(ctx workflow.Context, in *tendercontract.WorkflowInput) error {
	ctx = workflow.WithActivityOptions(ctx, tenderActivityOptions)

	var a *Activities
	base := TenderActivityInput{
		TenantInfo: in.TenantInfo(),
		TenderID:   in.TenderID,
	}

	var plan *tenderports.TenderPlan
	if err := workflow.ExecuteActivity(
		ctx, a.LoadPlanActivity, &base,
	).Get(ctx, &plan); err != nil {
		return err
	}
	if plan == nil || plan.Status != tender.StatusActive {
		return nil
	}

	signals := workflow.GetSignalChannel(ctx, tendercontract.TenderSignalName)

	if in.Mode.IsSequential() {
		return runSequential(ctx, a, base, plan, signals)
	}
	return runBroadcast(ctx, a, base, plan, signals)
}

type responseOutcome int

const (
	outcomeContinue responseOutcome = iota
	outcomeFinished
	outcomeOfferClosed
	// outcomeCountered means the offer now holds a counter waiting on a
	// person, until the deadline the signal result carries.
	outcomeCountered
)

type signalResult struct {
	outcome  responseOutcome
	deadline int64
}

func runSequential(
	ctx workflow.Context,
	a *Activities,
	base TenderActivityInput,
	plan *tenderports.TenderPlan,
	signals workflow.ReceiveChannel,
) error {
	for _, offer := range plan.Offers {
		if offer.Status.IsTerminal() {
			continue
		}

		deadline, live, err := ensureDispatched(ctx, a, base, offer)
		if err != nil {
			return err
		}
		if !live {
			continue
		}

		finished, ferr := awaitOffer(ctx, a, base, offer.OfferID, deadline, signals, nil)
		if ferr != nil {
			return ferr
		}
		if finished {
			return nil
		}
	}

	return workflow.ExecuteActivity(ctx, a.MarkExhaustedActivity, &base).Get(ctx, nil)
}

func runBroadcast(
	ctx workflow.Context,
	a *Activities,
	base TenderActivityInput,
	plan *tenderports.TenderPlan,
	signals workflow.ReceiveChannel,
) error {
	type outstandingOffer struct {
		OfferID   pulid.ID
		ExpiresAt int64
	}

	outstanding := make([]outstandingOffer, 0, len(plan.Offers))
	for _, offer := range plan.Offers {
		switch offer.Status {
		case tender.OfferStatusPending:
			deadline, live, err := ensureDispatched(ctx, a, base, offer)
			if err != nil {
				return err
			}
			if live {
				outstanding = append(outstanding, outstandingOffer{
					OfferID:   offer.OfferID,
					ExpiresAt: deadline,
				})
			}
		case tender.OfferStatusSent, tender.OfferStatusCountered:
			deadline := int64(0)
			if offer.ExpiresAt != nil {
				deadline = *offer.ExpiresAt
			}
			outstanding = append(outstanding, outstandingOffer{
				OfferID:   offer.OfferID,
				ExpiresAt: deadline,
			})
		default:
		}
	}

	for len(outstanding) > 0 {
		nextDeadline := outstanding[0].ExpiresAt
		for _, o := range outstanding[1:] {
			if o.ExpiresAt < nextDeadline {
				nextDeadline = o.ExpiresAt
			}
		}

		isOutstanding := func(offerID pulid.ID) bool {
			for _, o := range outstanding {
				if o.OfferID == offerID {
					return true
				}
			}
			return false
		}

		waited, err := awaitAny(
			ctx, a, base, nextDeadline, signals, isOutstanding,
		)
		if err != nil {
			return err
		}
		if waited.finished {
			return nil
		}

		now := workflow.Now(ctx).Unix()
		remaining := outstanding[:0]
		for _, o := range outstanding {
			if o.OfferID == waited.closedOfferID {
				continue
			}
			if o.OfferID == waited.counteredOfferID {
				o.ExpiresAt = waited.counterDeadline
			}
			if o.ExpiresAt <= now {
				expireIn := base
				expireIn.OfferID = o.OfferID
				if expErr := workflow.ExecuteActivity(
					ctx, a.ExpireOfferActivity, &expireIn,
				).Get(ctx, nil); expErr != nil {
					return expErr
				}
				continue
			}
			remaining = append(remaining, o)
		}
		outstanding = remaining
	}

	return workflow.ExecuteActivity(ctx, a.MarkExhaustedActivity, &base).Get(ctx, nil)
}

// ensureDispatched sends a pending offer (or adopts an already-sent one on
// replay) and reports whether it is live and until when.
func ensureDispatched(
	ctx workflow.Context,
	a *Activities,
	base TenderActivityInput,
	offer tenderports.TenderPlanOffer,
) (deadline int64, live bool, err error) {
	if offer.Status.IsAwaitingAnswer() {
		if offer.ExpiresAt != nil {
			return *offer.ExpiresAt, true, nil
		}
		return workflow.Now(ctx).Unix(), true, nil
	}

	in := base
	in.OfferID = offer.OfferID
	var result tenderports.TenderDispatchResult
	if err = workflow.ExecuteActivity(
		ctx, a.DispatchOfferActivity, &in,
	).Get(ctx, &result); err != nil {
		return 0, false, err
	}
	if !result.Delivered {
		return 0, false, nil
	}
	return result.ExpiresAt, true, nil
}

// awaitOffer drives one live offer to a conclusion: a response, a cancel, or
// expiry. It reports finished=true when the tender as a whole is done.
func awaitOffer(
	ctx workflow.Context,
	a *Activities,
	base TenderActivityInput,
	offerID pulid.ID,
	deadline int64,
	signals workflow.ReceiveChannel,
	_ func(pulid.ID) bool,
) (finished bool, err error) {
	for {
		sig, timedOut := receiveOrDeadline(ctx, signals, deadline)
		if timedOut {
			in := base
			in.OfferID = offerID
			return false, workflow.ExecuteActivity(
				ctx, a.ExpireOfferActivity, &in,
			).Get(ctx, nil)
		}

		res, oerr := handleSignal(ctx, a, base, sig, func(id pulid.ID) bool {
			return id == offerID
		})
		if oerr != nil {
			return false, oerr
		}
		switch res.outcome {
		case outcomeFinished:
			return true, nil
		case outcomeOfferClosed:
			return false, nil
		case outcomeCountered:
			deadline = res.deadline
		case outcomeContinue:
		}
	}
}

// awaitAnyResult is how one broadcast wait ended: the tender finished, one
// offer closed, or one offer took a counter with a new deadline.
type awaitAnyResult struct {
	finished         bool
	closedOfferID    pulid.ID
	counteredOfferID pulid.ID
	counterDeadline  int64
}

// awaitAny is the broadcast counterpart: any outstanding offer can respond.
func awaitAny(
	ctx workflow.Context,
	a *Activities,
	base TenderActivityInput,
	deadline int64,
	signals workflow.ReceiveChannel,
	isOutstanding func(pulid.ID) bool,
) (awaitAnyResult, error) {
	for {
		sig, timedOut := receiveOrDeadline(ctx, signals, deadline)
		if timedOut {
			return awaitAnyResult{}, nil
		}

		res, oerr := handleSignal(ctx, a, base, sig, isOutstanding)
		if oerr != nil {
			return awaitAnyResult{}, oerr
		}
		switch res.outcome {
		case outcomeFinished:
			return awaitAnyResult{finished: true}, nil
		case outcomeOfferClosed:
			return awaitAnyResult{closedOfferID: sig.OfferID}, nil
		case outcomeCountered:
			return awaitAnyResult{
				counteredOfferID: sig.OfferID,
				counterDeadline:  res.deadline,
			}, nil
		case outcomeContinue:
		}
	}
}

// handleSignal applies one drained signal. relevant filters responses to the
// offers this wait cares about; stale responses for other offers are ignored
// because the service funnel already recorded them as late.
func handleSignal(
	ctx workflow.Context,
	a *Activities,
	base TenderActivityInput,
	sig tendercontract.Signal,
	relevant func(pulid.ID) bool,
) (signalResult, error) {
	switch sig.Kind {
	case tendercontract.SignalKindCancel:
		in := base
		in.Reason = sig.Reason
		in.ActorUserID = sig.ActorUserID
		if err := workflow.ExecuteActivity(
			ctx, a.CancelTenderActivity, &in,
		).Get(ctx, nil); err != nil {
			return signalResult{}, err
		}
		return signalResult{outcome: outcomeFinished}, nil
	case SignalKindSweepNudge:
		return signalResult{}, nil
	case tendercontract.SignalKindResponse:
		if !relevant(sig.OfferID) {
			// The response was in flight when its offer expired or was
			// superseded, so the service-side gate never saw it as late.
			// Preserve it here or the carrier's answer vanishes entirely.
			lateIn := base
			lateIn.OfferID = sig.OfferID
			lateIn.Action = sig.Action
			lateIn.Source = sig.Source
			lateIn.ActorUserID = sig.ActorUserID
			if err := workflow.ExecuteActivity(
				ctx, a.RecordLateResponseActivity, &lateIn,
			).Get(ctx, nil); err != nil {
				workflow.GetLogger(ctx).Error(
					"failed to record late tender response", "error", err,
				)
			}
			return signalResult{}, nil
		}
		return handleResponse(ctx, a, base, sig)
	default:
		return signalResult{}, nil
	}
}

func handleResponse(
	ctx workflow.Context,
	a *Activities,
	base TenderActivityInput,
	sig tendercontract.Signal,
) (signalResult, error) {
	in := base
	in.OfferID = sig.OfferID
	in.Source = sig.Source
	in.Reason = sig.DeclineReason
	in.ActorUserID = sig.ActorUserID

	switch sig.Action {
	case tender.ResponseActionDecline:
		if err := workflow.ExecuteActivity(
			ctx, a.DeclineOfferActivity, &in,
		).Get(ctx, nil); err != nil {
			return signalResult{}, err
		}
		return signalResult{outcome: outcomeOfferClosed}, nil
	case tender.ResponseActionAccept:
		return acceptOffer(ctx, a, &in)
	case tender.ResponseActionCounter:
		in.CounterRate = sig.CounterRate
		in.CounterNote = sig.CounterNote
		var result tenderports.TenderCounterResult
		if err := workflow.ExecuteActivity(
			ctx, a.RecordCounterActivity, &in,
		).Get(ctx, &result); err != nil {
			return signalResult{}, err
		}
		switch {
		case result.Ignored:
			return signalResult{}, nil
		case result.AutoAccept:
			return acceptOffer(ctx, a, &in)
		default:
			return signalResult{outcome: outcomeCountered, deadline: result.DecisionDeadline}, nil
		}
	default:
		return signalResult{}, nil
	}
}

// acceptOffer lands an acceptance, whether the carrier accepted the offer or
// somebody, or the policy, took the carrier's counter.
func acceptOffer(
	ctx workflow.Context,
	a *Activities,
	in *TenderActivityInput,
) (signalResult, error) {
	var result tenderports.TenderAcceptResult
	if err := workflow.ExecuteActivity(
		ctx, a.AcceptOfferActivity, in,
	).Get(ctx, &result); err != nil {
		return signalResult{}, err
	}
	switch {
	case result.Ignored:
		return signalResult{}, nil
	case result.NeedsReview:
		reviewIn := *in
		reviewIn.Reason = result.ReviewReason
		if err := workflow.ExecuteActivity(
			ctx, a.MarkNeedsReviewActivity, &reviewIn,
		).Get(ctx, nil); err != nil {
			return signalResult{}, err
		}
		return signalResult{outcome: outcomeFinished}, nil
	default:
		if err := workflow.ExecuteActivity(
			ctx, a.FinalizeAcceptedActivity, in,
		).Get(ctx, nil); err != nil {
			return signalResult{}, err
		}
		// The acceptance is committed; its paperwork is best-effort here.
		// A persistent issuance failure already raised a dispatch
		// notification service-side and must never fail the workflow.
		if err := workflow.ExecuteActivity(
			ctx, a.IssueRateConfirmationActivity, in,
		).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Error(
				"failed to issue the rate confirmation for the accepted tender",
				"error", err,
			)
		}
		return signalResult{outcome: outcomeFinished}, nil
	}
}

// receiveOrDeadline blocks for the next signal or the absolute epoch
// deadline, whichever comes first.
func receiveOrDeadline(
	ctx workflow.Context,
	signals workflow.ReceiveChannel,
	deadline int64,
) (tendercontract.Signal, bool) {
	remaining := time.Duration(deadline-workflow.Now(ctx).Unix()) * time.Second
	if remaining <= 0 {
		var sig tendercontract.Signal
		if signals.ReceiveAsync(&sig) {
			return sig, false
		}
		return tendercontract.Signal{}, true
	}

	timer := workflow.NewTimer(ctx, remaining)

	var sig tendercontract.Signal
	var timedOut bool
	sel := workflow.NewSelector(ctx)
	sel.AddReceive(signals, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, &sig)
	})
	sel.AddFuture(timer, func(workflow.Future) {
		timedOut = true
	})
	sel.Select(ctx)

	return sig, timedOut
}
