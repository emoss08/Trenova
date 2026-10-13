package documentintelligencejobs

import (
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/api/enums/v1"
)

type ScheduleProvider struct {
	// extraction is ai.documentExtraction: without it no model reads documents,
	// so there is nothing for the extraction poller to collect.
	extraction bool
}

func NewScheduleProvider(cfg *config.Config) *ScheduleProvider {
	return &ScheduleProvider{extraction: cfg.AI.DocumentExtractionEnabled()}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	schedules := []*schedule.Schedule{
		{
			ID:            "document-intelligence-reconciliation",
			Description:   "Reconcile stale document intelligence jobs",
			Spec:          schedule.Every(2 * time.Hour),
			Workflow:      ReconcileDocumentIntelligenceWorkflow,
			TaskQueue:     temporaltype.DocumentIntelligenceTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		},
	}
	if !p.extraction {
		return schedules
	}

	return append(schedules,
		&schedule.Schedule{
			ID:            "document-ai-extraction-poller",
			Description:   "Poll pending OpenAI background extraction jobs",
			Spec:          schedule.Every(5 * time.Minute),
			Workflow:      PollPendingDocumentAIExtractionsWorkflow,
			TaskQueue:     temporaltype.DocumentIntelligenceTaskQueue,
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Args: []any{
				&PollPendingDocumentAIExtractionsPayload{
					Limit: 25,
				},
			},
		},
	)
}
