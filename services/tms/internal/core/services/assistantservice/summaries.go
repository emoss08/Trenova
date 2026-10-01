package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) latestSummary(
	ctx context.Context,
	threadID pulid.ID,
	tenant pagination.TenantInfo,
) (*conversation.Summary, error) {
	if s.summaries == nil {
		return nil, nil
	}

	return s.summaries.Latest(ctx, repositories.ThreadSummaryRequest{
		ThreadID:   threadID,
		TenantInfo: tenant,
	})
}

func summaryThrough(summary *conversation.Summary) *int {
	if summary == nil {
		return nil
	}
	through := summary.ThroughSequence

	return &through
}

func historySummary(summary *conversation.Summary) *services.HistorySummary {
	if summary == nil {
		return nil
	}

	return &services.HistorySummary{
		Content:         summary.Content,
		Tainted:         summary.Tainted,
		ThroughSequence: summary.ThroughSequence,
	}
}

func (s *Service) pageSummaries(
	ctx context.Context,
	thread repositories.GetThreadRequest,
	messages []conversation.Message,
) ([]services.ThreadSummaryView, error) {
	if s.summaries == nil || len(messages) == 0 {
		return nil, nil
	}

	all, err := s.summaries.List(ctx, repositories.ListThreadSummariesRequest{
		ThreadID:   thread.ID,
		TenantInfo: thread.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	first := messages[0].Sequence
	last := messages[len(messages)-1].Sequence
	views := make([]services.ThreadSummaryView, 0, len(all))
	for _, summary := range all {
		if summary.ThroughSequence < first || summary.ThroughSequence > last {
			continue
		}
		views = append(views, services.ThreadSummaryViewOf(summary))
	}

	return views, nil
}
