package retrievaljobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/airetrieval"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/retrievalservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/internal/core/temporaljobs/modelcall"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	StepsPerRun             = 200
	ReindexPagesPerRun      = 50
	IdleWait                = 5 * time.Minute
	organizationConcurrency = 4
	maxPurgeCalls           = 50
)

var retryPolicy = &temporal.RetryPolicy{
	InitialInterval:    5 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    time.Minute,
	MaximumAttempts:    3,
}

var (
	quickOptions = workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy:         retryPolicy,
	}
	longOptions = workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		HeartbeatTimeout:    modelcall.HeartbeatTimeout,
		RetryPolicy:         retryPolicy,
	}
)

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        retrievalservice.IndexOrganizationWorkflowName,
			Fn:          IndexOrganizationWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Embed an organization's stale retrieval sources until the outbox drains",
		},
		{
			Name:        retrievalservice.ReindexSourceWorkflowName,
			Fn:          ReindexRetrievalSourceWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Mark every source of one type stale so the indexer re-checks it",
		},
		{
			Name:        retrievalservice.SweepWorkflowName,
			Fn:          RetrievalIndexSweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Find retrieval sources changed since they were indexed, per organization",
		},
	}
}

func withOptions(
	ctx workflow.Context,
	options *workflow.ActivityOptions,
	item temporaljobs.TenantWorkItem,
) workflow.Context {
	scoped := *options
	scoped.Priority = retrievalservice.WorkflowPriorityFor(item.OrganizationID)

	return workflow.WithActivityOptions(ctx, scoped)
}

type indexRun struct {
	ctx     workflow.Context
	item    temporaljobs.TenantWorkItem
	signals workflow.ReceiveChannel
	result  *IndexOrganizationResult
	steps   int
}

func IndexOrganizationWorkflow(
	ctx workflow.Context,
	input retrievalservice.IndexOrganizationInput,
) (*IndexOrganizationResult, error) {
	run := &indexRun{
		ctx: ctx,
		item: temporaljobs.TenantWorkItem{
			OrganizationID: input.OrganizationID,
			BusinessUnitID: input.BusinessUnitID,
		},
		signals: workflow.GetSignalChannel(ctx, retrievalservice.IndexSignalName),
		result:  &IndexOrganizationResult{CostUSD: decimal.Zero},
	}

	idle := false
	for {
		run.drainSignals()
		if run.steps >= StepsPerRun {
			return run.result, workflow.NewContinueAsNewError(ctx, IndexOrganizationWorkflow,
				retrievalservice.IndexOrganizationInput{
					OrganizationID: input.OrganizationID,
					BusinessUnitID: input.BusinessUnitID,
					Batches:        input.Batches + run.result.Batches,
				})
		}

		plan, err := run.plan()
		if err != nil {
			return run.result, err
		}
		if !plan.Active {
			run.result.Stopped = plan.Reason
			return run.result, nil
		}

		worked, stopped, err := run.cycle(plan)
		if err != nil || stopped {
			return run.result, err
		}
		if worked {
			idle = false
			continue
		}

		if run.waitForSignal() {
			idle = false
			continue
		}
		if idle {
			return run.result, nil
		}
		idle = true
	}
}

func (r *indexRun) drainSignals() {
	for {
		var signal retrievalservice.IndexSignal
		if !r.signals.ReceiveAsync(&signal) {
			return
		}
	}
}

func (r *indexRun) waitForSignal() bool {
	timerCtx, cancel := workflow.WithCancel(r.ctx)
	defer cancel()

	received := false
	selector := workflow.NewSelector(r.ctx)
	selector.AddReceive(r.signals, func(channel workflow.ReceiveChannel, _ bool) {
		var signal retrievalservice.IndexSignal
		channel.Receive(r.ctx, &signal)
		received = true
	})
	selector.AddFuture(workflow.NewTimer(timerCtx, IdleWait), func(workflow.Future) {})
	selector.Select(r.ctx)

	return received
}

func (r *indexRun) plan() (*serviceports.RetrievalIndexPlan, error) {
	r.steps++
	actx := withOptions(r.ctx, &quickOptions, r.item)

	var a *Activities
	var plan serviceports.RetrievalIndexPlan
	err := workflow.ExecuteActivity(actx, a.PlanRetrievalIndexActivity, &PlanInput{
		TenantWorkItem: r.item,
	}).Get(actx, &plan)

	return &plan, err
}

func (r *indexRun) cycle(
	plan *serviceports.RetrievalIndexPlan,
) (more, stopped bool, cycleErr error) {
	for _, key := range plan.RetiredKeys {
		if err := r.purge(key); err != nil {
			return false, false, err
		}
	}

	worked := false
	for _, key := range plan.ModelKeys() {
		batch, err := r.indexBatch(key, plan.SourceTypes)
		if err != nil {
			return false, false, err
		}
		if batch.BudgetReached {
			r.result.Stopped = airetrieval.UnavailableReasonBudgetPaused
			return false, true, nil
		}
		worked = worked || !batch.Drained()
	}
	if worked {
		return true, false, nil
	}

	if plan.PendingModelKey != "" {
		swapped, err := r.completeModelChange(plan.PendingModelKey)
		if err != nil || swapped {
			return swapped, false, err
		}
	}

	sweep, err := r.sweep()
	if err != nil {
		return false, false, err
	}

	return sweep.Marked > 0, false, nil
}

