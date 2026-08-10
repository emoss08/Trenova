package tenderjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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
			Name:        CarrierTenderWorkflowName,
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
func CarrierTenderWorkflow(ctx workflow.Context, in *WorkflowInput) error {
	ctx = workflow.WithActivityOptions(ctx, tenderActivityOptions)

	var a *Activities
	base := TenderActivityInput{
		TenantInfo: in.TenantInfo(),
		TenderID:   in.TenderID,
	}

	var plan *serviceports.TenderPlan
	if err := workflow.ExecuteActivity(
		ctx, a.LoadPlanActivity, &base,
	).Get(ctx, &plan); err != nil {
		return err
	}
	if plan == nil || plan.Status != tender.StatusActive {
		return nil
	}

	signals := workflow.GetSignalChannel(ctx, TenderSignalName)

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
)

func runSequential(
	ctx workflow.Context,
	a *Activities,
	base TenderActivityInput,
	plan *serviceports.TenderPlan,
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
	plan *serviceports.TenderPlan,
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
		case tender.OfferStatusSent:
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

		finished, closedOfferID, err := awaitAny(
			ctx, a, base, nextDeadline, signals, isOutstanding,
		)
		if err != nil {
			return err
		}
		if finished {
			return nil
		}

		now := workflow.Now(ctx).Unix()
		remaining := outstanding[:0]
		for _, o := range outstanding {
			if o.OfferID == closedOfferID {
				continue
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
	offer serviceports.TenderPlanOffer,
) (deadline int64, live bool, err error) {
	if offer.Status == tender.OfferStatusSent {
		if offer.ExpiresAt != nil {
			return *offer.ExpiresAt, true, nil
		}
		return workflow.Now(ctx).Unix(), true, nil
	}

	in := base
	in.OfferID = offer.OfferID
	var result serviceports.TenderDispatchResult
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

		outcome, oerr := handleSignal(ctx, a, base, sig, func(id pulid.ID) bool {
			return id == offerID
		})
		if oerr != nil {
			return false, oerr
		}
		switch outcome {
		case outcomeFinished:
			return true, nil
		case outcomeOfferClosed:
			return false, nil
		case outcomeContinue:
		}
	}
}

// awaitAny is the broadcast counterpart: any outstanding offer can respond.
func awaitAny(
	ctx workflow.Context,
	a *Activities,
	base TenderActivityInput,
	deadline int64,
	signals workflow.ReceiveChannel,
	isOutstanding func(pulid.ID) bool,
) (finished bool, closedOfferID pulid.ID, err error) {
	for {
		sig, timedOut := receiveOrDeadline(ctx, signals, deadline)
		if timedOut {
			return false, pulid.Nil, nil
		}

		outcome, oerr := handleSignal(ctx, a, base, sig, isOutstanding)
		if oerr != nil {
			return false, pulid.Nil, oerr
		}
		switch outcome {
		case outcomeFinished:
			return true, pulid.Nil, nil
		case outcomeOfferClosed:
			return false, sig.OfferID, nil
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
	sig Signal,
	relevant func(pulid.ID) bool,
) (responseOutcome, error) {
	switch sig.Kind {
	case SignalKindCancel:
		in := base
		in.Reason = sig.Reason
		in.ActorUserID = sig.ActorUserID
		if err := workflow.ExecuteActivity(
			ctx, a.CancelTenderActivity, &in,
		).Get(ctx, nil); err != nil {
			return outcomeContinue, err
		}
		return outcomeFinished, nil
	case SignalKindSweepNudge:
		return outcomeContinue, nil
	case SignalKindResponse:
		if !relevant(sig.OfferID) {
			return outcomeContinue, nil
		}
		return handleResponse(ctx, a, base, sig)
	default:
		return outcomeContinue, nil
	}
}

func handleResponse(
	ctx workflow.Context,
	a *Activities,
	base TenderActivityInput,
	sig Signal,
) (responseOutcome, error) {
	in := base
	in.OfferID = sig.OfferID
	in.Source = sig.Source
	in.Reason = sig.DeclineReason

	switch sig.Action {
	case tender.ResponseActionDecline:
		if err := workflow.ExecuteActivity(
			ctx, a.DeclineOfferActivity, &in,
		).Get(ctx, nil); err != nil {
			return outcomeContinue, err
		}
		return outcomeOfferClosed, nil
	case tender.ResponseActionAccept:
		var result serviceports.TenderAcceptResult
		if err := workflow.ExecuteActivity(
			ctx, a.AcceptOfferActivity, &in,
		).Get(ctx, &result); err != nil {
			return outcomeContinue, err
		}
		switch {
		case result.Ignored:
			return outcomeContinue, nil
		case result.NeedsReview:
			reviewIn := in
			reviewIn.Reason = result.ReviewReason
			if err := workflow.ExecuteActivity(
				ctx, a.MarkNeedsReviewActivity, &reviewIn,
			).Get(ctx, nil); err != nil {
				return outcomeContinue, err
			}
			return outcomeFinished, nil
		default:
			if err := workflow.ExecuteActivity(
				ctx, a.FinalizeAcceptedActivity, &in,
			).Get(ctx, nil); err != nil {
				return outcomeContinue, err
			}
			return outcomeFinished, nil
		}
	default:
		return outcomeContinue, nil
	}
}

// receiveOrDeadline blocks for the next signal or the absolute epoch
// deadline, whichever comes first.
func receiveOrDeadline(
	ctx workflow.Context,
	signals workflow.ReceiveChannel,
	deadline int64,
) (Signal, bool) {
	remaining := time.Duration(deadline-workflow.Now(ctx).Unix()) * time.Second
	if remaining <= 0 {
		var sig Signal
		if signals.ReceiveAsync(&sig) {
			return sig, false
		}
		return Signal{}, true
	}

	timer := workflow.NewTimer(ctx, remaining)

	var sig Signal
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
