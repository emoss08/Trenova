package billingqueuejobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs/billingqueuecontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	NovaGenerationSweepWorkflowName = "BillingQueueNovaGenerationSweepWorkflow"

	novaPagesPerPass      = 10
	novaIterationsPerRun  = 10
	novaInlineWave        = 8
	novaBatchFirstCheck   = 30 * time.Second
	novaBatchLongestCheck = 5 * time.Minute
	novaCanceledCheck     = 5 * time.Second
	novaBatchWaitLimit    = 25 * time.Hour
	novaSweepTenants      = 500
)

var errNovaBatchOverdue = errors.New("the review batch did not end within its window")

var novaSubmitOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    2,
	},
}

var novaPollOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    5,
	},
}

var novaApplyOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

var novaPublishOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 2},
}

// BillingQueueNovaGenerationWorkflow writes, in bulk, the reviews an
// organization's queue is missing and then its brief. Its id is the
// organization's, so opening the queue and the schedule share one run; a
// request while it runs makes it look again when it is done. A pass someone
// opened the queue for reviews at once, several items at a time; the
// schedule's goes to the provider's batch API a page at a time and is
// recorded as each batch ends. Someone opening the queue while a batch is out
// cancels what has not run, so the rest is reviewed at once without paying
// for it twice. The progress query is what the queue shows while it runs.
func BillingQueueNovaGenerationWorkflow(
	ctx workflow.Context,
	payload *billingqueuecontract.NovaGenerationPayload,
) (*billingqueuecontract.NovaGenerationProgress, error) {
	var a *Activities
	logger := workflow.GetLogger(ctx)
	signals := workflow.GetSignalChannel(ctx, billingqueuecontract.NovaGenerationSignalName)
	progress := billingqueueports.NovaGeneration{Phase: billingqueueports.NovaPhaseReviewing}
	leftUnreviewed := false
	report := func() *billingqueuecontract.NovaGenerationProgress {
		return &billingqueuecontract.NovaGenerationProgress{Generation: progress, LeftUnreviewed: leftUnreviewed}
	}
	if err := workflow.SetQueryHandler(
		ctx,
		billingqueuecontract.NovaGenerationProgressQuery,
		func() (*billingqueuecontract.NovaGenerationProgress, error) { return report(), nil },
	); err != nil {
		return nil, err
	}

	watched := payload.Watched || drainNovaSignals(signals).watched
	for {
		progress = billingqueueports.NovaGeneration{Phase: billingqueueports.NovaPhaseReviewing}
		pass := &novaPass{payload: payload, progress: &progress, signals: signals, watched: watched}
		more, err := generateReviews(ctx, pass)
		if err != nil {
			logger.Warn("billing queue reviews could not be generated", "error", err)
		}

		if !pass.hurried() {
			progress.Phase = billingqueueports.NovaPhaseBriefing
			publishNovaProgress(ctx, payload, true)
			briefCtx := workflow.WithActivityOptions(ctx, briefActivityOptions)
			if err = workflow.ExecuteActivity(briefCtx, a.RefreshBriefActivity, &billingqueuecontract.BriefPayload{
				OrganizationID: payload.OrganizationID,
				BusinessUnitID: payload.BusinessUnitID,
			}).Get(briefCtx, nil); err != nil {
				logger.Warn("billing queue brief could not be generated", "error", err)
			}

			leftUnreviewed = novaLeftUnreviewed(ctx, payload)
			progress.Phase = billingqueueports.NovaPhaseDone
			publishNovaProgress(ctx, payload, true)
		}

		requests := pass.waiting.with(drainNovaSignals(signals))
		if !requests.received && (!more || err != nil) {
			return report(), nil
		}
		if requests.received {
			watched = requests.watched
		}
		payload.Iterations++
		if payload.Iterations >= novaIterationsPerRun {
			return nil, workflow.NewContinueAsNewError(
				ctx,
				BillingQueueNovaGenerationWorkflow,
				&billingqueuecontract.NovaGenerationPayload{
					OrganizationID: payload.OrganizationID,
					BusinessUnitID: payload.BusinessUnitID,
					Watched:        watched,
				},
			)
		}
	}
}

// novaLeftUnreviewed is whether the pass ends with items it could not review,
// which keeps opening the queue from sending them round again at once. Not
// knowing counts as none left, so a reader is never held back by a failed
// check.
func novaLeftUnreviewed(ctx workflow.Context, payload *billingqueuecontract.NovaGenerationPayload) bool {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, novaPollOptions)
	var left bool
	if err := workflow.ExecuteActivity(actx, a.NovaLeftUnreviewedActivity, payload).Get(actx, &left); err != nil {
		workflow.GetLogger(ctx).Warn("could not read whether billing queue items were left unreviewed", "error", err)
		return false
	}

	return left
}

