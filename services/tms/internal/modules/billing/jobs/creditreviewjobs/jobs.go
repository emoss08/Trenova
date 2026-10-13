// Package creditreviewjobs runs the nightly credit sweep: every customer's open
// receivables and credit standing reviewed, so a limit changed on the form or a
// review that failed after a posting does not leave a customer held or
// unheld for longer than a day.
package creditreviewjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/billing/app/creditreviewservice"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const SweepWorkflowName = "CustomerCreditSweepWorkflow"

var Module = fx.Module("credit-review-jobs",
	fx.Provide(NewActivities),
	fx.Provide(schedule.AsProvider(NewScheduleProvider)),
	fx.Provide(
		fx.Annotate(
			NewRegistry,
			fx.As(new(registry.WorkerRegistry)),
			fx.ResultTags(`group:"worker_registries"`),
		),
	),
)

// DomainConfig shares the billing task queue, so its WorkerConfig matches the
// other billing registries.
var DomainConfig = registry.DomainConfig{
	Name:         "credit-review-worker",
	TaskQueue:    temporaltype.TaskQueueBilling.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

type ActivitiesParams struct {
	fx.In

	Reviews *creditreviewservice.Service
	Logger  *zap.Logger
}

type Activities struct {
	reviews *creditreviewservice.Service
	l       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{reviews: p.Reviews, l: p.Logger.Named("credit-review-jobs")}
}

// SweepResult is what one nightly sweep reviewed.
type SweepResult struct {
	Customers int `json:"customers"`
	Failed    int `json:"failed"`
}

// CreditReviewSweepActivity reviews every customer's credit. Organizations that fail are
// logged and skipped so one cannot hold up the rest.
func (a *Activities) CreditReviewSweepActivity(ctx context.Context) (*SweepResult, error) {
	result, err := a.reviews.Sweep(ctx, func(detail string) {
		activity.RecordHeartbeat(ctx, detail)
	})
	if err != nil {
		return nil, err
	}
	if result.Failed > 0 {
		a.l.Warn("credit sweep finished with failures",
			zap.Int("customers", result.Customers), zap.Int("failed", result.Failed))
	}
	return &SweepResult{Customers: result.Customers, Failed: result.Failed}, nil
}

var sweepActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    30 * time.Second,
		BackoffCoefficient: 2,
		MaximumAttempts:    3,
		MaximumInterval:    5 * time.Minute,
	},
}

// CustomerCreditSweepWorkflow reviews every customer's credit once.
func CustomerCreditSweepWorkflow(ctx workflow.Context) (*SweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, sweepActivityOptions)

	var a *Activities
	var result *SweepResult
	if err := workflow.ExecuteActivity(ctx, a.CreditReviewSweepActivity).Get(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{{
		Name:        SweepWorkflowName,
		Fn:          CustomerCreditSweepWorkflow,
		TaskQueue:   temporaltype.TaskQueueBilling.String(),
		Description: "Review every customer's open receivables against its credit limit",
	}}
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	workflows := RegisterWorkflows()
	definitions := make([]registry.WorkflowDefinition, len(workflows))
	for i, wf := range workflows {
		definitions[i] = registry.WorkflowDefinition{
			Name:        wf.Name,
			Fn:          wf.Fn,
			Description: wf.Description,
		}
	}
	return registry.NewDomainRegistry(&DomainConfig, p.Activities, definitions, p.Logger)
}

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{{
		ID:          "customer-credit-sweep",
		Description: "Review every customer's open receivables against its credit limit",
		// After the 02:30 late-charge assessment, so charges it raised count
		// toward the balance the sweep reviews.
		Spec:          schedule.Cron("15 3 * * *").WithJitter(10 * time.Minute),
		Workflow:      CustomerCreditSweepWorkflow,
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		TaskQueue:     temporaltype.TaskQueueBilling.String(),
		Memo: map[string]any{
			"purpose": "customer-credit-sweep",
		},
	}}
}
