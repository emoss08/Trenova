// Package customerstatementjobs emails each month's customer statements for
// every organization that has monthly statements on, once its statement day
// has come.
package customerstatementjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/billing/app/customerstatementservice"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/receivablenotice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	SweepWorkflowName = "CustomerStatementSweepWorkflow"
	eventNoRecipients = "customer_statement_no_recipients"
)

var Module = fx.Module("customer-statement-jobs",
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
	Name:         "customer-statement-worker",
	TaskQueue:    temporaltype.TaskQueueBilling.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

type ActivitiesParams struct {
	fx.In

	Controls      tenantports.BillingControlRepository
	Statements    *customerstatementservice.Service
	Notifications *notificationservice.Service `optional:"true"`
	Logger        *zap.Logger
}

type Activities struct {
	controls   tenantports.BillingControlRepository
	statements customerstatementports.Service
	notifier   *notificationservice.Service
	l          *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		controls:   p.Controls,
		statements: p.Statements,
		notifier:   p.Notifications,
		l:          p.Logger.Named("customer-statement-jobs"),
	}
}

// SweepResult is what one day's statements did across organizations.
type SweepResult struct {
	Organizations int `json:"organizations"`
	Sent          int `json:"sent"`
	Failed        int `json:"failed"`
}

// SweepActivity gives every organization with monthly statements on its
// turn. One organization's failure is logged and the sweep moves on.
func (a *Activities) SweepActivity(ctx context.Context) (*SweepResult, error) {
	controls, err := a.controls.ListMonthlyStatements(ctx)
	if err != nil {
		return nil, err
	}
	result := new(SweepResult)
	now := timeutils.NowUnix()
	for _, control := range controls {
		activity.RecordHeartbeat(ctx, control.OrganizationID.String())
		result.Organizations++
		tenantCtx := dbscope.WithTenant(ctx, dbscope.Tenant{
			OrganizationID: control.OrganizationID,
			BusinessUnitID: control.BusinessUnitID,
		})
		sent, sendErr := a.statements.SendMonthly(tenantCtx, control, now)
		if sendErr != nil {
			result.Failed++
			a.l.Error("monthly statements failed for an organization",
				zap.String("organizationId", control.OrganizationID.String()), zap.Error(sendErr))
			continue
		}
		result.Sent += sent.Sent
		result.Failed += sent.Failed
		receivablenotice.Report(tenantCtx, a.notifier, a.l, &receivablenotice.Unreachable{
			Tenant: pagination.TenantInfo{
				OrgID: control.OrganizationID,
				BuID:  control.BusinessUnitID,
			},
			Customers: sent.NoRecipients,
			Now:       now,
			EventType: eventNoRecipients,
			Title:     "Monthly statements could not be sent",
			Reason: "have a statement this month but no invoice email recipient. " +
				"Add one to each customer's email profile so statements reach them.",
			Source: "customerstatementjobs",
			Link:   "/accounting/ar/aging",
		})
	}
	return result, nil
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

// CustomerStatementSweepWorkflow sends the monthly statements that are due.
func CustomerStatementSweepWorkflow(ctx workflow.Context) (*SweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, sweepActivityOptions)

	var a *Activities
	var result *SweepResult
	if err := workflow.ExecuteActivity(ctx, a.SweepActivity).Get(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(&DomainConfig, p.Activities, []registry.WorkflowDefinition{{
		Name:        SweepWorkflowName,
		Fn:          CustomerStatementSweepWorkflow,
		Description: "Email customers the statement of the month just closed",
	}}, p.Logger)
}

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{{
		ID:          "customer-statement-sweep",
		Description: "Email customers the statement of the month just closed",
		// Daily, so an organization's chosen day is never missed; the month is
		// sent once. Mid-morning on the US east coast, after the overnight
		// late-charge run, so a statement carries the charges it raised.
		Spec:          schedule.Cron("35 14 * * *").WithJitter(10 * time.Minute),
		Workflow:      CustomerStatementSweepWorkflow,
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		TaskQueue:     temporaltype.TaskQueueBilling.String(),
		Memo: map[string]any{
			"purpose": "customer-statement-sweep",
		},
	}}
}