// novaPass is one pass over the queue's unreviewed items.
type novaPass struct {
	payload  *billingqueuecontract.NovaGenerationPayload
	progress *billingqueueports.NovaGeneration
	signals  workflow.ReceiveChannel
	// watched is whether someone was waiting when the pass began.
	watched bool
	// waiting is what was asked of the generation while the pass waited on
	// a batch.
	waiting novaRequests
	// canceled is whether the pass's batches have been canceled.
	canceled bool
}

// hurried is whether someone opened the queue while the pass's reviews were
// out as batches.
func (p *novaPass) hurried() bool {
	return !p.watched && p.waiting.watched
}

// generateReviews submits up to novaPagesPerPass pages, then reviews each, or
// records it as its batch ends. It reports whether items were left for
// another pass.
func generateReviews(ctx workflow.Context, pass *novaPass) (bool, error) {
	var a *Activities
	submitCtx := workflow.WithActivityOptions(ctx, novaSubmitOptions)

	var after *billingqueueports.ReviewCursor
	submissions := make([]*billingqueueports.ReviewSubmission, 0, novaPagesPerPass)
	more := false
	for range novaPagesPerPass {
		var submission billingqueueports.ReviewSubmission
		if err := workflow.ExecuteActivity(submitCtx, a.SubmitNovaReviewsActivity, &SubmitNovaReviewsInput{
			Payload: pass.payload,
			After:   after,
			Inline:  pass.watched,
		}).Get(submitCtx, &submission); err != nil {
			return false, err
		}
		if submission.Count() > 0 {
			submissions = append(submissions, &submission)
			pass.progress.Total += submission.Count()
			publishNovaProgress(ctx, pass.payload, false)
		}
		after = submission.Next
		more = after != nil
		if !more {
			break
		}
	}

	for idx, submission := range submissions {
		reviewInline(ctx, pass.payload, submission.Inline, pass.progress)
		if submission.Handle == nil {
			continue
		}
		if err := pass.awaitBatch(ctx, submission, submissions[idx:]); err != nil {
			workflow.GetLogger(ctx).Warn("a billing review batch was not recorded", "error", err)
			pass.progress.Done += len(submission.Tickets)
			continue
		}
		applyCtx := workflow.WithActivityOptions(ctx, novaApplyOptions)
		if err := workflow.ExecuteActivity(applyCtx, a.ApplyNovaReviewsActivity, &NovaBatchInput{
			Payload:    pass.payload,
			Submission: submission,
		}).Get(applyCtx, nil); err != nil {
			workflow.GetLogger(ctx).Warn("a billing review batch could not be recorded", "error", err)
		}
		pass.progress.Done += len(submission.Tickets)
		publishNovaProgress(ctx, pass.payload, true)
	}

	return more, nil
}

// reviewInline reviews items novaInlineWave at a time, offers each item Nova
// agrees with to auto-approve as a batch's are, and tells the queue after each
// wave so the reviews appear as they are written.
func reviewInline(
	ctx workflow.Context,
	payload *billingqueuecontract.NovaGenerationPayload,
	items []pulid.ID,
	progress *billingqueueports.NovaGeneration,
) {
	for start := 0; start < len(items); start += novaInlineWave {
		wave := items[start:min(start+novaInlineWave, len(items))]
		agreed := reviewWave(ctx, payload, wave)
		progress.Done += len(wave)
		autoApproveWave(ctx, payload, agreed)
		publishNovaProgress(ctx, payload, true)
	}
}

func reviewWave(
	ctx workflow.Context,
	payload *billingqueuecontract.NovaGenerationPayload,
	wave []pulid.ID,
) []pulid.ID {
	var a *Activities
	reviewCtx := workflow.WithActivityOptions(ctx, agentReviewOptions)
	futures := make([]workflow.Future, 0, len(wave))
	for _, itemID := range wave {
		futures = append(futures, workflow.ExecuteActivity(reviewCtx, a.AgentReviewActivity,
			novaItemPayload(payload, itemID)))
	}

	agreed := make([]pulid.ID, 0, len(wave))
	for idx, future := range futures {
		var outcome billingqueueports.AgentReviewOutcome
		if err := future.Get(reviewCtx, &outcome); err != nil {
			workflow.GetLogger(ctx).Warn("billing queue agent review failed",
				"item", wave[idx].String(), "error", err)
			continue
		}
		if outcome.Verdict == billingqueue.AgentVerdictAgrees {
			agreed = append(agreed, wave[idx])
		}
	}

	return agreed
}

func autoApproveWave(
	ctx workflow.Context,
	payload *billingqueuecontract.NovaGenerationPayload,
	items []pulid.ID,
) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, recheckActivityOptions)
	futures := make([]workflow.Future, 0, len(items))
	for _, itemID := range items {
		futures = append(futures, workflow.ExecuteActivity(actx, a.AutoApproveActivity,
			novaItemPayload(payload, itemID)))
	}
	for idx, future := range futures {
		if err := future.Get(actx, nil); err != nil {
			workflow.GetLogger(ctx).Warn("billing queue auto-approve failed",
				"item", items[idx].String(), "error", err)
		}
	}
}

