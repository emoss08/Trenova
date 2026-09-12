package billingjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
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
			ID: "consolidated-invoice-run",
			Description: "Bill every statement period that has closed and not yet " +
				"been invoiced",
			// Hourly rather than daily because the cadence lives on each customer,
			// not here: the sweep asks every statement customer which of their
			// periods have closed, so checking often costs little and a missed
			// window is picked up on the next tick. At :15 rather than the top of
			// the hour, where the other schedules cluster.
			Spec:     schedule.Cron("15 * * * *").WithJitter(5 * time.Minute),
			Workflow: ConsolidatedInvoiceRunWorkflow,
			// SKIP, not BUFFER_ONE: a sweep that overruns an hour will overrun the
			// next one too, and buffering would queue a second attempt at the same
			// periods. Correctness does not rely on the policy anyway — the unique
			// index on a scheduled run's period, and the invoice back-link checked
			// under lock at commit, make a duplicate impossible.
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			Memo: map[string]any{
				"purpose": "consolidated-invoicing",
			},
		},
	}
}
