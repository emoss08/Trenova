package aiauditrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit/aiauditdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

var exportErrors = dberror.Mapping{Entity: "AI audit export"}

type exportRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewExport(p Params) aiauditports.AIAuditExportRepository {
	return &exportRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aiaudit-export-repository"),
	}
}

func (r *exportRepository) Create(
	ctx context.Context,
	export *aiaudit.AIAuditExport,
) (*aiaudit.AIAuditExport, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aiaudit.AIAuditExport, error) {
		if err := aiauditdb.AIAuditExports.Create(ctx, r.db.Keel(ctx), export); err != nil {
			r.l.Error("failed to create AI audit export", zap.Error(err))

			return nil, exportErrors.Map(err, export.ID.String())
		}

		return export, nil
	})
}

// Update saves an export under optimistic locking, so a worker finishing an
// export and the expiry sweep cannot overwrite each other.
func (r *exportRepository) Update(
	ctx context.Context,
	export *aiaudit.AIAuditExport,
) (*aiaudit.AIAuditExport, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aiaudit.AIAuditExport, error) {
		if err := aiauditdb.AIAuditExports.Save(ctx, r.db.Keel(ctx), export); err != nil {
			r.l.Error("failed to update AI audit export",
				zap.String("exportId", export.ID.String()), zap.Error(err))

			return nil, exportErrors.Map(err, export.ID.String())
		}

		return export, nil
	})
}

func (r *exportRepository) GetByID(
	ctx context.Context,
	req aiauditports.GetAIAuditExportRequest,
) (*aiaudit.AIAuditExport, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aiaudit.AIAuditExport, error) {
		entity, err := aiauditdb.AIAuditExports.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, exportErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *exportRepository) ListConnection(
	ctx context.Context,
	req *aiauditports.ListAIAuditExportsRequest,
) (*pagination.CursorListResult[*aiaudit.AIAuditExport], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*aiaudit.AIAuditExport], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				aiauditdb.AIAuditExports.Query(),
				&listquery.Request[aiaudit.AIAuditExport]{
					Filter: req.Filter,
					Cursor: req.Cursor,
				},
			)
			if err != nil {
				r.l.Error("failed to list AI audit exports", zap.Error(err))

				return nil, fmt.Errorf("list AI audit exports: %w", err)
			}

			return result, nil
		},
	)
}

// ListExpired finds exports whose files are past their download window.
// Deliberately not tenant-scoped: the sweep runs for the whole installation.
func (r *exportRepository) ListExpired(
	ctx context.Context,
	before int64,
	limit int,
) ([]*aiaudit.AIAuditExport, error) {
	ctx = dbscope.WithSystem(ctx, "list expired AI audit exports across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiaudit.AIAuditExport, error) {
		x := aiauditdb.AIAuditExports
		rows, err := x.AllTenants().
			Where(
				x.ArtifactKey.Ne(""),
				x.ArtifactExpiresAt.Lte(before),
				x.Status.Eq(aiaudit.ExportStatusSucceeded),
			).
			Order(x.ArtifactExpiresAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list expired AI audit exports: %w", err)
		}

		return rows, nil
	})
}

// ListStale finds exports left pending or running past any reasonable run,
// whose worker is gone. Not tenant-scoped, for the same reason.
func (r *exportRepository) ListStale(
	ctx context.Context,
	updatedBefore int64,
	limit int,
) ([]*aiaudit.AIAuditExport, error) {
	ctx = dbscope.WithSystem(ctx, "list stale AI audit exports across every organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aiaudit.AIAuditExport, error) {
		x := aiauditdb.AIAuditExports
		rows, err := x.AllTenants().
			Where(
				x.Status.In(aiaudit.ExportStatusPending, aiaudit.ExportStatusRunning),
				x.UpdatedAt.Lt(updatedBefore),
			).
			Order(x.UpdatedAt.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list stale AI audit exports: %w", err)
		}

		return rows, nil
	})
}
