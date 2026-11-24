package bankreceiptcandidatejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/bankreceiptcandidatejobs/bankreceiptcandidatecontract"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	WakeScheduleID   = "bank-receipt-candidate-wake"
	SweepScheduleID  = "bank-receipt-candidate-sweep"
	wakeInterval     = time.Minute
	sweepInterval    = 24 * time.Hour
	drainDescription = "Find the open items each changed bank receipt could pay, a page at a time"
	wakeDescription  = "Wake the bank receipt candidate drains of tenants whose open items changed"
	sweepDescription = "Queue every unapplied bank receipt so its candidates are recomputed"
)

var DomainConfig = registry.DomainConfig{
	Name:         "bank-receipt-candidate-worker",
	TaskQueue:    temporaltype.TaskQueueBilling.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

var Workflows = []registry.WorkflowDefinition{
	{
		Name:        bankreceiptcandidatecontract.DrainWorkflowName,
		Fn:          BankReceiptCandidateDrainWorkflow,
		Description: drainDescription,
	},
	{
		Name:        bankreceiptcandidatecontract.WakeWorkflowName,
		Fn:          BankReceiptCandidateWakeWorkflow,
		Description: wakeDescription,
	},
	{
		Name:        bankreceiptcandidatecontract.SweepWorkflowName,
		Fn:          BankReceiptCandidateSweepWorkflow,
		Description: sweepDescription,
	},
}

type RegistryParams struct {
	fx.In

	Activities *Activities
	Logger     *zap.Logger
}

func NewRegistry(p RegistryParams) registry.WorkerRegistry {
	return registry.NewDomainRegistry(&DomainConfig, p.Activities, Workflows, p.Logger)
}

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            WakeScheduleID,
			Feature:       platformcatalog.FeatureAccounting,
			Description:   wakeDescription,
			Spec:          schedule.Every(wakeInterval),
			Workflow:      BankReceiptCandidateWakeWorkflow,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{"purpose": WakeScheduleID},
		},
		{
			ID:            SweepScheduleID,
			Feature:       platformcatalog.FeatureAccounting,
			Description:   sweepDescription,
			Spec:          schedule.Every(sweepInterval),
			Workflow:      BankReceiptCandidateSweepWorkflow,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{"purpose": SweepScheduleID},
		},
	}
}

var Module = fx.Module("bank-receipt-candidate-jobs",
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
