package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	defaultSnapshotHistoryLimit = 12
	maxSnapshotHistoryLimit     = 100
	defaultRecomputeLimit       = 500
	maxRecomputeLimit           = 5000
	defaultReviewQueueLimit     = 100
	maxReviewQueueLimit         = 1000
	snapshotEntityName          = "CarrierIntelSnapshot"
	rankedSnapshotAlias         = "ranked"
	rankedSnapshotRankName      = "rn"
)

var rankedSnapshotRank = buncolgen.NewColumn(rankedSnapshotRankName, rankedSnapshotAlias)

type snapshotRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSnapshotRepository(p Params) repositories.CarrierIntelSnapshotRepository {
	return &snapshotRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-snapshot-repository"),
	}
}

func (r *snapshotRepository) GetCurrent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	subject repositories.CarrierIntelSubjectRef,
) (*carrierintel.CarrierIntelSnapshot, error) {
	cols := buncolgen.CarrierIntelSnapshotColumns
	entity := new(carrierintel.CarrierIntelSnapshot)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenant(sq, tenantInfo).
				Where(cols.SubjectType.Eq(), subject.SubjectType).
				Where(cols.SubjectID.Eq(), subject.SubjectID).
				Where(cols.IsCurrent.IsTrue())
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil snapshot represents a subject that was never looked up
		}
		r.l.Error("failed to get current carrier intel snapshot", zap.Error(err))
		return nil, fmt.Errorf("get current carrier intel snapshot: %w", err)
	}

	return entity, nil
}

func (r *snapshotRepository) GetCurrentByCarrierIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	if len(carrierIDs) == 0 {
		return []*carrierintel.CarrierIntelSnapshot{}, nil
	}

	cols := buncolgen.CarrierIntelSnapshotColumns
	entities := make([]*carrierintel.CarrierIntelSnapshot, 0, len(carrierIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenant(sq, tenantInfo).
				Where(cols.IsCurrent.IsTrue()).
				Where(cols.CarrierID.In(), bun.List(carrierIDs))
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get current carrier intel snapshots by carrier", zap.Error(err))
		return nil, fmt.Errorf("get current carrier intel snapshots by carrier: %w", err)
	}

	return entities, nil
}

func (r *snapshotRepository) GetCurrentBySubjectIDs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	subjectType carrierintel.SubjectType,
	subjectIDs []string,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	if len(subjectIDs) == 0 {
		return []*carrierintel.CarrierIntelSnapshot{}, nil
	}

	cols := buncolgen.CarrierIntelSnapshotColumns
	entities := make([]*carrierintel.CarrierIntelSnapshot, 0, len(subjectIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenant(sq, tenantInfo).
				Where(cols.IsCurrent.IsTrue()).
				Where(cols.SubjectType.Eq(), subjectType).
				Where(cols.SubjectID.In(), bun.List(subjectIDs))
		}).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to get current carrier intel snapshots by subject", zap.Error(err))
		return nil, fmt.Errorf("get current carrier intel snapshots by subject: %w", err)
	}

	return entities, nil
}

func (r *snapshotRepository) GetLatestFullByDOT(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	dotNumber string,
	since int64,
) (*carrierintel.CarrierIntelSnapshot, error) {
	cols := buncolgen.CarrierIntelSnapshotColumns
	entity := new(carrierintel.CarrierIntelSnapshot)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenant(sq, tenantInfo).
				Where(cols.DOTNumber.Eq(), dotNumber).
				Where(cols.Depth.Eq(), carrierintel.LookupDepthFull).
				Where(cols.FetchedAt.Gte(), since)
		}).
		Order(cols.FetchedAt.OrderDesc()).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // nil snapshot means no fresh full profile exists
		}
		r.l.Error("failed to get latest full carrier intel snapshot", zap.Error(err))
		return nil, fmt.Errorf("get latest full carrier intel snapshot: %w", err)
	}

	return entity, nil
}

func (r *snapshotRepository) ListHistory(
	ctx context.Context,
	req *repositories.ListCarrierIntelSnapshotHistoryRequest,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	cols := buncolgen.CarrierIntelSnapshotColumns
	limit := intutils.Clamp(
		intutils.WithDefault(max(req.Limit, 0), defaultSnapshotHistoryLimit),
		1,
		maxSnapshotHistoryLimit,
	)

	entities := make([]*carrierintel.CarrierIntelSnapshot, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenant(sq, req.TenantInfo).
				Where(cols.SubjectType.Eq(), req.SubjectType).
				Where(cols.SubjectID.Eq(), req.SubjectID)
		}).
		Order(cols.FetchedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list carrier intel snapshot history", zap.Error(err))
		return nil, fmt.Errorf("list carrier intel snapshot history: %w", err)
	}

	return entities, nil
}

