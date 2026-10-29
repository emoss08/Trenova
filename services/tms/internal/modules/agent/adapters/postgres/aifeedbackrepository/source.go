package aifeedbackrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aifeedbackports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxExchangeMessages = 200

var (
	answerErrors       = dberror.Mapping{Entity: "Answer"}
	conversationErrors = dberror.Mapping{Entity: "Conversation"}
)

type SourceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type sourceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSource(p SourceParams) aifeedbackports.AIFeedbackSourceRepository {
	return &sourceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aifeedback-source-repository"),
	}
}

func (r *sourceRepository) GetMessageContext(
	ctx context.Context,
	req aifeedbackports.GetAIFeedbackMessageRequest,
) (*aifeedbackports.AIFeedbackMessageContext, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*aifeedbackports.AIFeedbackMessageContext, error) {
			db := r.db.Keel(ctx)
			message, err := conversationdb.Messages.Get(ctx, db, req.MessageID)
			if err != nil {
				return nil, answerErrors.Map(err, req.MessageID.String())
			}

			thread, err := conversationdb.Threads.Get(ctx, db, message.ThreadID)
			if err != nil {
				return nil, conversationErrors.Map(err, message.ThreadID.String())
			}

			exchange, err := r.exchange(ctx, message)
			if err != nil {
				return nil, err
			}

			turn, err := r.turn(ctx, message)
			if err != nil {
				return nil, err
			}

			return &aifeedbackports.AIFeedbackMessageContext{
				Message:  message,
				Thread:   thread,
				Exchange: exchange,
				Turn:     turn,
			}, nil
		},
	)
}

func (r *sourceRepository) exchange(
	ctx context.Context,
	message *conversation.Message,
) ([]conversation.Message, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]conversation.Message, error) {
		m := conversationdb.Messages
		db := r.db.Keel(ctx)

		if message.Delegated() {
			rows, err := m.Where(
				m.ThreadID.Eq(message.ThreadID),
				m.Kind.Eq(conversation.MessageKindDelegated),
				m.DelegateCallID.Eq(message.DelegateCallID),
				m.Sequence.Lte(int64(message.Sequence)),
			).
				Order(m.Sequence.Desc()).
				Limit(maxExchangeMessages).
				List(ctx, db)
			if err != nil {
				return nil, fmt.Errorf("read delegated exchange: %w", err)
			}

			return chronological(rows), nil
		}

		start, err := keel.IntoOne[*int64](ctx, db, m.
			Where(
				m.ThreadID.Eq(message.ThreadID),
				m.Role.Eq(conversation.RoleUser),
				m.Kind.Ne(conversation.MessageKindDelegated),
				m.Sequence.Lt(int64(message.Sequence)),
			).
			Pick(m.Sequence.Max()))
		if err != nil {
			return nil, fmt.Errorf("find the question an answer replies to: %w", err)
		}
		var from int64
		if start != nil {
			from = *start
		}

		rows, err := m.Where(
			m.ThreadID.Eq(message.ThreadID),
			m.Kind.Ne(conversation.MessageKindDelegated),
			m.Sequence.Gte(from),
			m.Sequence.Lte(int64(message.Sequence)),
		).
			Order(m.Sequence.Desc()).
			Limit(maxExchangeMessages).
			List(ctx, db)
		if err != nil {
			return nil, fmt.Errorf("read exchange: %w", err)
		}

		return chronological(rows), nil
	})
}

func chronological(rows []*conversation.Message) []conversation.Message {
	out := make([]conversation.Message, 0, len(rows))
	for index := len(rows) - 1; index >= 0; index-- {
		out = append(out, *rows[index])
	}

	return out
}

func (r *sourceRepository) turn(
	ctx context.Context,
	message *conversation.Message,
) (*conversationturn.AssistantTurn, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*conversationturn.AssistantTurn, error) {
		t := conversationdb.AssistantTurns
		turn, err := t.Where(t.ThreadID.Eq(message.ThreadID), t.CreatedAt.Lte(message.CreatedAt)).
			Order(t.CreatedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if errors.Is(err, keel.ErrNotFound) {
			return nil, nil
		}
		if err != nil {
			r.l.Warn("failed to find the turn behind an answer", zap.Error(err))

			return nil, fmt.Errorf("find the turn behind an answer: %w", err)
		}

		return turn, nil
	})
}
