package aicorrectionrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPurgeLimit    = 1000
	maxPurgeLimit        = 10000
	defaultAccuracyLimit = 2000
	defaultTrainingLimit = 100
	maxTrainingLimit     = 500
	maxAccuracyLimit     = 10000
	correctionEntity     = "AICorrection"
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

func New(p Params) repositories.AICorrectionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.aicorrection-repository"),
	}
}

func uniqueSource() string {
	cols := buncolgen.CorrectionColumns

	return "CONFLICT (" + strings.Join([]string{
		cols.OrganizationID.Bare(),
		cols.BusinessUnitID.Bare(),
		cols.SourceType.Bare(),
		cols.SourceID.Bare(),
	}, ", ") + ") DO UPDATE"
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *aicorrection.Correction,
) (*aicorrection.Correction, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*aicorrection.Correction, error) {
		cols := buncolgen.CorrectionColumns

		_, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On(uniqueSource()).
			Set(cols.Task.SetExcluded()).
			Set(cols.DocumentID.SetExcluded()).
			Set(cols.SubjectType.SetExcluded()).
			Set(cols.SubjectID.SetExcluded()).
			Set(cols.CapturedByID.SetExcluded()).
			Set(cols.DocumentKind.SetExcluded()).
			Set(cols.DocumentFingerprint.SetExcluded()).
			Set(cols.ExtractionModel.SetExcluded()).
			Set(cols.ExtractionProviderID.SetExcluded()).
			Set(cols.PredictedConfidence.SetExcluded()).
			Set(cols.Predicted.SetExcluded()).
			Set(cols.Confirmed.SetExcluded()).
			Set(cols.FieldResults.SetExcluded()).
			Set(cols.ScoredCount.SetExcluded()).
			Set(cols.CorrectCount.SetExcluded()).
			Set(cols.CorrectedCount.SetExcluded()).
			Set(cols.MissedCount.SetExcluded()).
			Set(cols.UnconfirmedCount.SetExcluded()).
			Set(cols.UnscoredCount.SetExcluded()).
			Set(cols.CapturedAt.SetExcluded()).
			Set(cols.Version.IncConflict(1)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to upsert ai correction",
				zap.String("sourceType", entity.SourceType.String()),
				zap.String("sourceId", entity.SourceID.String()),
				zap.Error(err),
			)

			return nil, fmt.Errorf("upsert ai correction: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) PurgeBefore(
	ctx context.Context,
	req repositories.PurgeAICorrectionsRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		cols := buncolgen.CorrectionColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultPurgeLimit
		}
		if limit > maxPurgeLimit {
			limit = maxPurgeLimit
		}

		dba := r.db.DBForContext(ctx)
		expired := dba.NewSelect().
			Model((*aicorrection.Correction)(nil)).
			Column(cols.ID.Bare()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.CorrectionScopeTenant(sq, req.TenantInfo).
					Where(cols.CreatedAt.Lt(), req.Before)
			}).
			OrderExpr(cols.CreatedAt.OrderAsc()).
			Limit(limit)

		res, err := dba.NewDelete().
			Model((*aicorrection.Correction)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.CorrectionScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.In(), expired)
			}).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to purge expired ai corrections", zap.Error(err))

			return 0, fmt.Errorf("purge expired ai corrections: %w", err)
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("purge expired ai corrections rows: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAICorrectionRequest,
) (*aicorrection.Correction, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aicorrection.Correction, error) {
		entity := new(aicorrection.Correction)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.CorrectionScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.CorrectionColumns.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, correctionEntity)
		}

		return entity, nil
	})
}

func (r *repository) GetLatestByDocument(
	ctx context.Context,
	req *repositories.GetLatestAICorrectionByDocumentRequest,
) (*aicorrection.Correction, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*aicorrection.Correction, error) {
		cols := buncolgen.CorrectionColumns
		entity := new(aicorrection.Correction)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.CorrectionScopeTenant(sq, req.TenantInfo).
					Where(cols.Task.Eq(), req.Task).
					Where(cols.DocumentID.Eq(), req.DocumentID)
			}).
			Order(cols.CapturedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, correctionEntity)
		}

		return entity, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAICorrectionConnectionRequest,
) (*pagination.CursorListResult[*aicorrection.Correction], error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pagination.CursorListResult[*aicorrection.Correction], error) {
		dba := r.db.DBForContext(ctx)

		var totalCount *int
		if req.Cursor.IncludeTotalCount {
			total, err := dba.NewSelect().
				Model((*aicorrection.Correction)(nil)).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					sq = querybuilder.ApplyFiltersWithoutSort(
						sq,
						buncolgen.CorrectionTable.Alias,
						req.Filter,
						(*aicorrection.Correction)(nil),
					)

					return sq.Apply(buncolgen.CorrectionApplyTenant(req.Filter.TenantInfo))
				}).
				Count(ctx)
			if err != nil {
				r.l.Error("failed to count ai corrections", zap.Error(err))

				return nil, err
			}
			totalCount = &total
		}

		result, err := dbhelper.CursorList(ctx, dbhelper.CursorListParams[*aicorrection.Correction]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: totalCount,
			Query: func(entities *[]*aicorrection.Correction) *bun.SelectQuery {
				q := dba.NewSelect().Model(entities)
				if len(req.Columns) > 0 {
					q = q.Column(req.Columns...)
				}

				return q
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return querybuilder.ApplyCursorFilters(
					sq,
					buncolgen.CorrectionTable.Alias,
					req.Filter,
					req.Cursor,
					(*aicorrection.Correction)(nil),
				)
			},
		})
		if err != nil {
			r.l.Error("failed to list ai corrections", zap.Error(err))

			return nil, err
		}

		return result, nil
	})
}

