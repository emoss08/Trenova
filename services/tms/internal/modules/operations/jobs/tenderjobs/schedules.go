package tenderjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
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
			ID:            "tender-stalled-sweep",
			Feature:       platformcatalog.FeatureCarrierBrokerage,
			Description:   "Nudge or recover live tenders whose offers are overdue",
			Spec:          schedule.Every(15 * time.Minute),
			Workflow:      TenderSweepWorkflow,
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "tender-stalled-sweep",
			},
		},
		{
			ID:            "rate-confirmation-issue-sweep",
			Feature:       platformcatalog.FeatureCarrierBrokerage,
			Description:   "Issue rate confirmations for accepted tenders whose paperwork never landed",
			Spec:          schedule.Every(time.Hour),
			Workflow:      RateConfirmationIssueSweepWorkflow,
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "rate-confirmation-issue-sweep",
			},
		},
		{
			ID:            "tender-token-retention",
			Feature:       platformcatalog.FeatureCarrierBrokerage,
			Description:   "Purge tender offer and rate confirmation tokens that are no longer usable",
			Spec:          schedule.Every(24 * time.Hour),
			Workflow:      PurgeTenderTokensWorkflow,
			TaskQueue:     temporaltype.TaskQueueDispatch.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "tender-token-retention",
			},
		},
	}
}
