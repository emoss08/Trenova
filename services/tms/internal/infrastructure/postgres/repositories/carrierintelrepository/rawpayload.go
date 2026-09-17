package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	defaultPurgeBatchSize = 1000
	maxPurgeBatchSize     = 10000
	rawPayloadEntityName  = "CarrierIntelRawPayload"
)

type rawPayloadRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRawPayloadRepository(p Params) repositories.CarrierIntelRawPayloadRepository {
	return &rawPayloadRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-raw-payload-repository"),
	}
}

func (r *rawPayloadRepository) Insert(
	ctx context.Context,
	entity *carrierintel.CarrierIntelRawPayload,
) (*carrierintel.CarrierIntelRawPayload, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		r.l.Error("failed to insert carrier intel raw payload", zap.Error(err))
		return nil, fmt.Errorf("insert carrier intel raw payload: %w", err)
	}

	return entity, nil
}

func (r *rawPayloadRepository) GetByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*carrierintel.CarrierIntelRawPayload, error) {
	entity := new(carrierintel.CarrierIntelRawPayload)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CarrierIntelRawPayloadScopeTenant(sq, tenantInfo).
				Where(buncolgen.CarrierIntelRawPayloadColumns.ID.Eq(), id)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, rawPayloadEntityName)
	}

	return entity, nil
}

func (r *rawPayloadRepository) PurgeExpired(
	ctx context.Context,
	before int64,
	limit int,
) (int, error) {
	cols := buncolgen.CarrierIntelRawPayloadColumns
	batch := intutils.Clamp(
		intutils.WithDefault(max(limit, 0), defaultPurgeBatchSize),
		1,
		maxPurgeBatchSize,
	)

	dba := r.db.DBForContext(ctx)
	expired := dba.NewSelect().
		Model((*carrierintel.CarrierIntelRawPayload)(nil)).
		Column(cols.ID.Bare()).
		Where(cols.ExpiresAt.Lt(), before).
		Order(cols.ExpiresAt.OrderAsc()).
		Limit(batch)

	result, err := dba.NewDelete().
		Model((*carrierintel.CarrierIntelRawPayload)(nil)).
		Where(cols.ID.In(), expired).
		Where(cols.ExpiresAt.Lt(), before).
		Exec(ctx)
	if err != nil {
		r.l.Error("failed to purge expired carrier intel raw payloads", zap.Error(err))
		return 0, fmt.Errorf("purge expired carrier intel raw payloads: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge expired carrier intel raw payloads rows affected: %w", err)
	}

	return int(affected), nil
}