func (r *repository) TotalsByProvider(
	ctx context.Context,
	req *repositories.TotalAICorrectionsByProviderRequest,
) ([]repositories.AICorrectionProviderTotal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]repositories.AICorrectionProviderTotal, error) {
		cols := buncolgen.CorrectionColumns
		totals := make([]repositories.AICorrectionProviderTotal, 0, 2)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*aicorrection.Correction)(nil)).
			ColumnExpr(cols.ExtractionProviderID.Expr("{} = ? AS candidate"), req.ProviderID).
			ColumnExpr(cols.ScoredCount.Expr("COALESCE(SUM({}), 0) AS scored")).
			ColumnExpr(cols.CorrectCount.Expr("COALESCE(SUM({}), 0) AS correct")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.CorrectionScopeTenant(sq, req.TenantInfo).
					Where(cols.Task.Eq(), req.Task).
					Where(cols.CapturedAt.Gte(), req.Since).
					Where(cols.ExtractionProviderID.IsNotNull())
			}).
			GroupExpr("candidate").
			Scan(ctx, &totals)
		if err != nil {
			r.l.Error("failed to total ai corrections by provider", zap.Error(err))

			return nil, fmt.Errorf("total ai corrections by provider: %w", err)
		}

		return totals, nil
	})
}

const (
	firstMondayEpoch = 4 * timeutils.SecondsPerDay
	secondsPerWeek   = 7 * timeutils.SecondsPerDay
)

func (r *repository) WeeklyTotalsByProvider(
	ctx context.Context,
	req *repositories.WeeklyAICorrectionTotalsRequest,
) ([]aicorrection.WeekTotal, error) {
	return r.weeklyTotals(ctx, req.Task, req.Since, func(_ context.Context, sq *bun.SelectQuery) *bun.SelectQuery {
		return buncolgen.CorrectionScopeTenant(sq, req.TenantInfo)
	})
}

func (r *repository) WeeklyTrainableTotalsByProvider(
	ctx context.Context,
	req *repositories.WeeklyTrainableAICorrectionTotalsRequest,
) ([]aicorrection.WeekTotal, error) {
	return r.weeklyTotals(ctx, req.Task, req.Since, func(ctx context.Context, sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where("EXISTS (?)", r.trainingConsent(ctx))
	})
}

func (r *repository) weeklyTotals(
	ctx context.Context,
	task aicorrection.Task,
	since int64,
	scope func(context.Context, *bun.SelectQuery) *bun.SelectQuery,
) ([]aicorrection.WeekTotal, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]aicorrection.WeekTotal, error) {
		cols := buncolgen.CorrectionColumns
		totals := make([]aicorrection.WeekTotal, 0, aicorrection.TrendWeeks)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*aicorrection.Correction)(nil)).
			ColumnExpr(cols.ExtractionProviderID.Expr("{} AS provider_id")).
			ColumnExpr(
				cols.CapturedAt.Expr("{} - (({} - ?) % ?) AS week_start"),
				firstMondayEpoch,
				secondsPerWeek,
			).
			ColumnExpr(buncolgen.Count("corrections")).
			ColumnExpr(cols.ScoredCount.Expr("COALESCE(SUM({}), 0) AS scored")).
			ColumnExpr(cols.CorrectCount.Expr("COALESCE(SUM({}), 0) AS correct")).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scope(ctx, sq).
					Where(cols.Task.Eq(), task).
					Where(cols.CapturedAt.Gte(), since).
					Where(cols.ExtractionProviderID.IsNotNull())
			}).
			GroupExpr("provider_id, week_start").
			Scan(ctx, &totals)
		if err != nil {
			r.l.Error("failed to total ai corrections by week and provider", zap.Error(err))

			return nil, fmt.Errorf("total ai corrections by week and provider: %w", err)
		}

		return totals, nil
	})
}

func (r *repository) trainingConsent(ctx context.Context) *bun.SelectQuery {
	cols := buncolgen.CorrectionColumns
	consent := buncolgen.AgentControlColumns

	return r.db.DBForContext(ctx).
		NewSelect().
		TableExpr(buncolgen.AgentControlTable.Name + " AS " + buncolgen.AgentControlTable.Alias).
		ColumnExpr("1").
		Where(consent.OrganizationID.EqColumn(cols.OrganizationID)).
		Where(consent.BusinessUnitID.EqColumn(cols.BusinessUnitID)).
		Where(consent.AITrainingConsent.IsTrue())
}

