package settlementjobs

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
			ID:            "settlement-batch-generation",
			Description:   "Auto-generate driver settlement batches for closed pay periods",
			Spec:          schedule.Every(time.Hour),
			Workflow:      GenerateSettlementBatchesWorkflow,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "settlement-batch-generation",
			},
		},
		{
			ID:            "escrow-interest-accrual",
			Description:   "Accrue quarterly escrow interest per 49 CFR 376.12(k)",
			Spec:          schedule.Every(24 * time.Hour),
			Workflow:      AccrueEscrowInterestWorkflow,
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "escrow-interest-accrual",
			},
		},
	}
}
