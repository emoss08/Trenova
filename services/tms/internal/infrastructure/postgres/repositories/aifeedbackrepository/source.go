package aifeedbackrepository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxExchangeMessages = 200

type SourceParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type sourceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSource(p SourceParams) repositories.AIFeedbackSourceRepository {
	return &sourceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aifeedback-source-repository"),
	}
}

func (r *sourceRepository) GetMessageContext(
	ctx context.Context,
	req repositories.GetAIFeedbackMessageRequest,
) (*repositories.AIFeedbackMessageContext, error) {
	dba := r.db.DBForContext(ctx)
	msgCols := buncolgen.MessageColumns

	message := new(conversation.Message)
	if err := dba.NewSelect().
		Model(message).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.MessageScopeTenant(sq, req.TenantInfo).
				Where(msgCols.ID.Eq(), req.MessageID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Answer")
	}

	thread := new(conversation.Thread)
	if err := dba.NewSelect().
		Model(thread).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ThreadScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.ThreadColumns.ID.Eq(), message.ThreadID)
		}).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Conversation")
	}

	exchange, err := r.exchange(ctx, req, message)
	if err != nil {
		return nil, err
	}

	turn, err := r.turn(ctx, req, message)
	if err != nil {
		return nil, err
	}

	return &repositories.AIFeedbackMessageContext{
		Message:  message,
		Thread:   thread,
		Exchange: exchange,
		Turn:     turn,
	}, nil
}

func (r *sourceRepository) exchange(
	ctx context.Context,
	req repositories.GetAIFeedbackMessageRequest,
	message *conversation.Message,
) ([]conversation.Message, error) {
	cols := buncolgen.MessageColumns
	dba := r.db.DBForContext(ctx)
	rows := make([]conversation.Message, 0)

	if message.Delegated() {
		err := dba.NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.MessageScopeTenant(sq, req.TenantInfo).
					Where(cols.ThreadID.Eq(), message.ThreadID).
					Where(cols.Kind.Eq(), conversation.MessageKindDelegated).
					Where(cols.DelegateCallID.Eq(), message.DelegateCallID).
					Where(cols.Sequence.Lte(), message.Sequence)
			}).
			OrderExpr(cols.Sequence.OrderDesc()).
			Limit(maxExchangeMessages).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("read delegated exchange: %w", err)
		}

		return chronological(rows), nil
	}

	var start sql.NullInt64
	err := dba.NewSelect().
		Model((*conversation.Message)(nil)).
		ColumnExpr(buncolgen.Max(cols.Sequence, "start_sequence")).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.MessageScopeTenant(sq, req.TenantInfo).
				Where(cols.ThreadID.Eq(), message.ThreadID).
				Where(cols.Role.Eq(), conversation.RoleUser).
				Where(cols.Kind.NotEq(), conversation.MessageKindDelegated).
				Where(cols.Sequence.Lt(), message.Sequence)
		}).
		Scan(ctx, &start)
	if err != nil {
		return nil, fmt.Errorf("find the question an answer replies to: %w", err)
	}

	err = dba.NewSelect().
		Model(&rows).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.MessageScopeTenant(sq, req.TenantInfo).
				Where(cols.ThreadID.Eq(), message.ThreadID).
				Where(cols.Kind.NotEq(), conversation.MessageKindDelegated).
				Where(cols.Sequence.Gte(), start.Int64).
				Where(cols.Sequence.Lte(), message.Sequence)
		}).
		OrderExpr(cols.Sequence.OrderDesc()).
		Limit(maxExchangeMessages).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("read exchange: %w", err)
	}

	return chronological(rows), nil
}

func chronological(rows []conversation.Message) []conversation.Message {
	for left, right := 0, len(rows)-1; left < right; left, right = left+1, right-1 {
		rows[left], rows[right] = rows[right], rows[left]
	}

	return rows
}

func (r *sourceRepository) turn(
	ctx context.Context,
	req repositories.GetAIFeedbackMessageRequest,
	message *conversation.Message,
) (*conversation.AssistantTurn, error) {
	cols := buncolgen.AssistantTurnColumns
	turn := new(conversation.AssistantTurn)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(turn).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AssistantTurnScopeTenant(sq, req.TenantInfo).
				Where(cols.ThreadID.Eq(), message.ThreadID).
				Where(cols.CreatedAt.Lte(), message.CreatedAt)
		}).
		OrderExpr(cols.CreatedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		r.l.Warn("failed to find the turn behind an answer", zap.Error(err))

		return nil, fmt.Errorf("find the turn behind an answer: %w", err)
	}

	return turn, nil
}
