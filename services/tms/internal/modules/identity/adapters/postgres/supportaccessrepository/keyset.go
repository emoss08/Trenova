package supportaccessrepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess/supportaccessdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/zap"
)

const (
	keySetReadReason  = "read the support assertion keys the control plane set for this instance"
	keySetWriteReason = "record the support assertion keys the control plane set for this instance"
	ledgerReason      = "record a support assertion as used before any tenant is known"
	ledgerPruneBatch  = 100
)

type keySetRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewKeySetRepository(p Params) supportaccessports.KeySetRepository {
	return &keySetRepository{db: p.DB, l: p.Logger.Named("postgres.support-access-keyset-repository")}
}

func (r *keySetRepository) Get(ctx context.Context) (*supportaccess.KeySet, error) {
	ctx = dbscope.WithSystem(ctx, keySetReadReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*supportaccess.KeySet, error) {
		keys, err := supportaccessdb.KeySets.Get(ctx, r.db.Keel(ctx), supportaccess.CurrentKeySetID)
		if errors.Is(err, keel.ErrNotFound) {
			return nil, supportaccess.ErrNoKeySet
		}
		if err != nil {
			return nil, fmt.Errorf("read the support assertion keys: %w", err)
		}
		return keys, nil
	})
}

func (r *keySetRepository) Save(ctx context.Context, keys *supportaccess.KeySet) error {
	ctx = dbscope.WithSystem(ctx, keySetWriteReason)
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		k := supportaccessdb.KeySets
		keys.ID = supportaccess.CurrentKeySetID
		if _, err := k.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*supportaccess.KeySet{keys},
			keel.OnConflict(k.ID).Update(k.InstanceID, k.StaffDomain, k.Keys, k.ReceivedAt),
		); err != nil {
			return fmt.Errorf("record the support assertion keys: %w", err)
		}
		return nil
	})
}

type assertionLedger struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAssertionLedger(p Params) supportaccessports.AssertionLedger {
	return &assertionLedger{db: p.DB, l: p.Logger.Named("postgres.support-assertion-ledger")}
}

// Use records the assertion id and reports whether this call was the first to: the primary key
// decides between two uses at once. It also drops a bounded batch of ids that can no longer
// verify, so the table holds about a minute of entries.
func (r *assertionLedger) Use(ctx context.Context, use *supportaccess.AssertionUse) (bool, error) {
	ctx = dbscope.WithSystem(ctx, ledgerReason)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		db := r.db.Keel(ctx)
		u := supportaccessdb.AssertionUses
		inserted, err := u.CreateOrSkip(ctx, db, []*supportaccess.AssertionUse{use}, u.AssertionID)
		if err != nil {
			return false, fmt.Errorf("record a support assertion: %w", err)
		}

		stale := u.Pick(u.AssertionID).
			Where(u.ExpiresAt.Lt(use.UsedAt)).
			Order(u.ExpiresAt.Asc()).
			Limit(ledgerPruneBatch)
		if _, err = u.Where(u.AssertionID.InQuery(stale)).Delete(ctx, db); err != nil {
			return false, fmt.Errorf("drop expired support assertions: %w", err)
		}

		return inserted == 1, nil
	})
}
