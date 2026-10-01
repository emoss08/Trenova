package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type pairingRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPairingRepository(p Params) repositories.CapturePairingRepository {
	return &pairingRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.capture-pairing-repository"),
	}
}

var openPairingStatuses = []capture.PairingStatus{capture.PairingPending, capture.PairingApproved}

func (r *pairingRepository) Create(
	ctx context.Context,
	entity *capture.CapturePairing,
) (*capture.CapturePairing, error) {
	ctx = dbscope.WithSystem(ctx, "record a pairing request that belongs to no organization until a person claims it")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CapturePairing, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *pairingRepository) Update(
	ctx context.Context,
	entity *capture.CapturePairing,
) (*capture.CapturePairing, error) {
	ctx = dbscope.WithSystem(ctx, "update a pairing that may not belong to an organization yet")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CapturePairing, error) {
		ov := entity.Version
		entity.Version++

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WherePK().
			Where(buncolgen.CapturePairingColumns.Version.Eq(), ov).
			Returning("*").
			Exec(ctx)
		if err != nil {
			entity.Version = ov

			return nil, err
		}
		if err = dberror.CheckRowsAffected(results, "Pairing", entity.ID.String()); err != nil {
			entity.Version = ov

			return nil, err
		}

		return entity, nil
	})
}

// GetByDeviceCodeHash is not tenant-scoped: a grant has no tenant until it is
// approved, and the device code is what the machine holds to find it.
func (r *pairingRepository) GetByDeviceCodeHash(
	ctx context.Context,
	hash string,
) (*capture.CapturePairing, error) {
	ctx = dbscope.WithSystem(ctx, "resolve a pairing device code before it is bound to an organization")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CapturePairing, error) {
		entity := new(capture.CapturePairing)

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(buncolgen.CapturePairingColumns.DeviceCodeHash.Eq(), hash).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Pairing")
		}

		return entity, nil
	})
}

func (r *pairingRepository) GetOpenByUserCode(
	ctx context.Context,
	userCode string,
) (*capture.CapturePairing, error) {
	ctx = dbscope.WithSystem(ctx, "find an unclaimed pairing by the code a person typed; it has no organization yet")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CapturePairing, error) {
		entity := new(capture.CapturePairing)
		cols := buncolgen.CapturePairingColumns

		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Where(cols.UserCode.Eq(), userCode).
			Where(cols.Status.In(), bun.List(openPairingStatuses)).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Pairing")
		}

		return entity, nil
	})
}

func (r *pairingRepository) ExpireStale(ctx context.Context, now int64) (int, error) {
	ctx = dbscope.WithSystem(ctx, "expire abandoned pairings across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := buncolgen.CapturePairingColumns

		result, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*capture.CapturePairing)(nil)).
			Where(cols.Status.In(), bun.List(openPairingStatuses)).
			Where(cols.ExpiresAt.Lte(), now).
			Set(cols.Status.Set(), capture.PairingExpired).
			Set(cols.UpdatedAt.Set(), now).
			Exec(ctx)
		if err != nil {
			return 0, err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}

		return int(affected), nil
	})
}
