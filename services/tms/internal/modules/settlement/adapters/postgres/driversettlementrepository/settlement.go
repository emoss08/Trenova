package driversettlementrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driversettlementports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	settlementErrors     = dberror.Mapping{Entity: "DriverSettlement"}
	settlementLineErrors = dberror.Mapping{Entity: "DriverSettlementLine"}
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type settlementRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSettlement(p Params) driversettlementports.DriverSettlementRepository {
	return &settlementRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.driver-settlement-repository"),
	}
}

func settlementSearch(s *driversettlementdb.SettlementsTable, query string) keel.Pred {
	if query == "" {
		return nil
	}

	pattern := "%" + stringutils.EscapeLikePattern(query) + "%"
	w := s.Worker()

	return keel.Or(
		s.SettlementNumber.ILike(pattern),
		s.PayProfileName.ILike(pattern),
		keel.ConcatWS(" ", w.FirstName, w.LastName).ILike(pattern),
	)
}

func (r *settlementRepository) List(
	ctx context.Context,
	req *driversettlementports.ListDriverSettlementsRequest,
) (*pagination.ListResult[*driversettlement.Settlement], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*driversettlement.Settlement], error) {
			s := driversettlementdb.Settlements
			q := s.Where(
				settlementSearch(s, req.Filter.Query),
				when(!req.WorkerID.IsNil(), s.WorkerID.Eq(req.WorkerID)),
				when(!req.BatchID.IsNil(), s.BatchID.Eq(req.BatchID)),
				when(req.Status != "", s.Status.Eq(req.Status)),
				when(len(req.Statuses) > 0, s.Status.In(req.Statuses...)),
				hasExceptions(s, req.HasExceptions),
			).Load(s.Worker()).Order(s.PeriodEnd.Desc(), s.CreatedAt.Desc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list driver settlements: %w", err)
			}

			return result, nil
		},
	)
}

func hasExceptions(s *driversettlementdb.SettlementsTable, value *bool) keel.Pred {
	if value == nil {
		return nil
	}

	return s.HasExceptions.Eq(*value)
}

func (r *settlementRepository) ListConnection(
	ctx context.Context,
	req *driversettlementports.ListDriverSettlementConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.Settlement], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*driversettlement.Settlement], error) {
			s := driversettlementdb.Settlements
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				s.Query(),
				&listquery.Request[driversettlement.Settlement]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*driversettlement.Settlement],
					) *keel.Query[*driversettlement.Settlement] {
						return q.Load(s.Worker())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list driver settlements", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *settlementRepository) GetByID(
	ctx context.Context,
	req driversettlementports.GetDriverSettlementByIDRequest,
) (*driversettlement.Settlement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driversettlement.Settlement, error) {
		s := driversettlementdb.Settlements
		q := s.Where(s.ID.Eq(req.ID)).Load(s.Worker())
		if req.IncludeLines {
			l := s.Lines()
			q = q.Load(l.Order(l.LineNumber.Asc()))
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, settlementErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *settlementRepository) GetLatestForWorker(
	ctx context.Context,
	req driversettlementports.GetLatestSettlementForWorkerRequest,
) (*driversettlement.Settlement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driversettlement.Settlement, error) {
		s := driversettlementdb.Settlements
		entity, err := s.Where(
			s.WorkerID.Eq(req.WorkerID),
			s.Status.Ne(driversettlement.StatusVoided),
		).Order(s.PeriodEnd.Desc()).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, settlementErrors.Map(err, req.WorkerID.String())
		}

		return entity, nil
	})
}

func (r *settlementRepository) GetOpenDraftForWorker(
	ctx context.Context,
	req driversettlementports.GetOpenDraftForWorkerRequest,
) (*driversettlement.Settlement, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driversettlement.Settlement, error) {
		s := driversettlementdb.Settlements
		entity, err := s.Where(
			s.WorkerID.Eq(req.WorkerID),
			s.Status.Eq(driversettlement.StatusDraft),
		).Order(s.PeriodEnd.Desc()).First(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, settlementErrors.Map(err, req.WorkerID.String())
		}

		return entity, nil
	})
}

func (r *settlementRepository) GetWorkspaceCounts(
	ctx context.Context,
	req *driversettlementports.GetWorkspaceCountsRequest,
) (*driversettlementports.SettlementWorkspaceCounts, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*driversettlementports.SettlementWorkspaceCounts, error) {
			s := driversettlementdb.Settlements
			notVoided := s.Status.Ne(driversettlement.StatusVoided)
			counts, err := keel.IntoOne[driversettlementports.SettlementWorkspaceCounts](
				ctx,
				r.db.Keel(ctx),
				s.Pick(
					keel.Count().Filter(s.Status.Eq(driversettlement.StatusDraft)),
					keel.Count().Filter(s.Status.Eq(driversettlement.StatusPendingApproval)),
					keel.Count().Filter(s.Status.Eq(driversettlement.StatusApproved)),
					keel.Count().Filter(s.Status.Eq(driversettlement.StatusPosted)),
					keel.Count().Filter(s.Status.Eq(driversettlement.StatusPaid)),
					keel.Count().Filter(keel.And(
						s.HasExceptions.IsTrue(),
						s.Status.NotIn(driversettlement.StatusVoided, driversettlement.StatusPaid),
					)),
					coalesceZero(s.NetPayMinor.Sum().Filter(notVoided)),
					coalesceZero(s.GrossEarningsMinor.Sum().Filter(notVoided)),
				).Where(
					s.PeriodStart.Eq(req.PeriodStart),
					s.PeriodEnd.Eq(req.PeriodEnd),
				),
			)
			if err != nil {
				return nil, fmt.Errorf("get settlement workspace counts: %w", err)
			}

			return &counts, nil
		},
	)
}

