package managedfeaturesrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures/managedfeaturesdb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const replaceLockKey = "managed-feature-list"

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) managedfeaturesports.StateRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.managed-features-repository")}
}

func (r *repository) Get(ctx context.Context) (*managedfeatures.State, error) {
	ctx = dbscope.WithSystem(ctx, "read the feature list the control plane set for this instance")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*managedfeatures.State, error) {
		return r.held(ctx)
	})
}

func (r *repository) held(ctx context.Context) (*managedfeatures.State, error) {
	state, err := managedfeaturesdb.States.Get(
		ctx,
		r.db.Keel(ctx),
		managedfeatures.CurrentStateID,
	)
	if errors.Is(err, keel.ErrNotFound) {
		return nil, managedfeatures.ErrNoState
	}
	if err != nil {
		return nil, fmt.Errorf("read the managed feature list: %w", err)
	}
	return state, nil
}

func (r *repository) Replace(
	ctx context.Context,
	req *managedfeaturesports.ReplaceRequest,
) (*managedfeatures.Replacement, error) {
	ctx = dbscope.WithSystem(ctx, "record the feature list the control plane set for this instance")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*managedfeatures.Replacement, error) {
		db := r.db.Keel(ctx)
		if err := keel.LockKey(ctx, db, replaceLockKey); err != nil {
			return nil, fmt.Errorf("lock the managed feature list: %w", err)
		}

		held, err := r.held(ctx)
		if err != nil && !errors.Is(err, managedfeatures.ErrNoState) {
			return nil, err
		}

		replacement := managedfeatures.Replace(held, req.Features, req.Now)
		if !replacement.Changed() {
			return replacement, nil
		}

		s := managedfeaturesdb.States
		if _, err = s.Upsert(
			ctx,
			db,
			[]*managedfeatures.State{replacement.Current},
			keel.OnConflict(s.ID).Update(
				s.Features,
				s.ReceivedAt,
			),
		); err != nil {
			r.l.Error("failed to record the managed feature list", zap.Error(err))
			return nil, fmt.Errorf("record the managed feature list: %w", err)
		}

		if err = r.recordChange(ctx, db, replacement.Change); err != nil {
			return nil, err
		}
		return replacement, nil
	})
}

func (r *repository) recordChange(
	ctx context.Context,
	db *keel.DB,
	change *managedfeatures.Change,
) error {
	c := managedfeaturesdb.Changes
	if err := c.Create(ctx, db, change); err != nil {
		r.l.Error("failed to record the managed feature change", zap.Error(err))
		return fmt.Errorf("record the managed feature change: %w", err)
	}

	kept := managedfeaturesdb.Changes.As("kept")
	if _, err := c.Where(c.ID.NotInQuery(kept.
		Pick(kept.ID).
		Order(kept.ChangedAt.Desc(), kept.ID.Desc()).
		Limit(managedfeatures.ChangeHistoryLimit))).
		Delete(ctx, db); err != nil {
		r.l.Error("failed to prune the managed feature changes", zap.Error(err))
		return fmt.Errorf("prune the managed feature changes: %w", err)
	}
	return nil
}

func (r *repository) History(
	ctx context.Context,
	limit int,
) (*managedfeatures.State, []*managedfeatures.Change, error) {
	ctx = dbscope.WithSystem(ctx, "read the feature list history the control plane set for this instance")
	return dbtx.Read2(
		ctx,
		r.db,
		func(ctx context.Context) (*managedfeatures.State, []*managedfeatures.Change, error) {
			held, err := r.held(ctx)
			if err != nil && !errors.Is(err, managedfeatures.ErrNoState) {
				return nil, nil, err
			}

			c := managedfeaturesdb.Changes
			changes, err := c.
				Order(c.ChangedAt.Desc(), c.ID.Desc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, nil, fmt.Errorf("read the managed feature changes: %w", err)
			}
			return held, changes, nil
		},
	)
}
