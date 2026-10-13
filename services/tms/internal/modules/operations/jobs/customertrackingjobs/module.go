package customertrackingjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/registry"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/operations/jobs/customertrackingjobs/customertrackingcontract"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	SweepScheduleID     = "customer-status-sweep"
	PurgeScheduleID     = "customer-tracking-link-retention"
	sweepInterval       = 10 * time.Minute
	purgeInterval       = 24 * time.Hour
	evaluateDescription = "Send the customer the status emails a shipment's changes owe it"
	sweepDescription    = "Catch status emails the per-shipment evaluations missed and retry pending ones"
	purgeDescription    = "Delete customer tracking links revoked or expired more than 30 days ago"
)

var DomainConfig = registry.DomainConfig{
	Name:         "customer-status-worker",
	TaskQueue:    temporaltype.TaskQueueDispatch.String(),
	WorkerConfig: registry.DefaultWorkerConfig(),
}

var Workflows = []registry.WorkflowDefinition{
	{
		Name:        customertrackingcontract.EvaluateWorkflowName,
		Fn:          EvaluateCustomerStatusWorkflow,
		Description: evaluateDescription,
	},
	{
		Name:        customertrackingcontract.SweepWorkflowName,
		Fn:          CustomerStatusSweepWorkflow,
		Description: sweepDescription,
	},
	{
		Name:        customertrackingcontract.PurgeWorkflowName,
		Fn:          CustomerTrackingLinkPurgeWorkflow,
		Description: purgeDescription,
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
			ID:            SweepScheduleID,
			Description:   sweepDescription,
			Spec:          schedule.Every(sweepInterval),
			Workflow:      CustomerStatusSweepWorkflow,
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{"purpose": SweepScheduleID},
		},
		{
			ID:            PurgeScheduleID,
			Description:   purgeDescription,
			Spec:          schedule.Every(purgeInterval),
			Workflow:      CustomerTrackingLinkPurgeWorkflow,
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo:          map[string]any{"purpose": PurgeScheduleID},
		},
	}
}

var Module = fx.Module("customer-status-jobs",
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
