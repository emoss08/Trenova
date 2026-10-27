package capturerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/zap"
)

var pairingErrors = dberror.Mapping{Entity: "Pairing"}

type pairingRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewPairingRepository(p Params) captureports.CapturePairingRepository {
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
	ctx = dbscope.WithSystem(
		ctx,
		"record a pairing request that belongs to no organization until a person claims it",
	)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*capture.CapturePairing, error) {
		if err := capturedb.CapturePairings.Create(ctx, r.db.Keel(ctx), entity); err != nil {
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
		if err := capturedb.CapturePairings.Save(ctx, r.db.Keel(ctx), entity); err != nil {
			return nil, pairingErrors.Map(err, entity.ID.String())
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
	ctx = dbscope.WithSystem(
		ctx,
		"resolve a pairing device code before it is bound to an organization",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CapturePairing, error) {
		p := capturedb.CapturePairings
		entity, err := p.Where(p.DeviceCodeHash.Eq(hash)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, pairingErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *pairingRepository) GetOpenByUserCode(
	ctx context.Context,
	userCode string,
) (*capture.CapturePairing, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"find an unclaimed pairing by the code a person typed; it has no organization yet",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*capture.CapturePairing, error) {
		p := capturedb.CapturePairings
		entity, err := p.Where(p.UserCode.Eq(userCode), p.Status.In(openPairingStatuses...)).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, pairingErrors.Map(err, "")
		}

		return entity, nil
	})
}

func (r *pairingRepository) ExpireStale(ctx context.Context, now int64) (int, error) {
	ctx = dbscope.WithSystem(ctx, "expire abandoned pairings across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		p := capturedb.CapturePairings
		n, err := p.Where(p.Status.In(openPairingStatuses...), p.ExpiresAt.Lte(now)).Update(
			ctx,
			r.db.Keel(ctx),
			p.Status.To(capture.PairingExpired),
			p.UpdatedAt.To(now),
			p.Version.ToExpr(p.Version),
		)
		if err != nil {
			return 0, err
		}

		return int(n), nil
	})
}
