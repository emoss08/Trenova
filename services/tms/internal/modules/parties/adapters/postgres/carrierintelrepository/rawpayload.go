package carrierintelrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelsnapshot"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	defaultPurgeBatchSize = 1000
	maxPurgeBatchSize     = 10000
	rawPayloadEntityName  = "CarrierIntelRawPayload"
)

var rawPayloadErrors = dberror.Mapping{Entity: rawPayloadEntityName}

type rawPayloadRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewRawPayloadRepository(p Params) carrierintelports.CarrierIntelRawPayloadRepository {
	return &rawPayloadRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.carrier-intel-raw-payload-repository"),
	}
}

func (r *rawPayloadRepository) Insert(
	ctx context.Context,
	entity *carrierintelsnapshot.CarrierIntelRawPayload,
) (*carrierintelsnapshot.CarrierIntelRawPayload, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintelsnapshot.CarrierIntelRawPayload, error) {
			err := carrierinteldb.CarrierIntelRawPayloads.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error("failed to insert carrier intel raw payload", zap.Error(err))
				return nil, fmt.Errorf("insert carrier intel raw payload: %w", err)
			}

			return entity, nil
		},
	)
}

func (r *rawPayloadRepository) GetByID(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*carrierintelsnapshot.CarrierIntelRawPayload, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*carrierintelsnapshot.CarrierIntelRawPayload, error) {
			entity, err := carrierinteldb.CarrierIntelRawPayloads.Get(ctx, r.db.Keel(ctx), id)
			if err != nil {
				return nil, rawPayloadErrors.Map(err, id.String())
			}

			return entity, nil
		},
	)
}

func (r *rawPayloadRepository) PurgeExpired(
	ctx context.Context,
	before int64,
	limit int,
) (int, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"purge expired carrier intelligence payloads across every organization",
	)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		p := carrierinteldb.CarrierIntelRawPayloads
		batch := intutils.Clamp(
			intutils.WithDefault(max(limit, 0), defaultPurgeBatchSize),
			1,
			maxPurgeBatchSize,
		)

		e := p.As("expired")
		expired := e.AllTenants().
			Pick(e.ID).
			Where(e.ExpiresAt.Lt(before)).
			Order(e.ExpiresAt.Asc()).
			Limit(batch)

		affected, err := p.AllTenants().
			Where(p.ID.InQuery(expired), p.ExpiresAt.Lt(before)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to purge expired carrier intel raw payloads", zap.Error(err))
			return 0, fmt.Errorf("purge expired carrier intel raw payloads: %w", err)
		}

		return int(affected), nil
	})
}
