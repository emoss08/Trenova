package billingtransferjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:          "billing-transfer-automatic",
			Description: "Move ready freight into billing for organizations that transfer automatically",
			// Continuous is the tightest cadence billing control offers, so the
			// sweep runs that often; hourly, daily and weekly organizations are
			// held to their own interval by the run they last started.
			Spec:     schedule.Cron("*/5 * * * *").WithJitter(time.Minute),
			Workflow: AutomaticBillingTransferSweepWorkflow,
			// SKIP: an overrunning sweep is already starting the transfers that
			// are due, and a second would only find them active.
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			Memo: map[string]any{
				"purpose": "billing-transfer-automatic",
			},
		},
		{
			ID: "billing-transfer-zombie-reconcile",
			Description: "Close out billing transfer runs whose workflow is no " +
				"longer running",
			// The workflow finalizes itself on every path it can reach, so this
			// only ever catches a worker that died between batches. Hourly is
			// frequent enough that a biller is not left staring at a progress bar
			// that has stopped, and the sweep costs one indexed query when there
			// is nothing to do.
			Spec:     schedule.Cron("40 * * * *").WithJitter(5 * time.Minute),
			Workflow: ReconcileZombieBillingTransferRunsWorkflow,
			// SKIP: a sweep that overran would look at the same stalled runs
			// again, and finalizing is idempotent anyway.
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			Memo: map[string]any{
				"purpose": "billing-transfer-reconciliation",
			},
		},
	}
}
