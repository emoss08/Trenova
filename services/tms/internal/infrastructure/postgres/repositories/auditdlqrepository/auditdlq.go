package auditdlqrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AuditDLQRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.audit-dlq-repository"),
	}
}

func (r *repository) Insert(ctx context.Context, entry *audit.DLQEntry) error {
	if _, err := r.db.DB().NewInsert().Model(entry).Exec(ctx); err != nil {
		r.l.Error("failed to insert DLQ entry",
			zap.String("entryID", entry.ID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to insert DLQ entry: %w", err)
	}
	return nil
}

func (r *repository) InsertBatch(ctx context.Context, entries []*audit.DLQEntry) error {
	if len(entries) == 0 {
		return nil
	}

	if _, err := r.db.DB().NewInsert().Model(&entries).Exec(ctx); err != nil {
		r.l.Error("failed to insert DLQ entries batch",
			zap.Int("count", len(entries)),
			zap.Error(err),
		)
		return fmt.Errorf("failed to insert DLQ entries batch: %w", err)
	}
	return nil
}

func (r *repository) GetPendingEntries(ctx context.Context, limit int) ([]*audit.DLQEntry, error) {
	entries := make([]*audit.DLQEntry, 0, limit)

	now := timeutils.NowUnix()
	err := r.db.DB().NewSelect().
		Model(&entries).
		Where("status IN (?, ?)", audit.DLQStatusPending, audit.DLQStatusRetrying).
		Where("next_retry_at IS NULL OR next_retry_at <= ?", now).
		OrderExpr("created_at ASC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get pending DLQ entries",
			zap.Int("limit", limit),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get pending DLQ entries: %w", err)
	}

	return entries, nil
}

func (r *repository) GetByID(ctx context.Context, id pulid.ID) (*audit.DLQEntry, error) {
	entry := new(audit.DLQEntry)

	err := r.db.DB().NewSelect().
		Model(entry).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ entry by ID: %w", err)
	}

	return entry, nil
}

func (r *repository) Update(ctx context.Context, entry *audit.DLQEntry) error {
	entry.UpdatedAt = timeutils.NowUnix()

	if _, err := r.db.DB().NewUpdate().
		Model(entry).
		WherePK().
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to update DLQ entry: %w", err)
	}

	return nil
}

func (r *repository) MarkAsRecovered(ctx context.Context, ids []pulid.ID) error {
	if len(ids) == 0 {
		return nil
	}

	now := timeutils.NowUnix()
	if _, err := r.db.DB().NewUpdate().
		Model((*audit.DLQEntry)(nil)).
		Set("status = ?", audit.DLQStatusRecovered).
		Set("updated_at = ?", now).
		Where("id IN (?)", ids).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to mark DLQ entries as recovered: %w", err)
	}

	r.l.Info("marked DLQ entries as recovered", zap.Int("count", len(ids)))
	return nil
}

func (r *repository) MarkAsFailed(ctx context.Context, id pulid.ID, errMsg string) error {
	now := timeutils.NowUnix()
	if _, err := r.db.DB().NewUpdate().
		Model((*audit.DLQEntry)(nil)).
		Set("status = ?", audit.DLQStatusFailed).
		Set("last_error = ?", errMsg).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to mark DLQ entry as failed: %w", err)
	}

	return nil
}

func (r *repository) DeleteRecovered(ctx context.Context, olderThan int64) (int64, error) {
	result, err := r.db.DB().NewDelete().
		Model((*audit.DLQEntry)(nil)).
		Where("status = ?", audit.DLQStatusRecovered).
		Where("updated_at < ?", olderThan).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to delete recovered DLQ entries: %w", err)
	}

	count, _ := result.RowsAffected()
	r.l.Info("deleted recovered DLQ entries", zap.Int64("count", count))
	return count, nil
}

func (r *repository) Count(ctx context.Context) (int64, error) {
	count, err := r.db.DB().NewSelect().
		Model((*audit.DLQEntry)(nil)).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count DLQ entries: %w", err)
	}

	return int64(count), nil
}

func (r *repository) CountByStatus(ctx context.Context, status audit.DLQStatus) (int64, error) {
	count, err := r.db.DB().NewSelect().
		Model((*audit.DLQEntry)(nil)).
		Where("status = ?", status).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count DLQ entries by status: %w", err)
	}

	return int64(count), nil
}
