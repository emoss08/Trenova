package assistantchapterrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var chapterErrors = dberror.Mapping{Entity: "Chapter"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.ChapterRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.assistant-chapter-repository"),
	}
}

func chaptersQuery(threadID pulid.ID) *keel.Query[*conversation.Chapter] {
	c := conversationdb.Chapters
	tr := conversationdb.AssistantTurns.As("ctr")
	m := conversationdb.Messages.As("cfm")
	first := keel.Scalar(m.
		Pick(m.Sequence).
		Where(m.ThreadID.EqCol(c.ThreadID), m.TurnID.EqCol(c.TurnID)).
		Order(m.Sequence.Asc()).
		Limit(1))

	return c.Join(tr, tr.ID.EqCol(c.TurnID)).
		Compute(c.StartSequence, keel.Int64Of(first)).
		Compute(c.StartedAt, tr.CreatedAt).
		Where(c.ThreadID.Eq(threadID)).
		Order(tr.CreatedAt.Asc(), c.ID.Asc())
}

func (r *repository) List(
	ctx context.Context,
	req agentports.ListChaptersRequest,
) ([]*conversation.Chapter, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*conversation.Chapter, error) {
		rows, err := chaptersQuery(req.ThreadID).
			Limit(conversation.MaxChaptersPerThread).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list chapters: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) Get(
	ctx context.Context,
	req agentports.GetChapterRequest,
) (*conversation.Chapter, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversation.Chapter, error) {
		c := conversationdb.Chapters
		row, err := chaptersQuery(req.ThreadID).Where(c.ID.Eq(req.ID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, chapterErrors.Map(err, req.ID.String())
		}

		return row, nil
	})
}

func (r *repository) ForTurn(
	ctx context.Context,
	req agentports.ChapterForTurnRequest,
) (*conversation.Chapter, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversation.Chapter, error) {
		c := conversationdb.Chapters
		row, err := chaptersQuery(req.ThreadID).
			Where(c.TurnID.Eq(req.TurnID)).
			One(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read the chapter a turn opened: %w", err)
		}

		return row, nil
	})
}

func (r *repository) Count(ctx context.Context, req agentports.ListChaptersRequest) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		c := conversationdb.Chapters
		count, err := c.Where(c.ThreadID.Eq(req.ThreadID)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count chapters: %w", err)
		}

		return int(count), nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	chapter *conversation.Chapter,
) (*conversation.Chapter, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.Chapter, error) {
		db := r.db.Keel(ctx)
		c := conversationdb.Chapters
		if _, err := c.CreateOrSkip(ctx, db, []*conversation.Chapter{chapter}); err != nil {
			return nil, chapterErrors.Map(err, chapter.ID.String())
		}

		row, err := chaptersQuery(chapter.ThreadID).Where(c.TurnID.Eq(chapter.TurnID)).One(ctx, db)
		if err != nil {
			return nil, chapterErrors.Map(err, chapter.ID.String())
		}

		return row, nil
	})
}

func (r *repository) Retitle(
	ctx context.Context,
	req agentports.RetitleChapterRequest,
) (*conversation.Chapter, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*conversation.Chapter, error) {
		db := r.db.Keel(ctx)
		c := conversationdb.Chapters

		var guard keel.Pred
		switch {
		case req.Version != nil:
			guard = c.Version.Eq(*req.Version)
		case req.TitleSource != conversation.ChapterTitlePerson:
			guard = c.TitleSource.NotIn(conversation.ChapterTitlePerson)
		}

		n, err := c.Where(c.ID.Eq(req.ID), c.ThreadID.Eq(req.ThreadID), guard).Update(ctx, db,
			c.Title.To(req.Title),
			c.TitleSource.To(req.TitleSource),
		)
		if err != nil {
			return nil, chapterErrors.Map(err, req.ID.String())
		}

		row, err := chaptersQuery(req.ThreadID).Where(c.ID.Eq(req.ID)).One(ctx, db)
		if err != nil {
			return nil, chapterErrors.Map(err, req.ID.String())
		}
		if n == 0 && req.Version != nil {
			return nil, errortypes.NewValidationError(
				"version",
				errortypes.ErrVersionMismatch,
				"This chapter was renamed since you opened it. Read its new title and try again.",
			)
		}

		return row, nil
	})
}

func (r *repository) Delete(
	ctx context.Context,
	req agentports.DeleteChapterRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		c := conversationdb.Chapters
		n, err := c.Where(
			c.ID.Eq(req.ID),
			c.ThreadID.Eq(req.ThreadID),
			keel.When(req.KeepPerson, c.TitleSource.NotIn(conversation.ChapterTitlePerson)),
		).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, chapterErrors.Map(err, req.ID.String())
		}

		return n > 0, nil
	})
}

func (r *repository) TurnOfMessage(
	ctx context.Context,
	req agentports.TurnOfMessageRequest,
) (pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (pulid.ID, error) {
		m := conversationdb.Messages
		turn, err := keel.IntoOne[*string](ctx, r.db.Keel(ctx), m.
			Pick(m.TurnID).
			Where(m.ThreadID.Eq(req.ThreadID), m.ID.Eq(req.MessageID)))
		if errors.Is(err, keel.ErrNotFound) {
			return "", errortypes.NewNotFoundError("Message not found in this conversation")
		}
		if err != nil {
			return "", fmt.Errorf("read the turn of a message: %w", err)
		}
		if turn == nil || *turn == "" {
			return "", nil
		}

		return pulid.ID(*turn), nil
	})
}