func (r *snapshotRepository) InsertCurrent(
	ctx context.Context,
	entity *carrierintel.CarrierIntelSnapshot,
) (*carrierintel.CarrierIntelSnapshot, error) {
	cols := buncolgen.CarrierIntelSnapshotColumns
	tenantInfo := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, _ bun.Tx) error {
		dba := r.db.DBForContext(c)
		if _, uErr := dba.NewUpdate().
			Model((*carrierintel.CarrierIntelSnapshot)(nil)).
			Set(cols.IsCurrent.Set(), false).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.CarrierIntelSnapshotScopeTenantUpdate(uq, tenantInfo).
					Where(cols.SubjectType.Eq(), entity.SubjectType).
					Where(cols.SubjectID.Eq(), entity.SubjectID).
					Where(cols.IsCurrent.IsTrue())
			}).
			Exec(c); uErr != nil {
			return fmt.Errorf("retire current carrier intel snapshot: %w", uErr)
		}

		entity.IsCurrent = true
		if _, iErr := dba.NewInsert().
			Model(entity).
			Returning("*").
			Exec(c); iErr != nil {
			return fmt.Errorf("insert carrier intel snapshot: %w", iErr)
		}

		return nil
	})
	if err != nil {
		r.l.Error("failed to insert current carrier intel snapshot", zap.Error(err))
		return nil, dberror.MapRetryableTransactionError(
			err,
			"Carrier intelligence snapshot is busy. Retry the request.",
		)
	}

	return entity, nil
}

func (r *snapshotRepository) UpdateEvaluation(
	ctx context.Context,
	entity *carrierintel.CarrierIntelSnapshot,
) error {
	cols := buncolgen.CarrierIntelSnapshotColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Column(
			cols.Findings.Bare(),
			cols.BlockingCodes.Bare(),
			cols.AdvisoryCodes.Bare(),
			cols.RiskLevel.Bare(),
			cols.ReviewState.Bare(),
			cols.PolicyVersion.Bare(),
			cols.ConfirmedAt.Bare(),
			cols.UpdatedAt.Bare(),
		).
		WherePK().
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to update carrier intel snapshot evaluation", zap.Error(err))
		return fmt.Errorf("update carrier intel snapshot evaluation: %w", err)
	}

	return nil
}

func (r *snapshotRepository) TouchConfirmed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	snapshotIDs []pulid.ID,
	confirmedAt int64,
) error {
	if len(snapshotIDs) == 0 {
		return nil
	}

	cols := buncolgen.CarrierIntelSnapshotColumns
	_, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*carrierintel.CarrierIntelSnapshot)(nil)).
		Set(cols.ConfirmedAt.Set(), confirmedAt).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenantUpdate(uq, tenantInfo).
				Where(cols.ID.In(), bun.List(snapshotIDs))
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to touch carrier intel snapshots", zap.Error(err))
		return fmt.Errorf("touch carrier intel snapshots: %w", err)
	}

	return nil
}

func (r *snapshotRepository) MarkReviewed(
	ctx context.Context,
	req *repositories.MarkSnapshotReviewedRequest,
) error {
	cols := buncolgen.CarrierIntelSnapshotColumns
	var note any
	if req.Note != "" {
		note = req.Note
	}

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*carrierintel.CarrierIntelSnapshot)(nil)).
		Set(cols.ReviewState.Set(), carrierintel.ReviewStateReviewed).
		Set(cols.ReviewedByID.Set(), req.UserID).
		Set(cols.ReviewedAt.Set(), req.ReviewedAt).
		Set(cols.ReviewNote.Set(), note).
		Set(cols.UpdatedAt.Set(), req.ReviewedAt).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.SubjectType.Eq(), req.SubjectType).
				Where(cols.SubjectID.Eq(), req.SubjectID).
				Where(cols.IsCurrent.IsTrue())
		}).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to mark carrier intel snapshot reviewed", zap.Error(err))
		return fmt.Errorf("mark carrier intel snapshot reviewed: %w", err)
	}

	return dberror.CheckRowsAffected(
		results,
		snapshotEntityName,
		carrierintel.SubjectKey(req.SubjectType, req.SubjectID),
	)
}

func (r *snapshotRepository) ListForRecompute(
	ctx context.Context,
	req *repositories.ListSnapshotsForRecomputeRequest,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	cols := buncolgen.CarrierIntelSnapshotColumns
	limit := intutils.Clamp(
		intutils.WithDefault(max(req.Limit, 0), defaultRecomputeLimit),
		1,
		maxRecomputeLimit,
	)
	versionPredicate := cols.PolicyVersion.Lt()
	if req.IncludeUpToDateAt {
		versionPredicate = cols.PolicyVersion.Lte()
	}

	entities := make([]*carrierintel.CarrierIntelSnapshot, 0, limit)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenant(sq, req.TenantInfo).
				Where(cols.IsCurrent.IsTrue()).
				Where(versionPredicate, req.BelowVersion)
		}).
		Order(cols.ID.OrderAsc()).
		Limit(limit)
	if !req.AfterID.IsNil() {
		q = q.Where(cols.ID.Gt(), req.AfterID)
	}

	if err := q.Scan(ctx); err != nil {
		r.l.Error("failed to list carrier intel snapshots for recompute", zap.Error(err))
		return nil, fmt.Errorf("list carrier intel snapshots for recompute: %w", err)
	}

	return entities, nil
}

