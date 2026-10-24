package auditdlqrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dbscope"
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

func New(p Params) auditports.AuditDLQRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.audit-dlq-repository"),
	}
}

func (r *repository) Insert(ctx context.Context, entry *audit.DLQEntry) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := auditdb.DLQEntries.Create(ctx, r.db.Keel(ctx), entry); err != nil {
			r.l.Error("failed to insert DLQ entry",
				zap.String("entryID", entry.ID.String()),
				zap.Error(err),
			)
			return fmt.Errorf("failed to insert DLQ entry: %w", err)
		}
		return nil
	})
}

func (r *repository) InsertBatch(ctx context.Context, entries []*audit.DLQEntry) error {
	ctx = dbscope.WithSystem(ctx, "dead-letter an audit batch that mixes organizations")
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(entries) == 0 {
			return nil
		}

		if err := auditdb.DLQEntries.CreateMany(ctx, r.db.Keel(ctx), entries); err != nil {
			r.l.Error("failed to insert DLQ entries batch",
				zap.Int("count", len(entries)),
				zap.Error(err),
			)
			return fmt.Errorf("failed to insert DLQ entries batch: %w", err)
		}
		return nil
	})
}

func (r *repository) GetPendingEntries(ctx context.Context, limit int) ([]*audit.DLQEntry, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"retry audit entries that failed to flush, which mix organizations",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*audit.DLQEntry, error) {
		d := auditdb.DLQEntries
		entries, err := d.AllTenants().
			Where(
				d.Status.In(audit.DLQStatusPending, audit.DLQStatusRetrying),
				keel.Or(d.NextRetryAt.IsNull(), d.NextRetryAt.Lte(timeutils.NowUnix())),
			).
			Order(d.CreatedAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to get pending DLQ entries",
				zap.Int("limit", limit),
				zap.Error(err),
			)
			return nil, fmt.Errorf("failed to get pending DLQ entries: %w", err)
		}

		return entries, nil
	})
}

func (r *repository) GetByID(ctx context.Context, id pulid.ID) (*audit.DLQEntry, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*audit.DLQEntry, error) {
		entry, err := auditdb.DLQEntries.Get(ctx, r.db.Keel(ctx), id)
		if err != nil {
			return nil, fmt.Errorf("failed to get DLQ entry by ID: %w", err)
		}

		return entry, nil
	})
}

func (r *repository) Update(ctx context.Context, entry *audit.DLQEntry) error {
	ctx = dbscope.WithSystem(ctx, "update a dead-lettered audit entry")
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := auditdb.DLQEntries.Save(ctx, r.db.Keel(ctx), entry); err != nil {
			return fmt.Errorf("failed to update DLQ entry: %w", err)
		}

		return nil
	})
}

func (r *repository) MarkAsRecovered(ctx context.Context, ids []pulid.ID) error {
	ctx = dbscope.WithSystem(ctx, "mark dead-lettered audit entries recovered across organizations")
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(ids) == 0 {
			return nil
		}

		d := auditdb.DLQEntries
		if _, err := d.AllTenants().
			Where(d.ID.In(ids...)).
			Update(ctx, r.db.Keel(ctx), d.Status.To(audit.DLQStatusRecovered)); err != nil {
			return fmt.Errorf("failed to mark DLQ entries as recovered: %w", err)
		}

		r.l.Info("marked DLQ entries as recovered", zap.Int("count", len(ids)))
		return nil
	})
}

func (r *repository) MarkAsFailed(ctx context.Context, id pulid.ID, errMsg string) error {
	ctx = dbscope.WithSystem(ctx, "mark dead-lettered audit entries failed across organizations")
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		d := auditdb.DLQEntries
		if _, err := d.AllTenants().
			Where(d.ID.Eq(id)).
			Update(
				ctx,
				r.db.Keel(ctx),
				d.Status.To(audit.DLQStatusFailed),
				d.LastError.To(errMsg),
			); err != nil {
			return fmt.Errorf("failed to mark DLQ entry as failed: %w", err)
		}

		return nil
	})
}

func (r *repository) DeleteRecovered(ctx context.Context, olderThan int64) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		d := auditdb.DLQEntries
		count, err := d.Where(
			d.Status.Eq(audit.DLQStatusRecovered),
			d.UpdatedAt.Lt(olderThan),
		).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("failed to delete recovered DLQ entries: %w", err)
		}

		r.l.Info("deleted recovered DLQ entries", zap.Int64("count", count))
		return count, nil
	})
}

func (r *repository) Count(ctx context.Context) (int64, error) {
	ctx = dbscope.WithSystem(ctx, "count dead-lettered audit entries instance-wide")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		count, err := auditdb.DLQEntries.AllTenants().Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("failed to count DLQ entries: %w", err)
		}

		return count, nil
	})
}

func (r *repository) CountByStatus(ctx context.Context, status audit.DLQStatus) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		d := auditdb.DLQEntries
		count, err := d.Where(d.Status.Eq(status)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("failed to count DLQ entries by status: %w", err)
		}

		return count, nil
	})
}
