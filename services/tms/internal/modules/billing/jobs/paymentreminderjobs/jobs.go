// Package paymentreminderjobs sends the day's scheduled payment reminders for
// every organization that has them on.
package paymentreminderjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/billing/app/paymentreminderservice"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/receivablenotice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
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
	SweepWorkflowName = "PaymentReminderSweepWorkflow"
	eventNoRecipients = "payment_reminder_no_recipients"
)

var Module = fx.Module("payment-reminder-jobs",
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
	Name:         "payment-reminder-worker",
	TaskQueue:    temporaltype.TaskQueueBilling.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

type ActivitiesParams struct {
	fx.In

	Controls      tenantports.BillingControlRepository
	Reminders     *paymentreminderservice.Service
	Notifications *notificationservice.Service `optional:"true"`
	Logger        *zap.Logger
}

type Activities struct {
	controls  tenantports.BillingControlRepository
	reminders paymentreminderports.Service
	notifier  *notificationservice.Service
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		controls:  p.Controls,
		reminders: p.Reminders,
		notifier:  p.Notifications,
		l:         p.Logger.Named("payment-reminder-jobs"),
	}
}

// SweepResult is what one day's reminders did across organizations.
type SweepResult struct {
	Organizations int `json:"organizations"`
	Sent          int `json:"sent"`
	Failed        int `json:"failed"`
}

// PaymentReminderSweepActivity gives every organization with reminders on its turn. One
// organization's failure is logged and the sweep moves on.
func (a *Activities) PaymentReminderSweepActivity(ctx context.Context) (*SweepResult, error) {
	controls, err := a.controls.ListPaymentReminders(ctx)
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
		sent, sendErr := a.reminders.SendDue(tenantCtx, control, now)
		if sendErr != nil {
			result.Failed++
			a.l.Error("payment reminders failed for an organization",
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
			Title:     "Payment reminders could not be sent",
			Reason: "have overdue invoices but no invoice email recipient. " +
				"Add one to each customer's email profile so reminders reach them.",
			Source: "paymentreminderjobs",
			Link:   "/accounting/ar/collections",
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

// PaymentReminderSweepWorkflow sends one day's scheduled reminders.
func PaymentReminderSweepWorkflow(ctx workflow.Context) (*SweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, sweepActivityOptions)

	var a *Activities
	var result *SweepResult
	if err := workflow.ExecuteActivity(ctx, a.PaymentReminderSweepActivity).Get(ctx, &result); err != nil {
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
		Fn:          PaymentReminderSweepWorkflow,
		Description: "Email customers the payment reminders their schedule says are due",
	}}, p.Logger)
}

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{{
		ID:          "payment-reminder-sweep",
		Description: "Email customers the payment reminders their schedule says are due",
		// Mid-morning on the US east coast, after the overnight late-charge run
		// and credit sweep, so a reminder lists the charges they raised.
		Spec:          schedule.Cron("5 14 * * *").WithJitter(10 * time.Minute),
		Workflow:      PaymentReminderSweepWorkflow,
		OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		TaskQueue:     temporaltype.TaskQueueBilling.String(),
		Memo: map[string]any{
			"purpose": "payment-reminder-sweep",
		},
	}}
}
