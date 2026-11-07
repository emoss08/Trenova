package driverpositioningrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/driverstart"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning/driverpositioningdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/driverpositioningports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
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

func New(p Params) driverpositioningports.Repository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.driver-positioning-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req *driverpositioningports.GetRequest,
) (*driverpositioning.Positioning, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*driverpositioning.Positioning, error) {
		positioning, err := driverpositioningdb.Positionings.Get(ctx, r.db.Keel(ctx), req.ID)
		if errors.Is(err, keel.ErrNotFound) {
			return nil, errortypes.NewNotFoundError("Positioning not found")
		}
		if err != nil {
			return nil, fmt.Errorf("read positioning: %w", err)
		}

		return positioning, nil
	})
}

func (r *repository) ListLive(
	ctx context.Context,
	req *driverpositioningports.LiveRequest,
) ([]*driverpositioning.Positioning, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*driverpositioning.Positioning, error) {
			p := driverpositioningdb.Positionings
			rows, err := p.
				Where(append(
					[]keel.Pred{keel.When(len(req.WorkerIDs) > 0, p.WorkerID.In(req.WorkerIDs...))},
					driverstart.LivePositioning(p, p.WorkerID, req.Now)...,
				)...).
				Order(p.SlotStart.Asc(), p.ID.Asc()).
				Limit(driverpositioningports.MaxLive).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list live positionings: %w", err)
			}

			return rows, nil
		},
	)
}

func (r *repository) Replace(
	ctx context.Context,
	req *driverpositioningports.ReplaceRequest,
) (*driverpositioningports.Replaced, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*driverpositioningports.Replaced, error) {
			db := r.db.Keel(ctx)
			p := driverpositioningdb.Positionings
			previous, err := p.
				Where(
					p.WorkerID.Eq(req.Positioning.WorkerID),
					p.Status.Eq(driverpositioning.StatusActive),
				).
				Lock(keel.ForUpdate).
				Limit(1).
				List(ctx, db)
			if err != nil {
				return nil, fmt.Errorf("read active positioning: %w", err)
			}

			out := &driverpositioningports.Replaced{Positioning: req.Positioning}
			if len(previous) > 0 {
				out.Previous = previous[0]
				out.Previous.Status = driverpositioning.StatusReplaced
				if err = p.Save(ctx, db, out.Previous, p.Status); err != nil {
					return nil, fmt.Errorf("replace active positioning: %w", err)
				}
			}
			if err = p.Create(ctx, db, req.Positioning); err != nil {
				if keel.IsUnique(err, "uq_driver_positionings_active_worker") {
					return nil, errortypes.NewConflictError(
						"This driver was just sent somewhere else. Refresh and try again.",
					)
				}
				return nil, fmt.Errorf("create positioning: %w", err)
			}

			return out, nil
		},
	)
}

func (r *repository) Cancel(
	ctx context.Context,
	req *driverpositioningports.CancelRequest,
) (*driverpositioning.Positioning, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*driverpositioning.Positioning, error) {
		db := r.db.Keel(ctx)
		p := driverpositioningdb.Positionings
		positioning, err := p.Where(p.ID.Eq(req.ID)).Lock(keel.ForUpdate).One(ctx, db)
		if errors.Is(err, keel.ErrNotFound) {
			return nil, errortypes.NewNotFoundError("Positioning not found")
		}
		if err != nil {
			return nil, fmt.Errorf("read positioning: %w", err)
		}
		if positioning.Status != driverpositioning.StatusActive {
			return nil, errortypes.NewBusinessError(
				"This positioning was already called off or replaced.",
			)
		}

		positioning.Status = driverpositioning.StatusCanceled
		positioning.CanceledByID = req.CanceledByID
		canceledAt := req.CanceledAt
		positioning.CanceledAt = &canceledAt
		if err = p.Save(ctx, db, positioning, p.Status, p.CanceledByID, p.CanceledAt); err != nil {
			return nil, fmt.Errorf("cancel positioning: %w", err)
		}

		return positioning, nil
	})
}
