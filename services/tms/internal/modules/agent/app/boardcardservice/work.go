package boardcardservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardassemble"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desklens"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskattentionports"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) readWork(
	ctx context.Context,
	req *assistantports.BoardFillRequest,
	reads *boardassemble.Reads,
) error {
	ids := req.Board.WorkRecords()
	reads.Work = make(map[pulid.ID][]board.WorkSignal, len(ids))
	if len(ids) == 0 || s.attention == nil {
		return nil
	}
	attention, err := s.attention.Attention(ctx, &deskattentionports.AttentionRequest{
		TenantInfo: req.Actor.TenantInfo(),
		First:      len(ids),
		RecordIDs:  ids,
	}, req.Actor)
	if err != nil {
		return err
	}
	for _, group := range attention.Groups {
		if group.Record == nil {
			continue
		}
		for _, item := range group.Items {
			reads.Work[group.Record.ID] = append(reads.Work[group.Record.ID], board.WorkSignal{
				Lens:  desklens.Of(item.SourceKind).String(),
				Title: item.Title,
				Tone:  toneOf(item.Severity),
				DueAt: item.DueAt,
			})
		}
	}

	return nil
}

func toneOf(severity watchtower.Severity) board.Tone {
	switch severity {
	case watchtower.SeverityCritical:
		return board.ToneBad
	case watchtower.SeverityWarning:
		return board.ToneWarn
	default:
		return board.ToneInfo
	}
}
