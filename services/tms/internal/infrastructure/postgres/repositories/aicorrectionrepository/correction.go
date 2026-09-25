package aicorrectionrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultPurgeLimit = 1000
	maxPurgeLimit     = 10000
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
}

func (r *repository) PurgeBefore(
	ctx context.Context,
	req repositories.PurgeAICorrectionsRequest,
) (int64, error) {
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
}
