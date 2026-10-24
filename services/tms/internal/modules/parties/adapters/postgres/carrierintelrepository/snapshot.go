package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
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
)

type snapshotRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSnapshotRepository(p Params) carrierintelports.CarrierIntelSnapshotRepository {
	return &snapshotRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-snapshot-repository"),
	}
}

func (r *snapshotRepository) GetCurrent(
	ctx context.Context,
	_ pagination.TenantInfo,
	subject carrierintelports.CarrierIntelSubjectRef,
) (*carrierintel.CarrierIntelSnapshot, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierIntelSnapshot, error) {
			s := carrierinteldb.CarrierIntelSnapshots
			entity, err := s.Where(
				s.SubjectType.Eq(subject.SubjectType),
				s.SubjectID.Eq(subject.SubjectID),
				s.IsCurrent.IsTrue(),
			).One(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			if err != nil {
				r.l.Error("failed to get current carrier intel snapshot", zap.Error(err))
				return nil, fmt.Errorf("get current carrier intel snapshot: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *snapshotRepository) GetCurrentByCarrierIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	carrierIDs []pulid.ID,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierIntelSnapshot, error) {
			if len(carrierIDs) == 0 {
				return []*carrierintel.CarrierIntelSnapshot{}, nil
			}

			s := carrierinteldb.CarrierIntelSnapshots
			entities, err := s.Where(s.IsCurrent.IsTrue(), s.CarrierID.In(carrierIDs...)).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get current carrier intel snapshots by carrier",
					zap.Error(err),
				)
				return nil, fmt.Errorf("get current carrier intel snapshots by carrier: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *snapshotRepository) GetCurrentBySubjectIDs(
	ctx context.Context,
	_ pagination.TenantInfo,
	subjectType carrierintel.SubjectType,
	subjectIDs []string,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierIntelSnapshot, error) {
			if len(subjectIDs) == 0 {
				return []*carrierintel.CarrierIntelSnapshot{}, nil
			}

			s := carrierinteldb.CarrierIntelSnapshots
			entities, err := s.Where(
				s.IsCurrent.IsTrue(),
				s.SubjectType.Eq(subjectType),
				s.SubjectID.In(subjectIDs...),
			).All(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error(
					"failed to get current carrier intel snapshots by subject",
					zap.Error(err),
				)
				return nil, fmt.Errorf("get current carrier intel snapshots by subject: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *snapshotRepository) GetLatestFullByDOT(
	ctx context.Context,
	_ pagination.TenantInfo,
	dotNumber string,
	since int64,
) (*carrierintel.CarrierIntelSnapshot, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierIntelSnapshot, error) {
			s := carrierinteldb.CarrierIntelSnapshots
			entity, err := s.Where(
				s.DOTNumber.Eq(dotNumber),
				s.Depth.Eq(carrierintel.LookupDepthFull),
				s.DepthFetchedAt.Gte(since),
			).
				Order(s.DepthFetchedAt.Desc()).
				First(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}
			if err != nil {
				r.l.Error("failed to get latest full carrier intel snapshot", zap.Error(err))
				return nil, fmt.Errorf("get latest full carrier intel snapshot: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *snapshotRepository) ListHistory(
	ctx context.Context,
	req *carrierintelports.ListCarrierIntelSnapshotHistoryRequest,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierIntelSnapshot, error) {
			s := carrierinteldb.CarrierIntelSnapshots
			limit := intutils.Clamp(
				intutils.WithDefault(max(req.Limit, 0), defaultSnapshotHistoryLimit),
				1,
				maxSnapshotHistoryLimit,
			)

			entities, err := s.Where(
				s.SubjectType.Eq(req.SubjectType),
				s.SubjectID.Eq(req.SubjectID),
			).
				Order(s.FetchedAt.Desc(), s.ID.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list carrier intel snapshot history", zap.Error(err))
				return nil, fmt.Errorf("list carrier intel snapshot history: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *snapshotRepository) InsertCurrent(
	ctx context.Context,
	entity *carrierintel.CarrierIntelSnapshot,
) (*carrierintel.CarrierIntelSnapshot, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintel.CarrierIntelSnapshot, error) {
			s := carrierinteldb.CarrierIntelSnapshots
			db := r.db.Keel(ctx)

			_, err := s.Where(
				s.SubjectType.Eq(entity.SubjectType),
				s.SubjectID.Eq(entity.SubjectID),
				s.IsCurrent.IsTrue(),
			).Update(ctx, db, s.IsCurrent.To(false))
			if err != nil {
				err = fmt.Errorf("retire current carrier intel snapshot: %w", err)
			} else {
				entity.IsCurrent = true
				if err = s.Create(ctx, db, entity); err != nil {
					err = fmt.Errorf("insert carrier intel snapshot: %w", err)
				}
			}
			if err != nil {
				r.l.Error("failed to insert current carrier intel snapshot", zap.Error(err))
				return nil, dberror.MapRetryableTransactionError(
					err,
					"Carrier intelligence snapshot is busy. Retry the request.",
				)
			}

			return entity, nil
		},
	)
}

func (r *snapshotRepository) UpdateEvaluation(
	ctx context.Context,
	entity *carrierintel.CarrierIntelSnapshot,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := carrierinteldb.CarrierIntelSnapshots
		err := s.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			s.Findings,
			s.BlockingCodes,
			s.AdvisoryCodes,
			s.RiskLevel,
			s.ReviewState,
			s.PolicyVersion,
			s.ConfirmedAt,
			s.FetchedDepth,
			s.DepthFetchedAt,
		)
		if err != nil {
			r.l.Error("failed to update carrier intel snapshot evaluation", zap.Error(err))
			return fmt.Errorf("update carrier intel snapshot evaluation: %w", err)
		}

		return nil
	})
}

func (r *snapshotRepository) TouchConfirmed(
	ctx context.Context,
	_ pagination.TenantInfo,
	snapshotIDs []pulid.ID,
	confirmedAt int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(snapshotIDs) == 0 {
			return nil
		}

		s := carrierinteldb.CarrierIntelSnapshots
		_, err := s.Where(s.ID.In(snapshotIDs...)).Update(
			ctx,
			r.db.Keel(ctx),
			s.ConfirmedAt.To(confirmedAt),
			s.UpdatedAt.ToExpr(s.UpdatedAt),
		)
		if err != nil {
			r.l.Error("failed to touch carrier intel snapshots", zap.Error(err))
			return fmt.Errorf("touch carrier intel snapshots: %w", err)
		}

		return nil
	})
}

func (r *snapshotRepository) MarkReviewed(
	ctx context.Context,
	req *carrierintelports.MarkSnapshotReviewedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := carrierinteldb.CarrierIntelSnapshots
		affected, err := s.Where(
			s.SubjectType.Eq(req.SubjectType),
			s.SubjectID.Eq(req.SubjectID),
			s.IsCurrent.IsTrue(),
		).Update(
			ctx,
			r.db.Keel(ctx),
			s.ReviewState.To(carrierintel.ReviewStateReviewed),
			s.ReviewedByID.To(req.UserID),
			s.ReviewedAt.To(req.ReviewedAt),
			s.ReviewNote.To(req.Note),
			s.UpdatedAt.To(req.ReviewedAt),
		)
		if err != nil {
			r.l.Error("failed to mark carrier intel snapshot reviewed", zap.Error(err))
			return fmt.Errorf("mark carrier intel snapshot reviewed: %w", err)
		}

		return dberror.CheckAffected(
			affected,
			snapshotEntityName,
			carrierintel.SubjectKey(req.SubjectType, req.SubjectID),
		)
	})
}

