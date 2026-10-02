package trackingeventrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/trackingevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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

func New(p Params) repositories.TrackingEventRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tracking-event-repository"),
	}
}

func (r *repository) Insert(
	ctx context.Context,
	entity *trackingevent.TrackingEvent,
) (*trackingevent.TrackingEvent, bool, error) {
	return dbtx.Write2(
		ctx,
		r.db,
		func(ctx context.Context) (*trackingevent.TrackingEvent, bool, error) {
			result, err := r.db.DBForContext(ctx).NewInsert().
				Model(entity).
				On("CONFLICT (organization_id, business_unit_id, source, source_key) DO NOTHING").
				Exec(ctx)
			if err != nil {
				return nil, false, fmt.Errorf("insert tracking event: %w", err)
			}
			rows, err := result.RowsAffected()
			if err != nil {
				return nil, false, fmt.Errorf("insert tracking event rows affected: %w", err)
			}
			if rows > 0 {
				return entity, true, nil
			}

			cols := buncolgen.TrackingEventColumns
			existing := new(trackingevent.TrackingEvent)
			err = r.db.DBForContext(ctx).NewSelect().
				Model(existing).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return buncolgen.TrackingEventScopeTenant(sq, pagination.TenantInfo{
						OrgID: entity.OrganizationID,
						BuID:  entity.BusinessUnitID,
					}).
						Where(cols.Source.Eq(), entity.Source).
						Where(cols.SourceKey.Eq(), entity.SourceKey)
				}).
				Scan(ctx)
			if err != nil {
				return nil, false, dberror.HandleNotFoundError(err, "Tracking event")
			}
			return existing, false, nil
		},
	)
}

func (r *repository) ListByMove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) ([]*trackingevent.TrackingEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*trackingevent.TrackingEvent, error) {
		cols := buncolgen.TrackingEventColumns
		entities := make([]*trackingevent.TrackingEvent, 0)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TrackingEventScopeTenant(sq, tenantInfo).
					Where(cols.ShipmentMoveID.Eq(), moveID)
			}).
			Order(cols.EventAt.OrderAsc(), cols.ID.OrderAsc()).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list tracking events by move: %w", err)
		}
		return entities, nil
	})
}

func (r *repository) ListByShipment(
	ctx context.Context,
	req *repositories.ListTrackingEventsByShipmentRequest,
) ([]*trackingevent.TrackingEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*trackingevent.TrackingEvent, error) {
		cols := buncolgen.TrackingEventColumns
		entities := make([]*trackingevent.TrackingEvent, 0)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&entities).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TrackingEventScopeTenant(sq, req.TenantInfo).
					Where(cols.ShipmentID.Eq(), req.ShipmentID)
			}).
			Order(cols.EventAt.OrderDesc(), cols.ID.OrderDesc()).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list tracking events by shipment: %w", err)
		}
		return entities, nil
	})
}

func (r *repository) RecordVerdicts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	verdicts []repositories.TrackingEventVerdict,
	at int64,
) error {
	if len(verdicts) == 0 {
		return nil
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.TrackingEventColumns
		for _, verdict := range verdicts {
			_, err := r.db.DBForContext(ctx).NewUpdate().
				Model((*trackingevent.TrackingEvent)(nil)).
				Set(cols.Outcome.Set(), verdict.Outcome).
				Set(cols.OutcomeReason.Set(), nullableReason(verdict.Reason)).
				Set(cols.UpdatedAt.Set(), at).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return buncolgen.TrackingEventScopeTenantUpdate(uq, tenantInfo).
						Where(cols.ID.Eq(), verdict.ID)
				}).
				Exec(ctx)
			if err != nil {
				return fmt.Errorf("record tracking event verdict: %w", err)
			}
		}
		return nil
	})
}

func nullableReason(reason string) any {
	if reason == "" {
		return nil
	}
	return reason
}