func (r *indexRun) indexBatch(
	modelKey string,
	sourceTypes []airetrieval.SourceType,
) (*serviceports.RetrievalIndexBatchResult, error) {
	r.steps++
	actx := withOptions(r.ctx, &longOptions, r.item)

	var a *Activities
	var batch serviceports.RetrievalIndexBatchResult
	if err := workflow.ExecuteActivity(actx, a.IndexRetrievalBatchActivity,
		&serviceports.RetrievalIndexBatchRequest{
			TenantInfo:  r.item.TenantInfo(),
			ModelKey:    modelKey,
			SourceTypes: sourceTypes,
			Limit:       retrievalservice.DefaultBatchSize,
		}).Get(actx, &batch); err != nil {
		return nil, err
	}
	if !batch.Drained() {
		r.result.absorb(&batch)
	}

	return &batch, nil
}

func (r *indexRun) completeModelChange(pendingModelKey string) (bool, error) {
	r.steps++
	actx := withOptions(r.ctx, &quickOptions, r.item)

	var a *Activities
	var change serviceports.RetrievalModelChangeResult
	if err := workflow.ExecuteActivity(actx, a.CompleteRetrievalModelChangeActivity,
		&ModelChangeInput{TenantWorkItem: r.item, PendingModelKey: pendingModelKey}).
		Get(actx, &change); err != nil {
		return false, err
	}
	if !change.Swapped {
		return false, nil
	}

	r.result.Swapped = true
	if change.RetiredModelKey != "" {
		if err := r.purge(change.RetiredModelKey); err != nil {
			return true, err
		}
	}

	return true, nil
}

func (r *indexRun) purge(modelKey string) error {
	var a *Activities
	for range maxPurgeCalls {
		r.steps++
		actx := withOptions(r.ctx, &longOptions, r.item)

		var purged serviceports.RetrievalPurgeResult
		if err := workflow.ExecuteActivity(actx, a.PurgeRetrievalModelActivity,
			&PurgeInput{TenantWorkItem: r.item, ModelKey: modelKey}).
			Get(actx, &purged); err != nil {
			return err
		}
		r.result.Purged += purged.Embeddings + purged.IndexEntries
		if purged.Done {
			return nil
		}
	}

	return nil
}

func (r *indexRun) sweep() (*serviceports.RetrievalSweepResult, error) {
	r.steps++
	actx := withOptions(r.ctx, &longOptions, r.item)

	var a *Activities
	var sweep serviceports.RetrievalSweepResult
	err := workflow.ExecuteActivity(actx, a.SweepRetrievalSourcesActivity,
		&OrganizationSweepInput{TenantWorkItem: r.item}).Get(actx, &sweep)

	return &sweep, err
}

func ReindexRetrievalSourceWorkflow(
	ctx workflow.Context,
	input retrievalservice.ReindexSourceInput,
) (*ReindexResult, error) {
	item := temporaljobs.TenantWorkItem{
		OrganizationID: input.OrganizationID,
		BusinessUnitID: input.BusinessUnitID,
	}
	actx := withOptions(ctx, &quickOptions, item)
	result := &ReindexResult{Marked: input.Marked}

	var a *Activities
	after := input.AfterID
	for range ReindexPagesPerRun {
		var page serviceports.RetrievalReindexPage
		if err := workflow.ExecuteActivity(actx, a.ReindexRetrievalPageActivity,
			&serviceports.RetrievalReindexPageRequest{
				TenantInfo: item.TenantInfo(),
				SourceType: input.SourceType,
				AfterID:    after,
			}).Get(actx, &page); err != nil {
			return result, err
		}

		result.Marked += page.Marked
		if page.Done {
			result.Done = true
			return result, nil
		}
		after = page.Next
	}

	return result, workflow.NewContinueAsNewError(ctx, ReindexRetrievalSourceWorkflow,
		retrievalservice.ReindexSourceInput{
			OrganizationID: input.OrganizationID,
			BusinessUnitID: input.BusinessUnitID,
			SourceType:     input.SourceType,
			AfterID:        after,
			Marked:         result.Marked,
		})
}

func RetrievalIndexSweepWorkflow(ctx workflow.Context, input *SweepInput) (*SweepResult, error) {
	if input == nil {
		input = &SweepInput{}
	}

	var a *Activities
	result := &SweepResult{FailedOrganizations: make([]string, 0)}
	_, next, err := temporaljobs.RunTenantFanOut(ctx, temporaljobs.FanOutOptions{
		Label:       "Retrieval index sweep",
		Concurrency: organizationConcurrency,
		ListPage: func(
			wctx workflow.Context,
			after *temporaljobs.TenantWorkItem,
		) (*temporaljobs.TenantPage, error) {
			if after == nil {
				after = input.After
			}
			listCtx := workflow.WithActivityOptions(wctx, quickOptions)
			var page temporaljobs.TenantPage
			err := workflow.ExecuteActivity(listCtx, a.ListRetrievalOrganizationsActivity,
				&ListOrganizationsInput{
					After: after,
					Limit: temporaljobs.DefaultOrganizationPageSize,
				}).Get(listCtx, &page)

			return &page, err
		},
		RunTenant: func(wctx workflow.Context, tenant temporaljobs.TenantWorkItem) (int, error) {
			actx := withOptions(wctx, &longOptions, tenant)
			var swept serviceports.RetrievalSweepResult
			if err := workflow.ExecuteActivity(actx, a.SweepRetrievalSourcesActivity,
				&OrganizationSweepInput{TenantWorkItem: tenant, Wake: true}).
				Get(actx, &swept); err != nil {
				result.FailedOrganizations = append(
					result.FailedOrganizations,
					tenant.OrganizationID.String(),
				)
				return 0, err
			}
			result.OrganizationsSwept++
			result.Marked += swept.Marked

			return swept.Marked, nil
		},
	})
	if err != nil {
		return result, err
	}
	if next != nil {
		return result, workflow.NewContinueAsNewError(ctx, RetrievalIndexSweepWorkflow,
			&SweepInput{After: next})
	}

	return result, nil
}