func (r *snapshotRepository) ListForRecompute(
	ctx context.Context,
	req *carrierintelports.ListSnapshotsForRecomputeRequest,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierIntelSnapshot, error) {
			s := carrierinteldb.CarrierIntelSnapshots
			limit := intutils.Clamp(
				intutils.WithDefault(max(req.Limit, 0), defaultRecomputeLimit),
				1,
				maxRecomputeLimit,
			)
			version := s.PolicyVersion.Lt(req.BelowVersion)
			if req.IncludeUpToDateAt {
				version = s.PolicyVersion.Lte(req.BelowVersion)
			}

			entities, err := s.Where(
				s.IsCurrent.IsTrue(),
				version,
				keel.When(!req.AfterID.IsNil(), s.ID.Gt(req.AfterID)),
			).
				Order(s.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list carrier intel snapshots for recompute", zap.Error(err))
				return nil, fmt.Errorf("list carrier intel snapshots for recompute: %w", err)
			}

			return entities, nil
		},
	)
}

func (r *snapshotRepository) ListReviewQueue(
	ctx context.Context,
	req *carrierintelports.ListCarrierIntelReviewQueueRequest,
) ([]*carrierintel.CarrierIntelSnapshot, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*carrierintel.CarrierIntelSnapshot, error) {
			s := carrierinteldb.CarrierIntelSnapshots
			limit := intutils.Clamp(
				intutils.WithDefault(max(req.Limit, 0), defaultReviewQueueLimit),
				1,
				maxReviewQueueLimit,
			)

			entities, err := s.Where(
				s.IsCurrent.IsTrue(),
				s.ReviewState.Eq(carrierintel.ReviewStateNeedsReview),
			).
				Order(s.FetchedAt.Desc(), s.ID.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to list carrier intel review queue", zap.Error(err))
				return nil, fmt.Errorf("list carrier intel review queue: %w", err)
			}

			return entities, nil
		},
	)
}

