package temporaljobs

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

// A tenant drain works one tenant's queued subjects of one kind
// (workqueueports.Queue) from a single debounced workflow, however many
// subjects change: a burst of a thousand shipment updates is one workflow and
// a few activities, not a workflow per shipment. The rows are the requests;
// a signal only wakes the drain, so it carries nothing.

// DrainWorkflowID is the one drain of a kind for a tenant.
func DrainWorkflowID(kind string, tenant pagination.TenantInfo) string {
	return "drain-" + kind + "-" + tenant.OrgID.String() + "-" + tenant.BuID.String()
}

// DrainPayload carries a tenant drain across continue-as-new. Kind is the
// queue it drains, for a drain whose workflow serves a family of kinds.
type DrainPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Kind           string   `json:"kind,omitempty"`
	Iterations     int      `json:"iterations"`
}

func (p *DrainPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

// ErrDeferred is what a handler returns for a subject it chose not to work
// yet, such as one past a concurrency cap: the subject is due again after
// RetryAfter and the drain stays alive to come back for it.
var ErrDeferred = errors.New("deferred until there is room")

// DrainPage is what one page of a drain did. Blocked says subjects were
// deferred, so the drain must come back for them rather than end.
type DrainPage struct {
	Done     int  `json:"done"`
	Failed   int  `json:"failed"`
	Deferred int  `json:"deferred"`
	Full     bool `json:"full"`
	Blocked  bool `json:"blocked"`
}

// DrainTotals adds up a drain's pages.
type DrainTotals struct {
	Done   int `json:"done"`
	Failed int `json:"failed"`
}

type TenantDrainOptions struct {
	SignalName string
	// Debounce gathers a burst of requests into one round.
	Debounce time.Duration
	// PagesPerRound bounds the history one round writes; a backlog past it
	// runs the next round at once rather than after the debounce.
	PagesPerRound int
	// IterationsPerRun is how many rounds run before the drain continues as
	// new; Iterations is how many this execution started with.
	IterationsPerRun int
	Iterations       int
	// Page works one page of due subjects in an activity.
	Page func(ctx workflow.Context) (*DrainPage, error)
}

// RunTenantDrain runs rounds until nothing is due, nothing was deferred and
// no request arrived, and reports whether the caller should continue as new.
func RunTenantDrain(
	ctx workflow.Context,
	opts TenantDrainOptions,
) (*DrainTotals, bool, error) {
	signals := workflow.GetSignalChannel(ctx, opts.SignalName)
	totals := &DrainTotals{}
	backlog := false

	for iteration := opts.Iterations; ; iteration++ {
		if iteration >= opts.IterationsPerRun {
			return totals, true, nil
		}
		if !backlog {
			if err := workflow.Sleep(ctx, opts.Debounce); err != nil {
				return totals, false, err
			}
		}
		drainWakes(signals)

		blocked := false
		for range NormalizeLimit(opts.PagesPerRound, 1) {
			page, err := opts.Page(ctx)
			if err != nil {
				return totals, false, err
			}
			totals.Done += page.Done
			totals.Failed += page.Failed
			blocked = blocked || page.Blocked
			backlog = page.Full
			if !page.Full {
				break
			}
		}

		if !drainWakes(signals) && !backlog && !blocked {
			return totals, false, nil
		}
	}
}

func drainWakes(signals workflow.ReceiveChannel) bool {
	received := false
	for signals.ReceiveAsync(nil) {
		received = true
	}

	return received
}

// DrainDueParams is one page of a tenant drain's work, in an activity.
type DrainDueParams struct {
	Queue      workqueueports.Queue
	Kind       string
	Tenant     pagination.TenantInfo
	PageSize   int
	RetryAfter time.Duration
	// Handle works the page's subjects and returns the ones that failed or
	// were deferred (ErrDeferred); they are due again after RetryAfter, the
	// rest are settled. details holds each subject's queued detail.
	Handle func(ctx context.Context, subjects []pulid.ID, details map[pulid.ID]string) map[pulid.ID]error
}

// DrainDue claims a page of the tenant's due subjects, works them and settles
// each, so a retry after a partial page picks up what is left.
func DrainDue(ctx context.Context, p *DrainDueParams) (*DrainPage, error) {
	now := timeutils.NowUnix()
	due, err := p.Queue.ListDue(ctx, &workqueueports.ListDueRequest{
		Kind:       p.Kind,
		TenantInfo: p.Tenant,
		Now:        now,
		Limit:      p.PageSize,
	})
	if err != nil || len(due) == 0 {
		return &DrainPage{}, err
	}

	subjects := make([]pulid.ID, 0, len(due))
	details := make(map[pulid.ID]string, len(due))
	for _, item := range due {
		subjects = append(subjects, item.SubjectID)
		if item.Detail != "" {
			details[item.SubjectID] = item.Detail
		}
	}
	if activity.IsActivity(ctx) {
		activity.RecordHeartbeat(ctx, len(subjects))
	}
	failures := p.Handle(ctx, subjects, details)

	page := &DrainPage{Full: len(due) == p.PageSize}
	settle := &workqueueports.SettleManyRequest{
		Kind:       p.Kind,
		TenantInfo: p.Tenant,
		RetryAt:    now + int64(p.RetryAfter.Seconds()),
	}
	for _, item := range due {
		failure, failed := failures[item.SubjectID]
		switch {
		case failed && errors.Is(failure, ErrDeferred):
			settle.Retry = append(settle.Retry, item)
			page.Deferred++
			page.Blocked = true
		case failed:
			settle.Retry = append(settle.Retry, item)
			page.Failed++
		default:
			settle.Done = append(settle.Done, item)
			page.Done++
		}
	}
	if err = p.Queue.SettleMany(ctx, settle); err != nil {
		return nil, err
	}

	return page, nil
}

// DrainRequester queues subjects for a kind's tenant drain and wakes it. The
// row is written in the caller's transaction; the wake goes after commit, and
// a lost wake costs only the backstop sweep's delay.
type DrainRequester struct {
	Queue     workqueueports.Queue
	Signals   kernelports.WorkflowSignalStarter
	Kind      string
	Workflow  string
	TaskQueue string
	Signal    string
	Summary   string
	Logger    *zap.Logger
}

func (r *DrainRequester) Request(
	ctx context.Context,
	tenant pagination.TenantInfo,
	subjectIDs ...pulid.ID,
) error {
	return r.RequestWithDetail(ctx, tenant, "", subjectIDs...)
}

// RequestWithDetail queues the subjects with what their work needs beyond
// their ids, replacing any detail a subject was queued with before.
func (r *DrainRequester) RequestWithDetail(
	ctx context.Context,
	tenant pagination.TenantInfo,
	detail string,
	subjectIDs ...pulid.ID,
) error {
	if tenant.OrgID.IsNil() || tenant.BuID.IsNil() || len(subjectIDs) == 0 {
		return nil
	}
	if err := r.Queue.Enqueue(ctx, &workqueueports.EnqueueRequest{
		Kind:       r.Kind,
		TenantInfo: tenant,
		SubjectIDs: subjectIDs,
		At:         timeutils.NowUnix(),
		Detail:     detail,
	}); err != nil {
		return err
	}

	kernelports.AfterCommit(ctx, func(committed context.Context) {
		if err := r.Wake(committed, tenant); err != nil {
			r.Logger.Warn("could not wake a tenant drain; the sweep will",
				zap.String("kind", r.Kind),
				zap.String("organizationId", tenant.OrgID.String()),
				zap.Error(err))
		}
	})

	return nil
}

// Wake starts the tenant's drain, or tells the running one to look again.
func (r *DrainRequester) Wake(ctx context.Context, tenant pagination.TenantInfo) error {
	id := DrainWorkflowID(r.Kind, tenant)
	_, err := r.Signals.SignalWithStartWorkflow(
		ctx,
		id,
		r.Signal,
		nil,
		client.StartWorkflowOptions{
			ID:            id,
			TaskQueue:     r.TaskQueue,
			StaticSummary: r.Summary,
			Priority:      temporal.Priority{FairnessKey: tenant.OrgID.String()},
		},
		r.Workflow,
		&DrainPayload{OrganizationID: tenant.OrgID, BusinessUnitID: tenant.BuID, Kind: r.Kind},
	)

	return err
}

// WakeDue wakes every tenant with due work of the kind: the backstop a sweep
// runs for wakes that were lost.
func (r *DrainRequester) WakeDue(ctx context.Context, limit int) (int, error) {
	tenants, err := r.Queue.ListDueTenants(ctx, &workqueueports.ListDueTenantsRequest{
		Kind:  r.Kind,
		Now:   timeutils.NowUnix(),
		Limit: limit,
	})
	if err != nil {
		return 0, err
	}

	woken := 0
	for _, tenant := range tenants {
		if wErr := r.Wake(ctx, tenant); wErr != nil {
			r.Logger.Warn("could not wake a tenant drain from the sweep",
				zap.String("kind", r.Kind),
				zap.String("organizationId", tenant.OrgID.String()),
				zap.Error(wErr))
			continue
		}
		woken++
	}

	return woken, nil
}