func (r *snapshotRepository) ListReviewQueue(
	ctx context.Context,
	req *repositories.ListCarrierIntelReviewQueueRequest,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	cols := buncolgen.CarrierIntelSnapshotColumns
	limit := intutils.Clamp(
		intutils.WithDefault(max(req.Limit, 0), defaultReviewQueueLimit),
		1,
		maxReviewQueueLimit,
	)

	entities := make([]*carrierintel.CarrierIntelSnapshot, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenant(sq, req.TenantInfo).
				Where(cols.IsCurrent.IsTrue()).
				Where(cols.ReviewState.Eq(), carrierintel.ReviewStateNeedsReview)
		}).
		Order(cols.FetchedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list carrier intel review queue", zap.Error(err))
		return nil, fmt.Errorf("list carrier intel review queue: %w", err)
	}

	return entities, nil
}

func buildPruneHistoryDelete(
	db bun.IDB,
	tenantInfo pagination.TenantInfo,
	keep int,
) *bun.DeleteQuery {
	cols := buncolgen.CarrierIntelSnapshotColumns

	ranked := db.NewSelect().
		Model((*carrierintel.CarrierIntelSnapshot)(nil)).
		Column(cols.ID.Bare()).
		ColumnExpr(
			buncolgen.Expr(
				"row_number() OVER (PARTITION BY {0}, {1} ORDER BY {2} DESC, {3} DESC) AS ?",
				cols.SubjectType,
				cols.SubjectID,
				cols.FetchedAt,
				cols.ID,
			),
			bun.Ident(rankedSnapshotRankName),
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenant(sq, tenantInfo).
				Where(cols.IsCurrent.IsFalse())
		})

	stale := db.NewSelect().
		TableExpr("(?) AS ?", ranked, bun.Ident(rankedSnapshotAlias)).
		ColumnExpr(cols.ID.WithAlias(rankedSnapshotAlias).Qualified()).
		Where(rankedSnapshotRank.Gt(), keep)

	return db.NewDelete().
		Model((*carrierintel.CarrierIntelSnapshot)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.CarrierIntelSnapshotScopeTenantDelete(dq, tenantInfo).
				Where(cols.IsCurrent.IsFalse()).
				Where(cols.ID.In(), stale)
		})
}

func (r *snapshotRepository) PruneHistory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	keep int,
) (int, error) {
	result, err := buildPruneHistoryDelete(r.db.DBForContext(ctx), tenantInfo, max(keep, 0)).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to prune carrier intel snapshot history", zap.Error(err))
		return 0, fmt.Errorf("prune carrier intel snapshot history: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune carrier intel snapshot history rows affected: %w", err)
	}

	return int(affected), nil
}

func buildCarrierSummaryUpdate(
	db bun.IDB,
	req *repositories.UpdateCarrierIntelSummaryRequest,
) *bun.UpdateQuery {
	cols := buncolgen.CarrierColumns
	var riskLevel any
	if req.RiskLevel != "" {
		riskLevel = req.RiskLevel
	}

	return db.NewUpdate().
		Model((*carrier.Carrier)(nil)).
		Set(cols.IntelRiskLevel.Set(), riskLevel).
		Set(cols.IntelReviewRequired.Set(), req.ReviewRequired).
		Set(cols.IntelBlockingCount.Set(), req.BlockingCount).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.CarrierScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.CarrierID).
				WhereGroup(" AND ", func(inner *bun.UpdateQuery) *bun.UpdateQuery {
					return inner.
						Where(cols.IntelRiskLevel.Expr("{} IS DISTINCT FROM ?"), riskLevel).
						WhereOr(
							cols.IntelReviewRequired.Expr("{} IS DISTINCT FROM ?"),
							req.ReviewRequired,
						).
						WhereOr(
							cols.IntelBlockingCount.Expr("{} IS DISTINCT FROM ?"),
							req.BlockingCount,
						)
				})
		})
}

func (r *snapshotRepository) UpdateCarrierSummary(
	ctx context.Context,
	req *repositories.UpdateCarrierIntelSummaryRequest,
) error {
	if _, err := buildCarrierSummaryUpdate(r.db.DBForContext(ctx), req).Exec(ctx); err != nil {
		r.l.Error("failed to update carrier intel summary", zap.Error(err))
		return fmt.Errorf("update carrier intel summary: %w", err)
	}

	return nil
}