func pruneHistoryQuery(keep int) *keel.Query[carrierintel.CarrierIntelSnapshot] {
	s := carrierinteldb.CarrierIntelSnapshots
	h := s.As("history")
	ranked := keel.CTE("ranked", []string{"id", "rn"}, h.Pick(
		h.ID,
		keel.RowNumber().
			Over([]keel.Expr{h.SubjectType, h.SubjectID}, h.FetchedAt.Desc(), h.ID.Desc()),
	).Where(h.IsCurrent.IsFalse()))

	return s.With(ranked).
		Join(ranked, ranked.Col("id").EqCol(s.ID)).
		Where(s.IsCurrent.IsFalse(), ranked.Col("rn").Gt(keep))
}

func (r *snapshotRepository) PruneHistory(
	ctx context.Context,
	_ pagination.TenantInfo,
	keep int,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		affected, err := pruneHistoryQuery(max(keep, 0)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to prune carrier intel snapshot history", zap.Error(err))
			return 0, fmt.Errorf("prune carrier intel snapshot history: %w", err)
		}

		return int(affected), nil
	})
}

func carrierSummaryUpdate(
	req *carrierintelports.UpdateCarrierIntelSummaryRequest,
) (*keel.Query[carrier.Carrier], []keel.Assignment) {
	c := carrierdb.Carriers
	q := c.Where(
		c.ID.Eq(req.CarrierID),
		keel.Or(
			c.IntelRiskLevel.IsDistinctFrom(req.RiskLevel),
			keel.Raw("? IS DISTINCT FROM ?", c.IntelReviewRequired, req.ReviewRequired),
			c.IntelBlockingCount.IsDistinctFrom(int64(req.BlockingCount)),
		),
	)

	return q, []keel.Assignment{
		c.IntelRiskLevel.To(req.RiskLevel),
		c.IntelReviewRequired.To(req.ReviewRequired),
		c.IntelBlockingCount.To(int64(req.BlockingCount)),
		c.Version.ToExpr(c.Version),
		c.UpdatedAt.ToExpr(c.UpdatedAt),
	}
}

func (r *snapshotRepository) UpdateCarrierSummary(
	ctx context.Context,
	req *carrierintelports.UpdateCarrierIntelSummaryRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		q, sets := carrierSummaryUpdate(req)
		if _, err := q.Update(ctx, r.db.Keel(ctx), sets...); err != nil {
			r.l.Error("failed to update carrier intel summary", zap.Error(err))
			return fmt.Errorf("update carrier intel summary: %w", err)
		}

		return nil
	})
}
