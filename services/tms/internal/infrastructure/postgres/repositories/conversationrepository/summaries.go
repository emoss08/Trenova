package conversationrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const maxSummaryList = 200

var _ repositories.ThreadSummaryRepository = (*repository)(nil)

func NewThreadSummaries(p Params) repositories.ThreadSummaryRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.thread-summary-repository"),
	}
}

func (r *repository) CreateIfNewest(
	ctx context.Context,
	summary *conversation.Summary,
) (bool, error) {
	created := false
	err := r.db.DB().RunInTx(ctx, nil, func(txCtx context.Context, tx bun.Tx) error {
		threadCols := buncolgen.ThreadColumns
		var threadID string
		err := tx.NewSelect().
			Model((*conversation.Thread)(nil)).
			Column(threadCols.ID.Bare()).
			Where(threadCols.ID.Eq(), summary.ThreadID).
			Where(threadCols.OrganizationID.Eq(), summary.OrganizationID).
			Where(threadCols.BusinessUnitID.Eq(), summary.BusinessUnitID).
			For("UPDATE").
			Scan(txCtx, &threadID)
		if err != nil {
			return dberror.HandleNotFoundError(err, "Thread")
		}

		cols := buncolgen.SummaryColumns
		newer, err := tx.NewSelect().
			Model((*conversation.Summary)(nil)).
			Where(cols.ThreadID.Eq(), summary.ThreadID).
			Where(cols.OrganizationID.Eq(), summary.OrganizationID).
			Where(cols.BusinessUnitID.Eq(), summary.BusinessUnitID).
			Where(cols.ThroughSequence.Gte(), summary.ThroughSequence).
			Exists(txCtx)
		if err != nil {
			return fmt.Errorf("check for a newer summary: %w", err)
		}
		if newer {
			return nil
		}

		if _, err = tx.NewInsert().Model(summary).Returning("*").Exec(txCtx); err != nil {
			return fmt.Errorf("insert summary: %w", err)
		}
		created = true

		return nil
	})
	if err != nil {
		r.l.Error("failed to save conversation summary",
			zap.String("thread", summary.ThreadID.String()),
			zap.Error(err),
		)

		return false, err
	}

	return created, nil
}

func (r *repository) Latest(
	ctx context.Context,
	req repositories.ThreadSummaryRequest,
) (*conversation.Summary, error) {
	cols := buncolgen.SummaryColumns
	entity := new(conversation.Summary)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.SummaryScopeTenant(sq, req.TenantInfo).
				Where(cols.ThreadID.Eq(), req.ThreadID)
		}).
		Order(cols.ThroughSequence.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read latest summary: %w", err)
	}

	return entity, nil
}

func (r *repository) List(
	ctx context.Context,
	req repositories.ListThreadSummariesRequest,
) ([]*conversation.Summary, error) {
	limit := req.Limit
	if limit <= 0 || limit > maxSummaryList {
		limit = maxSummaryList
	}

	cols := buncolgen.SummaryColumns
	summaries := make([]*conversation.Summary, 0, 4)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&summaries).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.SummaryScopeTenant(sq, req.TenantInfo).
				Where(cols.ThreadID.Eq(), req.ThreadID)
		}).
		Order(cols.ThroughSequence.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list summaries: %w", err)
	}

	return summaries, nil
}
