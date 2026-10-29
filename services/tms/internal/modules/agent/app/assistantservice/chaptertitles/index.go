package chaptertitles

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantchapterports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Index) Titles(
	ctx context.Context,
	threadID pulid.ID,
	tenant pagination.TenantInfo,
) map[int]string {
	if s.Chapters == nil {
		return nil
	}

	chapters, err := s.Chapters.List(ctx, assistantchapterports.ListChaptersRequest{
		ThreadID:   threadID,
		TenantInfo: tenant,
	})
	if err != nil {
		s.Logger.Warn("could not read a conversation's chapters",
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

type Index struct {
	Logger   *zap.Logger
	Chapters assistantchapterports.ChapterRepository
}
