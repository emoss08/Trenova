package managedusagerepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage/managedusagedb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/pkg/dbscope"
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

func New(p Params) managedusageports.StateRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.managed-usage-repository")}
}

func (r *repository) Get(ctx context.Context) (*managedusage.State, error) {
	ctx = dbscope.WithSystem(ctx, "read the usage state the control plane set for this instance")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*managedusage.State, error) {
		state, err := managedusagedb.States.Get(ctx, r.db.Keel(ctx), managedusage.CurrentStateID)
		if errors.Is(err, keel.ErrNotFound) {
			return nil, managedusage.ErrNoState
		}
		if err != nil {
			return nil, fmt.Errorf("read the managed usage state: %w", err)
		}
		return state, nil
	})
}

func (r *repository) Save(ctx context.Context, state *managedusage.State) error {
	ctx = dbscope.WithSystem(ctx, "record the usage state the control plane set for this instance")
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		s := managedusagedb.States
		state.ID = managedusage.CurrentStateID
		if _, err := s.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*managedusage.State{state},
			keel.OnConflict(s.ID).Update(
				s.WindowStart,
				s.WindowEnd,
				s.Overage,
				s.Meters,
				s.ReceivedAt,
			),
		); err != nil {
			r.l.Error("failed to record the managed usage state", zap.Error(err))
			return fmt.Errorf("record the managed usage state: %w", err)
		}
		return nil
	})
}
