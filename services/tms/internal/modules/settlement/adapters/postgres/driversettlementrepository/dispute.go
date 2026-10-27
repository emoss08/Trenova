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
	"go.uber.org/zap"
)

const (
	defaultWorkerDisputeLimit = 50
	maxWorkerDisputeLimit     = 100
)

var disputeErrors = dberror.Mapping{Entity: "SettlementDispute"}

type disputeRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewDispute(p Params) driversettlementports.SettlementDisputeRepository {
	return &disputeRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.settlement-dispute-repository"),
	}
}

func (r *disputeRepository) Create(
	ctx context.Context,
	entity *driversettlement.Dispute,
) (*driversettlement.Dispute, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driversettlement.Dispute, error) {
		if err := driversettlementdb.Disputes.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, disputeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *disputeRepository) Update(
	ctx context.Context,
	entity *driversettlement.Dispute,
) (*driversettlement.Dispute, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driversettlement.Dispute, error) {
		if err := driversettlementdb.Disputes.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, disputeErrors.Map(err, entity.ID.String())
		}

		return entity, nil
	})
}

func (r *disputeRepository) GetByID(
	ctx context.Context,
	req driversettlementports.GetSettlementDisputeByIDRequest,
) (*driversettlement.Dispute, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driversettlement.Dispute, error) {
		d := driversettlementdb.Disputes
		q := d.Where(d.ID.Eq(req.ID))
		if req.IncludeRelations {
			q = q.Load(d.Settlement(), d.SettlementLine(), d.Worker(), d.ResolvedBy())
		}

		entity, err := q.One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, disputeErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *disputeRepository) ListConnection(
	ctx context.Context,
	req *driversettlementports.ListSettlementDisputeConnectionRequest,
) (*pagination.CursorListResult[*driversettlement.Dispute], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*driversettlement.Dispute], error) {
			d := driversettlementdb.Disputes
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				d.Query(),
				&listquery.Request[driversettlement.Dispute]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape: func(
						q *keel.Query[*driversettlement.Dispute],
					) *keel.Query[*driversettlement.Dispute] {
						return q.Load(d.Worker(), d.Settlement(), d.ResolvedBy())
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list settlement disputes", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *disputeRepository) ListForWorker(
	ctx context.Context,
	req *driversettlementports.ListSettlementDisputesForWorkerRequest,
) ([]*driversettlement.Dispute, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*driversettlement.Dispute, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxWorkerDisputeLimit {
			limit = defaultWorkerDisputeLimit
		}

		d := driversettlementdb.Disputes
		items, err := d.Where(
			d.WorkerID.Eq(req.WorkerID),
			when(len(req.Statuses) > 0, d.Status.In(req.Statuses...)),
		).
			Load(d.Settlement(), d.SettlementLine()).
			Order(d.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list settlement disputes for worker: %w", err)
		}

		return items, nil
	})
}

func (r *disputeRepository) ListBySettlementIDs(
	ctx context.Context,
	req *driversettlementports.ListSettlementDisputesBySettlementIDsRequest,
) (map[pulid.ID][]*driversettlement.Dispute, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID][]*driversettlement.Dispute, error) {
			if len(req.SettlementIDs) == 0 {
				return map[pulid.ID][]*driversettlement.Dispute{}, nil
			}

			d := driversettlementdb.Disputes
			items, err := d.Where(d.SettlementID.In(req.SettlementIDs...)).
				Load(d.SettlementLine()).
				Order(d.CreatedAt.Desc(), d.ID.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list settlement disputes by settlement ids: %w", err)
			}

			grouped := make(map[pulid.ID][]*driversettlement.Dispute, len(req.SettlementIDs))
			for _, item := range items {
				grouped[item.SettlementID] = append(grouped[item.SettlementID], item)
			}

			return grouped, nil
		},
	)
}

func (r *disputeRepository) CountOpen(
	ctx context.Context,
	_ pagination.TenantInfo,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		d := driversettlementdb.Disputes
		count, err := d.Where(d.Status.In(
			driversettlement.DisputeStatusOpen,
			driversettlement.DisputeStatusInReview,
		)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count open settlement disputes: %w", err)
		}

		return int(count), nil
	})
}
