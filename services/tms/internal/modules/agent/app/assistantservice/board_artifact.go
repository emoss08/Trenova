package assistantservice

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"go.uber.org/zap"
)

var (
	errUnknownBoard error = &agentports.PublishMistake{
		Reason: "there is no board with that boardId in this conversation; leave boardId out " +
			"to show a new one",
	}
	errNoBoards   = errors.New("boards cannot be shown here")
	errBoardStore = errors.New("the board could not be saved")
)

func (r *artifactRecorder) showBoard(
	callID string,
	request agentports.BoardRequest,
) (*agentports.ShownArtifact, error) {
	boards := r.boards()
	if boards == nil || r.reader == nil {
		return nil, errNoBoards
	}
	fill := &agentports.BoardFillRequest{
		Actor:    r.reader,
		ThreadID: r.thread.ID,
		Board:    &request.Board,
	}
	if err := boards.Check(r.ctx, fill); err != nil {
		return nil, err
	}

	var latest *assistantartifact.Artifact
	if request.ArtifactID.IsNotNil() {
		existing, err := r.repo.GetByID(r.ctx, assistantartifactports.GetArtifactRequest{
			ID:         request.ArtifactID,
			TenantInfo: r.tenant,
		})
		if err != nil || existing.ThreadID != r.thread.ID ||
			existing.Kind != assistantartifact.KindBoard {
			return nil, errUnknownBoard
		}
		latest = r.latestVersion(existing)
	}

	filled, err := boards.Fill(r.ctx, fill)
	if err != nil {
		r.logger.Error("board could not be filled", zap.Error(err))
		return nil, errBoardStore
	}
	author := r.author()
	payload := &board.Payload{Board: request.Board, Snapshot: filled.Snapshot(), Author: author}
	fields, err := payload.Map()
	if err != nil {
		return nil, errBoardStore
	}

	artifact := &assistantartifact.Artifact{
		Kind:             assistantartifact.KindBoard,
		Status:           assistantartifact.StatusReady,
		Title:            assistantartifact.NormalizeTitle(request.Board.Title),
		Payload:          fields,
		SourceToolCallID: callID,
	}
	if latest != nil {
		artifact.FollowLineage(latest)
		artifact.Slug = latest.Slug
		artifact.Pinned = latest.Pinned
	}
	saved, err := r.save(artifact)
	if err != nil {
		return nil, errBoardStore
	}

	shown := shownArtifact(saved)
	shown.Brief = filled.Markdown(board.MarkdownMeta{
		Author:     author,
		AnsweredAt: saved.CreatedAt,
		Location:   boards.Location(r.ctx, r.tenant),
	})

	return shown, nil
}

func (r *artifactRecorder) boards() agentports.BoardFiller {
	if r.svc == nil {
		return nil
	}

	return r.svc.boards
}
