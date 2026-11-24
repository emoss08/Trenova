package boardcardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"go.uber.org/zap"
)

func ReadListsWith(boards *Service, lists assistantports.ListSizer) {
	boards.lists = lists
}

func (s *Service) readLists(
	ctx context.Context,
	req *assistantports.BoardFillRequest,
	reads *boardassemble.Reads,
) {
	for _, id := range req.Board.Lists() {
		read := &boardassemble.ListRead{}
		reads.Lists[id] = read
		if s.lists == nil {
			s.l.Error("a board names a list, and no list sizer is linked")
			continue
		}
		size, err := s.lists.ListSize(ctx, &assistantports.ListSizeRequest{
			Actor:      req.Actor,
			ThreadID:   req.ThreadID,
			ArtifactID: id,
			Agent:      req.Agent,
		})
		if err != nil {
			s.l.Warn("a board's list could not be counted", zap.Error(err))
			continue
		}
		read.Title, read.Kind, read.Total = size.Title, recordkind.Kind(size.Kind), &size.Total
	}
}

func (s *Service) checkLists(ctx context.Context, req *assistantports.BoardFillRequest) error {
	for _, id := range req.Board.Lists() {
		artifact, err := s.artifact(ctx, req, id)
		if err != nil || !tabular(artifact) {
			return mistake("a figure counts the list %s, which is not a table this "+
				"conversation kept; name it by the artifactId its result gave you", id)
		}
	}

	return nil
}
