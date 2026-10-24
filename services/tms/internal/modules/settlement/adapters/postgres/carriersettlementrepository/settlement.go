package carriersettlementrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var settlementErrors = dberror.Mapping{Entity: "CarrierSettlement"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type settlementRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSettlement(p Params) carriersettlementports.CarrierSettlementRepository {
	return &settlementRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-settlement-repository"),
	}
}

func (r *settlementRepository) List(
	ctx context.Context,
	req *carriersettlementports.ListCarrierSettlementsRequest,
) (*pagination.ListResult[*carriersettlement.CarrierSettlement], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*carriersettlement.CarrierSettlement], error) {
			s := carriersettlementdb.CarrierSettlements
			q := s.Where(
				when(req.Filter.Query != "", s.SettlementNumber.ContainsFold(req.Filter.Query)),
				when(!req.CarrierID.IsNil(), s.CarrierID.Eq(req.CarrierID)),
				when(!req.BatchID.IsNil(), s.BatchID.Eq(req.BatchID)),
				when(req.Status != "", s.Status.Eq(req.Status)),
				when(len(req.Statuses) > 0, s.Status.In(req.Statuses...)),
			).Load(s.Carrier()).Order(s.PeriodEnd.Desc(), s.CreatedAt.Desc())

			result, err := listquery.Offset(ctx, r.db.Keel(ctx), q, req.Filter.Pagination)
			if err != nil {
				return nil, fmt.Errorf("list carrier settlements: %w", err)
			}

			return result, nil
		},
	)
}

func (r *settlementRepository) ListConnection(
	ctx context.Context,
	req *carriersettlementports.ListCarrierSettlementConnectionRequest,
) (*pagination.CursorListResult[*carriersettlement.CarrierSettlement], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*carriersettlement.CarrierSettlement], error) {
			s := carriersettlementdb.CarrierSettlements
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				s.Query(),
				&listquery.Request[carriersettlement.CarrierSettlement]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[carriersettlement.CarrierSettlement],
					) *keel.Query[carriersettlement.CarrierSettlement] {
						return q.Load(s.Carrier())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list carrier settlements", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *settlementRepository) GetByID(
	ctx context.Context,
	req carriersettlementports.GetCarrierSettlementByIDRequest,
) (*carriersettlement.CarrierSettlement, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carriersettlement.CarrierSettlement, error) {
			s := carriersettlementdb.CarrierSettlements
			q := s.Where(s.ID.Eq(req.ID)).Load(s.Carrier())
			if req.IncludeLines {
				l := s.Lines()
				q = q.Load(l.Order(l.LineNumber.Asc()))
			}

			entity, err := q.One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, settlementErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *settlementRepository) GetWorkspaceCounts(
	ctx context.Context,
	req *carriersettlementports.GetCarrierSettlementWorkspaceCountsRequest,
) (*carriersettlementports.CarrierSettlementWorkspaceCounts, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carriersettlementports.CarrierSettlementWorkspaceCounts, error) {
			s := carriersettlementdb.CarrierSettlements
			notVoided := s.Status.Ne(carriersettlement.StatusVoided)
			counts, err := keel.IntoOne[carriersettlementports.CarrierSettlementWorkspaceCounts](
				ctx,
				r.db.Keel(ctx),
				s.Pick(
					keel.Count().Filter(s.Status.Eq(carriersettlement.StatusDraft)),
					keel.Count().Filter(s.Status.Eq(carriersettlement.StatusPendingApproval)),
					keel.Count().Filter(s.Status.Eq(carriersettlement.StatusApproved)),
					keel.Count().Filter(s.Status.Eq(carriersettlement.StatusPosted)),
					keel.Count().Filter(s.Status.Eq(carriersettlement.StatusPaid)),
					coalesceZero(s.NetPayableMinor.Sum().Filter(notVoided)),
					coalesceZero(s.GrossCostMinor.Sum().Filter(notVoided)),
				).Where(
					s.PeriodStart.Eq(req.PeriodStart),
					s.PeriodEnd.Eq(req.PeriodEnd),
				),
			)
			if err != nil {
				return nil, fmt.Errorf("get carrier settlement workspace counts: %w", err)
			}

			return &counts, nil
		},
	)
}

func (r *settlementRepository) ExistsForCarrierPeriod(
	ctx context.Context,
	_ pagination.TenantInfo,
	carrierID pulid.ID,
	periodStart, periodEnd int64,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		s := carriersettlementdb.CarrierSettlements
		exists, err := s.Where(
			s.CarrierID.Eq(carrierID),
			s.PeriodStart.Eq(periodStart),
			s.PeriodEnd.Eq(periodEnd),
			s.Status.Ne(carriersettlement.StatusVoided),
		).Exists(ctx, r.db.Keel(ctx))
		if err != nil {
			return false, fmt.Errorf("check carrier settlement exists for period: %w", err)
		}

		return exists, nil
	})
}

func (r *settlementRepository) Create(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
) (*carriersettlement.CarrierSettlement, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carriersettlement.CarrierSettlement, error) {
			db := r.db.Keel(ctx)
			if entity.ID.IsNil() {
				entity.ID = pulid.MustNew("carstl_")
			}
			assignLineFields(entity)

			if err := carriersettlementdb.CarrierSettlements.Create(ctx, db, entity); err != nil {
				return nil, fmt.Errorf(
					"create carrier settlement: %w",
					settlementErrors.Map(err, entity.ID.String()),
				)
			}
			if err := carriersettlementdb.CarrierSettlementLines.CreateMany(
				ctx,
				db,
				entity.Lines,
			); err != nil {
				return nil, fmt.Errorf("create carrier settlement lines: %w", err)
			}

			return r.GetByID(ctx, carriersettlementports.GetCarrierSettlementByIDRequest{
				ID:           entity.ID,
				TenantInfo:   tenantOf(entity.OrganizationID, entity.BusinessUnitID),
				IncludeLines: true,
			})
		},
	)
}

func (r *settlementRepository) Update(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
) (*carriersettlement.CarrierSettlement, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carriersettlement.CarrierSettlement, error) {
			s := carriersettlementdb.CarrierSettlements
			if err := s.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
				s.Status,
				s.BatchID,
				s.PayDate,
				s.GrossCostMinor,
				s.AdjustmentsMinor,
				s.NetPayableMinor,
				s.ShipmentCount,
				s.CurrencyCode,
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
				s.PaidJournalBatchID,
				s.VoidedByID,
				s.VoidedAt,
				s.VoidReason,
				s.VoidJournalBatchID,
			); err != nil {
				return nil, settlementErrors.Map(err, entity.ID.String())
			}

			return r.GetByID(ctx, carriersettlementports.GetCarrierSettlementByIDRequest{
				ID:           entity.ID,
				TenantInfo:   tenantOf(entity.OrganizationID, entity.BusinessUnitID),
				IncludeLines: true,
			})
		},
	)
}

func (r *settlementRepository) ReplaceLines(
	ctx context.Context,
	entity *carriersettlement.CarrierSettlement,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		db := r.db.Keel(ctx)
		assignLineFields(entity)

		l := carriersettlementdb.CarrierSettlementLines
		if _, err := l.Where(l.SettlementID.Eq(entity.ID)).Delete(ctx, db); err != nil {
			return fmt.Errorf("delete carrier settlement lines: %w", err)
		}
		if err := l.CreateMany(ctx, db, entity.Lines); err != nil {
			return fmt.Errorf("insert carrier settlement lines: %w", err)
		}

		return nil
	})
}

func assignLineFields(entity *carriersettlement.CarrierSettlement) {
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