func novaItemPayload(
	payload *billingqueuecontract.NovaGenerationPayload,
	itemID pulid.ID,
) *billingqueuecontract.RecheckPayload {
	return &billingqueuecontract.RecheckPayload{
		OrganizationID: payload.OrganizationID,
		BusinessUnitID: payload.BusinessUnitID,
		ItemID:         itemID,
	}
}

// awaitBatch waits for a batch with a widening timer: batches usually end
// within minutes and the provider allows a day. Someone opening the queue
// while it waits cancels this batch and those after it (rest), which then end
// with what had already run.
func (p *novaPass) awaitBatch(
	ctx workflow.Context,
	submission *billingqueueports.ReviewSubmission,
	rest []*billingqueueports.ReviewSubmission,
) error {
	var a *Activities
	pollCtx := workflow.WithActivityOptions(ctx, novaPollOptions)
	deadline := workflow.Now(ctx).Add(novaBatchWaitLimit)
	wait := novaBatchFirstCheck
	if p.canceled {
		wait = novaCanceledCheck
	}
	for {
		if err := p.sleep(ctx, wait); err != nil {
			return err
		}
		if p.hurried() && !p.canceled {
			p.cancelBatches(ctx, rest)
			wait = novaCanceledCheck
			continue
		}
		var ended bool
		if err := workflow.ExecuteActivity(pollCtx, a.NovaBatchEndedActivity, &NovaBatchInput{
			Payload:    p.payload,
			Submission: submission,
		}).Get(pollCtx, &ended); err != nil {
			return err
		}
		if ended {
			return nil
		}
		if workflow.Now(ctx).After(deadline) {
			return errNovaBatchOverdue
		}
		wait = min(wait*2, novaBatchLongestCheck)
	}
}

// sleep waits d, ending early when someone opens the queue while the pass's
// reviews are out as batches.
func (p *novaPass) sleep(ctx workflow.Context, d time.Duration) error {
	if p.watched || p.canceled {
		return workflow.Sleep(ctx, d)
	}

	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	defer cancelTimer()
	timer := workflow.NewTimer(timerCtx, d)
	fired := false
	var err error
	for !fired && !p.hurried() {
		selector := workflow.NewSelector(ctx)
		selector.AddFuture(timer, func(f workflow.Future) {
			fired = true
			err = f.Get(timerCtx, nil)
		})
		selector.AddReceive(p.signals, func(c workflow.ReceiveChannel, _ bool) {
			var signal billingqueuecontract.NovaGenerationSignal
			c.Receive(ctx, &signal)
			p.waiting.note(signal)
		})
		selector.Select(ctx)
	}

	return err
}

func (p *novaPass) cancelBatches(ctx workflow.Context, submissions []*billingqueueports.ReviewSubmission) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, novaPollOptions)
	futures := make([]workflow.Future, 0, len(submissions))
	for _, submission := range submissions {
		if submission.Handle == nil {
			continue
		}
		futures = append(futures, workflow.ExecuteActivity(actx, a.CancelNovaReviewsActivity, &NovaBatchInput{
			Payload:    p.payload,
			Submission: submission,
		}))
	}
	for _, future := range futures {
		if err := future.Get(actx, nil); err != nil {
			workflow.GetLogger(ctx).Warn("a billing review batch could not be canceled", "error", err)
		}
	}
	p.canceled = true
}

func publishNovaProgress(
	ctx workflow.Context,
	payload *billingqueuecontract.NovaGenerationPayload,
	records bool,
) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, novaPublishOptions)
	if err := workflow.ExecuteActivity(actx, a.PublishNovaProgressActivity, &NovaProgressInput{
		Payload: payload,
		Records: records,
	}).Get(actx, nil); err != nil {
		workflow.GetLogger(ctx).Warn("billing queue Nova progress was not published", "error", err)
	}
}

// novaRequests is what the requests waiting on a generation ask of it.
type novaRequests struct {
	received bool
	// watched is whether any came from someone who opened the queue.
	watched bool
}

func (r *novaRequests) note(signal billingqueuecontract.NovaGenerationSignal) {
	r.received = true
	r.watched = r.watched || signal.Trigger == billingqueueports.NovaTriggerQueueOpened
}

func (r novaRequests) with(other novaRequests) novaRequests {
	return novaRequests{received: r.received || other.received, watched: r.watched || other.watched}
}

func drainNovaSignals(signals workflow.ReceiveChannel) novaRequests {
	var requests novaRequests
	for {
		var signal billingqueuecontract.NovaGenerationSignal
		if !signals.ReceiveAsync(&signal) {
			return requests
		}
		requests.note(signal)
	}
}

// BillingQueueNovaGenerationSweepWorkflow starts a generation for every
// organization whose queue has items Nova has not reviewed and can review.
func BillingQueueNovaGenerationSweepWorkflow(ctx workflow.Context) (int, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, recheckSweepOptions)

	var requested int
	err := workflow.ExecuteActivity(actx, a.RequestNovaGenerationsActivity).Get(actx, &requested)

	return requested, err
}
