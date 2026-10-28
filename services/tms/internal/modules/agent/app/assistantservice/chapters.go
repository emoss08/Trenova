package assistantservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Service) chapterTitles(
	ctx context.Context,
	threadID pulid.ID,
	tenant pagination.TenantInfo,
) map[int]string {
	if s.chapters == nil {
		return nil
	}

	chapters, err := s.chapters.List(ctx, agentports.ListChaptersRequest{
		ThreadID:   threadID,
		TenantInfo: tenant,
	})
	if err != nil {
		s.logger.Warn("could not read a conversation's chapters",
			zap.String("thread", threadID.String()), zap.Error(err))

		return nil
	}

	titles := make(map[int]string, len(chapters))
	for _, chapter := range chapters {
		if chapter.StartSequence != nil {
			titles[int(*chapter.StartSequence)] = chapter.Title
		}
	}

	return titles
}

func snapToChapter(
	older []conversation.Message,
	through int,
	titles map[int]string,
) ([]conversation.Message, int) {
	if len(titles) == 0 || len(older) == 0 {
		return older, through
	}

	first := -1
	spoken := 0
	for idx := range older {
		if older[idx].Compacted() {
			continue
		}
		if first < 0 {
			first = older[idx].Sequence
		}
		spoken++
	}
	if first < 0 {
		return older, through
	}

	start := -1
	for sequence := range titles {
		if sequence > first && sequence <= through && sequence > start {
			start = sequence
		}
	}
	if start < 0 {
		return older, through
	}

	cut := len(older)
	for idx := range older {
		if !older[idx].Compacted() && older[idx].Sequence >= start {
			cut = idx

			break
		}
	}

	kept := 0
	for idx := range older[:cut] {
		if !older[idx].Compacted() {
			kept++
		}
	}
	if kept == 0 || kept*2 < spoken {
		return older, through
	}

	return older[:cut], older[cut-1].Sequence
}