func (r *repository) trainable(
	ctx context.Context,
	sq *bun.SelectQuery,
	task aicorrection.Task,
	capturedFrom, capturedTo int64,
) *bun.SelectQuery {
	cols := buncolgen.CorrectionColumns
	caseCols := buncolgen.ExtractionCaseColumns
	promoted := r.db.DBForContext(ctx).NewSelect().
		Model((*extractioneval.ExtractionCase)(nil)).
		ColumnExpr("1").
		Where(caseCols.SourceCorrectionID.EqColumn(cols.ID)).
		Where(caseCols.OrganizationID.EqColumn(cols.OrganizationID)).
		Where(caseCols.BusinessUnitID.EqColumn(cols.BusinessUnitID))

	return sq.Where(cols.Task.Eq(), task).
		Where(cols.CapturedAt.Gte(), capturedFrom).
		Where(cols.CapturedAt.Lt(), capturedTo).
		Where(cols.DocumentID.IsNotNull()).
		Where(cols.ScoredCount.Gt(), 0).
		Where("NOT EXISTS (?)", promoted)
}

func (r *repository) CountTrainable(
	ctx context.Context,
	req *repositories.CountTrainableAICorrectionsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := buncolgen.CorrectionColumns
		capPerOrganization := req.PerOrganizationCap
		if capPerOrganization <= 0 {
			return 0, errortypes.NewValidationError(
				"perOrganizationCap",
				errortypes.ErrInvalid,
				"The per-organization cap must be positive",
			)
		}

		db := r.db.DBForContext(ctx)
		perOrganization := db.NewSelect().
			Model((*aicorrection.Correction)(nil)).
			ColumnExpr("COUNT(*) AS examples").
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return r.trainable(ctx, sq, req.Task, req.CapturedFrom, req.CapturedTo).
					Where("EXISTS (?)", r.trainingConsent(ctx))
			}).
			GroupExpr(cols.OrganizationID.Qualified() + ", " + cols.BusinessUnitID.Qualified())

		var total int
		err := db.NewSelect().
			TableExpr("(?) AS per_organization", perOrganization).
			ColumnExpr("COALESCE(SUM(LEAST(per_organization.examples, ?)), 0)", capPerOrganization).
			Scan(ctx, &total)
		if err != nil {
			r.l.Error("failed to count trainable ai corrections", zap.Error(err))

			return 0, fmt.Errorf("count trainable ai corrections: %w", err)
		}

		return total, nil
	})
}

func (r *repository) ListForAccuracy(
	ctx context.Context,
	req repositories.ListAICorrectionsForAccuracyRequest,
) ([]*aicorrection.Correction, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*aicorrection.Correction, error) {
		cols := buncolgen.CorrectionColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultAccuracyLimit
		}
		limit = min(limit, maxAccuracyLimit)

		entities := make([]*aicorrection.Correction, 0, min(limit, defaultAccuracyLimit))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Column(
				cols.ID.Bare(),
				cols.DocumentKind.Bare(),
				cols.ExtractionModel.Bare(),
				cols.ExtractionProviderID.Bare(),
				cols.FieldResults.Bare(),
				cols.ScoredCount.Bare(),
				cols.CorrectCount.Bare(),
				cols.CorrectedCount.Bare(),
				cols.MissedCount.Bare(),
				cols.UnconfirmedCount.Bare(),
				cols.UnscoredCount.Bare(),
				cols.CapturedAt.Bare(),
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.CorrectionScopeTenant(sq, req.TenantInfo).
					Where(cols.Task.Eq(), req.Task).
					Where(cols.CapturedAt.Gte(), req.Since)
			}).
			Order(cols.CapturedAt.OrderDesc(), cols.ID.OrderAsc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list ai corrections for accuracy", zap.Error(err))

			return nil, fmt.Errorf("list ai corrections for accuracy: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListForTraining(
	ctx context.Context,
	req *repositories.ListAICorrectionsForTrainingRequest,
) ([]*aicorrection.Correction, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*aicorrection.Correction, error) {
		cols := buncolgen.CorrectionColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultTrainingLimit
		}
		limit = min(limit, maxTrainingLimit)

		entities := make([]*aicorrection.Correction, 0, limit)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = r.trainable(
					ctx,
					buncolgen.CorrectionScopeTenant(sq, req.TenantInfo),
					req.Task,
					req.CapturedFrom,
					req.CapturedTo,
				)
				if req.BeforeID.IsNotNil() {
					sq = sq.WhereGroup(" AND ", func(cq *bun.SelectQuery) *bun.SelectQuery {
						return cq.Where(cols.CapturedAt.Lt(), req.BeforeCapturedAt).
							WhereGroup(" OR ", func(tq *bun.SelectQuery) *bun.SelectQuery {
								return tq.Where(cols.CapturedAt.Eq(), req.BeforeCapturedAt).
									Where(cols.ID.Lt(), req.BeforeID)
							})
					})
				}
				return sq
			}).
			Order(cols.CapturedAt.OrderDesc(), cols.ID.OrderDesc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list ai corrections for training", zap.Error(err))

			return nil, fmt.Errorf("list ai corrections for training: %w", err)
		}

		return entities, nil
	})
}
