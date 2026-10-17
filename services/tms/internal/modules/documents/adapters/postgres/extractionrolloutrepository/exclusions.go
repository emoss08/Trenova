package extractionrolloutrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionrolloutports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/extractionrolloutcol"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type exclusionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewExclusions(p Params) extractionrolloutports.RolloutExclusionRepository {
	return &exclusionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.extractionrollout-exclusion-repository"),
	}
}

func uniqueExcludedExtraction() string {
	cols := extractionrolloutcol.RolloutExclusionColumns
	return "CONFLICT (" + strings.Join([]string{
		cols.OrganizationID.Bare(),
		cols.BusinessUnitID.Bare(),
		cols.DocumentID.Bare(),
		cols.ExtractedAt.Bare(),
	}, ", ") + ") DO NOTHING"
}

func (r *exclusionRepository) Record(
	ctx context.Context,
	entity *extractionrollout.RolloutExclusion,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On(uniqueExcludedExtraction()).
			Exec(ctx); err != nil {
			r.l.Error("failed to record rollout exclusion",
				zap.String("documentId", entity.DocumentID.String()),
				zap.Error(err),
			)

			return fmt.Errorf("record rollout exclusion: %w", err)
		}

		return nil
	})
}

func (r *exclusionRepository) Totals(
	ctx context.Context,
	req extractionrolloutports.TotalRolloutAssignmentsRequest,
) ([]extractionrolloutports.RolloutExclusionTotal, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]extractionrolloutports.RolloutExclusionTotal, error) {
			cols := extractionrolloutcol.RolloutExclusionColumns
			totals := make(
				[]extractionrolloutports.RolloutExclusionTotal,
				0,
				len(extractionrollout.AllExclusionReasons()),
			)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model((*extractionrollout.RolloutExclusion)(nil)).
				Column(cols.Reason.Bare()).
				ColumnExpr(buncolgen.Count("count")).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return extractionrolloutcol.RolloutExclusionScopeTenant(sq, req.TenantInfo).
						Where(cols.CandidateProviderID.Eq(), req.CandidateProviderID).
						Where(cols.CreatedAt.Gte(), req.Since)
				}).
				Group(cols.Reason.Bare()).
				Order(cols.Reason.OrderAsc()).
				Scan(ctx, &totals)
			if err != nil {
				r.l.Error("failed to total rollout exclusions", zap.Error(err))

				return nil, fmt.Errorf("total rollout exclusions: %w", err)
			}

			return totals, nil
		},
	)
}

func (r *exclusionRepository) PurgeBefore(
	ctx context.Context,
	req extractionrolloutports.PurgeRolloutAssignmentsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		cols := extractionrolloutcol.RolloutExclusionColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultPurgeLimit
		}
		limit = min(limit, maxPurgeLimit)

		dba := r.db.DBForContext(ctx)
		expired := dba.NewSelect().
			Model((*extractionrollout.RolloutExclusion)(nil)).
			Column(cols.ID.Bare()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return extractionrolloutcol.RolloutExclusionScopeTenant(sq, req.TenantInfo).
					Where(cols.CreatedAt.Lt(), req.Before)
			}).
			OrderExpr(cols.CreatedAt.OrderAsc()).
			Limit(int64(limit))

		res, err := dba.NewDelete().
			Model((*extractionrollout.RolloutExclusion)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return extractionrolloutcol.RolloutExclusionScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.In(), expired)
			}).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to purge rollout exclusions", zap.Error(err))

			return 0, fmt.Errorf("purge rollout exclusions: %w", err)
		}

		purged, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("purge rollout exclusions rows: %w", err)
		}

		return purged, nil
	})
}