func (r *settlementRepository) ListTrailingNetPay(
	ctx context.Context,
	req *driversettlementports.ListTrailingNetPayRequest,
) ([]int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]int64, error) {
		if req.Limit <= 0 {
			return []int64{}, nil
		}

		s := driversettlementdb.Settlements
		nets, err := keel.Into[int64](
			ctx,
			r.db.Keel(ctx),
			s.Pick(s.NetPayMinor).
				Where(
					s.WorkerID.Eq(req.WorkerID),
					s.Status.Ne(driversettlement.StatusVoided),
					s.PeriodEnd.Lt(req.BeforeDate),
				).
				Order(s.PeriodEnd.Desc()).
				Limit(req.Limit),
		)
		if err != nil {
			return nil, fmt.Errorf("list trailing net pay: %w", err)
		}

		return nets, nil
	})
}

func (r *settlementRepository) ExistsForWorkerPeriod(
	ctx context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
	periodStart, periodEnd int64,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		s := driversettlementdb.Settlements
		exists, err := s.Where(
			s.WorkerID.Eq(workerID),
			s.PeriodStart.Eq(periodStart),
			s.PeriodEnd.Eq(periodEnd),
			s.Status.Ne(driversettlement.StatusVoided),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("check settlement exists for period: %w", err)
		}

		return exists, nil
	})
}

func (r *settlementRepository) Create(
	ctx context.Context,
	entity *driversettlement.Settlement,
) (*driversettlement.Settlement, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driversettlement.Settlement, error) {
		db := r.db.Keel(ctx)
		if entity.ID.IsNil() {
			entity.ID = pulid.MustNew("dstl_")
		}
		assignLineFields(entity)

		if err := driversettlementdb.Settlements.Create(ctx, db, entity); err != nil {
			return nil, settlementErrors.Map(err, entity.ID.String())
		}
		if err := driversettlementdb.SettlementLines.CreateMany(ctx, db, entity.Lines); err != nil {
			return nil, fmt.Errorf(
				"create driver settlement lines: %w",
				settlementLineErrors.Map(err, entity.ID.String()),
			)
		}

		return r.GetByID(ctx, driversettlementports.GetDriverSettlementByIDRequest{
			ID:           entity.ID,
			TenantInfo:   tenantOf(entity.OrganizationID, entity.BusinessUnitID),
			IncludeLines: true,
		})
	})
}

func (r *settlementRepository) Update(
	ctx context.Context,
	entity *driversettlement.Settlement,
) (*driversettlement.Settlement, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driversettlement.Settlement, error) {
		s := driversettlementdb.Settlements
		if err := s.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			s.Status,
			s.BatchID,
			s.PayProfileID,
			s.PayProfileName,
			s.Classification,
			s.PayDate,
			s.GrossEarningsMinor,
			s.ReimbursementsMinor,
			s.DeductionsMinor,
			s.CarryForwardInMinor,
			s.CarryForwardOutMinor,
			s.NetPayMinor,
			s.TotalMiles,
			s.ShipmentCount,
			s.HasExceptions,
			s.Exceptions,
			s.Notes,
			s.SubmittedByID,
			s.SubmittedAt,
			s.ApprovedByID,
			s.ApprovedAt,
			s.PostedByID,
			s.PostedAt,
			s.PostedJournalBatchID,
			s.PaidAt,
			s.PaidByID,
			s.PaymentMethod,
			s.PaymentReference,
			s.VoidedByID,
			s.VoidedAt,
			s.VoidReason,
			s.VoidJournalBatchID,
		); err != nil {
			return nil, settlementErrors.Map(err, entity.ID.String())
		}

		return r.GetByID(ctx, driversettlementports.GetDriverSettlementByIDRequest{
			ID:           entity.ID,
			TenantInfo:   tenantOf(entity.OrganizationID, entity.BusinessUnitID),
			IncludeLines: true,
		})
	})
}

func (r *settlementRepository) ReplaceLines(
	ctx context.Context,
	entity *driversettlement.Settlement,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		assignLineFields(entity)

		l := driversettlementdb.SettlementLines
		if _, err := l.Where(l.SettlementID.Eq(entity.ID)).Delete(ctx, db); err != nil {
			return fmt.Errorf("delete driver settlement lines: %w", err)
		}
		if err := l.CreateMany(ctx, db, entity.Lines); err != nil {
			return fmt.Errorf(
				"insert driver settlement lines: %w",
				settlementLineErrors.Map(err, entity.ID.String()),
			)
		}

		return nil
	})
}

func assignLineFields(entity *driversettlement.Settlement) {
	for idx, line := range entity.Lines {
		if line == nil {
			continue
		}
		line.OrganizationID = entity.OrganizationID
		line.BusinessUnitID = entity.BusinessUnitID
		line.SettlementID = entity.ID
		line.LineNumber = idx + 1
	}
}
