package aicorrectionjobs

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

type ScheduleProvider struct {
	// extraction is ai.documentExtraction: the drift report compares extraction
	// accuracy, which only exists when a model reads documents.
	extraction bool
}

func NewScheduleProvider(cfg *aiconfig.Config) *ScheduleProvider {
	return &ScheduleProvider{extraction: cfg.DocumentExtractionEnabled()}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	schedules := []*schedule.Schedule{
		{
			ID:            "ai-correction-retention",
			Description:   "Purge AI corrections older than each organization's retention period",
			Spec:          schedule.Cron("40 2 * * *"),
			Workflow:      AICorrectionRetentionWorkflow,
			Args:          []any{&AICorrectionRetentionInput{}},
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "ai-correction-retention",
			},
		},
	}
	if !p.extraction {
		return schedules
	}

	return append(schedules,
		&schedule.Schedule{
			ID:            "extraction-accuracy-drift",
			Description:   "Tell each organization when an extraction provider read last week's documents worse than its own recent weeks",
			Spec:          schedule.Cron("20 6 * * 1"),
			Workflow:      ExtractionAccuracyDriftWorkflow,
			Args:          []any{&ExtractionAccuracyDriftInput{}},
			TaskQueue:     temporaltype.TaskQueueSystem.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": "extraction-accuracy-drift",
			},
		},
	)
}
