package aiauditrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type exportRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewExport(p Params) repositories.AIAuditExportRepository {
	return &exportRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aiaudit-export-repository"),
	}
}

func (r *exportRepository) Create(
	ctx context.Context,
	export *aiaudit.AIAuditExport,
) (*aiaudit.AIAuditExport, error) {
	if _, err := r.db.DBForContext(ctx).NewInsert().Model(export).Returning("*").Exec(ctx); err != nil {
		r.l.Error("failed to create AI audit export", zap.Error(err))

		return nil, fmt.Errorf("create AI audit export: %w", err)
	}

	return export, nil
}

// Update saves an export under optimistic locking, so a worker finishing an
// export and the expiry sweep cannot overwrite each other.
func (r *exportRepository) Update(
	ctx context.Context,
	export *aiaudit.AIAuditExport,
) (*aiaudit.AIAuditExport, error) {
	cols := buncolgen.AIAuditExportColumns
	previous := export.Version
	export.Version++

	res, err := r.db.DBForContext(ctx).NewUpdate().
		Model(export).
		WherePK().
		Where(cols.Version.Eq(), previous).
		Returning("*").
		Exec(ctx)
	if err != nil {
		export.Version = previous
		r.l.Error("failed to update AI audit export",
			zap.String("exportId", export.ID.String()), zap.Error(err))

		return nil, fmt.Errorf("update AI audit export: %w", err)
	}

	if err = dberror.CheckRowsAffected(res, "AI audit export", export.ID.String()); err != nil {
		export.Version = previous

		return nil, err
	}

	return export, nil
}

func (r *exportRepository) GetByID(
	ctx context.Context,
	req repositories.GetAIAuditExportRequest,
) (*aiaudit.AIAuditExport, error) {
	entity := new(aiaudit.AIAuditExport)
	err := r.db.DBForContext(ctx).NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.AIAuditExportScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.AIAuditExportColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AI audit export")
	}

	return entity, nil
}

func (r *exportRepository) ListConnection(
	ctx context.Context,
	req *repositories.ListAIAuditExportsRequest,
) (*pagination.CursorListResult[*aiaudit.AIAuditExport], error) {
	dba := r.db.DBForContext(ctx)

	var totalCount *int
	if req.Cursor.IncludeTotalCount {
		total, err := dba.NewSelect().
			Model((*aiaudit.AIAuditExport)(nil)).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return querybuilder.ApplyFiltersWithoutSort(
					sq,
					buncolgen.AIAuditExportTable.Alias,
					req.Filter,
					(*aiaudit.AIAuditExport)(nil),
				)
			}).
			Count(ctx)
		if err != nil {
			return nil, fmt.Errorf("count AI audit exports: %w", err)
		}
		totalCount = &total
	}

	result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*aiaudit.AIAuditExport]{
		Filter:     req.Filter,
		Cursor:     req.Cursor,
		TotalCount: totalCount,
		Query: func(entities *[]*aiaudit.AIAuditExport) *bun.SelectQuery {
			return dba.NewSelect().Model(entities)
		},
		Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
			return querybuilder.ApplyCursorFilters(
				sq,
				buncolgen.AIAuditExportTable.Alias,
				req.Filter,
				req.Cursor,
				(*aiaudit.AIAuditExport)(nil),
			)
		},
	})
	if err != nil {
		r.l.Error("failed to list AI audit exports", zap.Error(err))

		return nil, fmt.Errorf("list AI audit exports: %w", err)
	}

	return result, nil
}

// ListExpired finds exports whose files are past their download window.
// Deliberately not tenant-scoped: the sweep runs for the whole installation.
func (r *exportRepository) ListExpired(
	ctx context.Context,
	before int64,
	limit int,
) ([]*aiaudit.AIAuditExport, error) {
	cols := buncolgen.AIAuditExportColumns
	rows := make([]*aiaudit.AIAuditExport, 0, limit)

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Where(cols.ArtifactKey.IsNotNull()).
		Where(cols.ArtifactExpiresAt.Lte(), before).
		Where(cols.Status.Eq(), aiaudit.ExportStatusSucceeded).
		Order(cols.ArtifactExpiresAt.OrderAsc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list expired AI audit exports: %w", err)
	}

	return rows, nil
}

// ListStale finds exports left pending or running past any reasonable run,
// whose worker is gone. Not tenant-scoped, for the same reason.
func (r *exportRepository) ListStale(
	ctx context.Context,
	updatedBefore int64,
	limit int,
) ([]*aiaudit.AIAuditExport, error) {
	cols := buncolgen.AIAuditExportColumns
	rows := make([]*aiaudit.AIAuditExport, 0, limit)

	if err := r.db.DBForContext(ctx).NewSelect().
		Model(&rows).
		Where(cols.Status.In(), bun.List([]aiaudit.ExportStatus{
			aiaudit.ExportStatusPending,
			aiaudit.ExportStatusRunning,
		})).
		Where(cols.UpdatedAt.Lt(), updatedBefore).
		Order(cols.UpdatedAt.OrderAsc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("list stale AI audit exports: %w", err)
	}

	return rows, nil
}
