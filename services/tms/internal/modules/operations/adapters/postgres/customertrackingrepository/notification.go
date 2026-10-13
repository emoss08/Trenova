package customertrackingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/buncolgen/customertrackingcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type notificationRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewNotifications(p Params) customertrackingports.NotificationRepository {
	return &notificationRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.customer-status-notification-repository"),
	}
}

func (r *notificationRepository) InsertOnce(
	ctx context.Context,
	entity *customertracking.Notification,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		cols := customertrackingcol.NotificationColumns
		result, err := r.db.DBForContext(ctx).NewInsert().
			Model(entity).
			On("CONFLICT (?, ?, ?, ?, ?) DO NOTHING",
				bun.Ident(cols.OrganizationID.Bare()),
				bun.Ident(cols.BusinessUnitID.Bare()),
				bun.Ident(cols.ShipmentID.Bare()),
				bun.Ident(cols.Event.Bare()),
				bun.Ident(cols.SubjectKey.Bare()),
			).
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("insert customer status notification: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("insert customer status notification rows affected: %w", err)
		}
		return rows == 1, nil
	})
}

func (r *notificationRepository) GetByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*customertracking.Notification, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*customertracking.Notification, error) {
		cols := customertrackingcol.NotificationColumns
		entity := new(customertracking.Notification)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return customertrackingcol.NotificationScopeTenant(sq, tenantInfo).
					Where(cols.ID.Eq(), id)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "Customer status notification")
		}
		return entity, nil
	})
}

func (r *notificationRepository) ListForShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*customertracking.Notification, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customertracking.Notification, error) {
			cols := customertrackingcol.NotificationColumns
			entities := make([]*customertracking.Notification, 0, 4)
			err := r.db.DBForContext(ctx).NewSelect().
				Model(&entities).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return customertrackingcol.NotificationScopeTenant(sq, tenantInfo).
						Where(cols.ShipmentID.Eq(), shipmentID)
				}).
				Order(cols.OccurredAt.OrderDesc(), cols.ID.OrderDesc()).
				Limit(maxShipmentLedger).
				Scan(ctx)
			if err != nil {
				return nil, fmt.Errorf("list customer status notifications: %w", err)
			}
			return entities, nil
		},
	)
}

func (r *notificationRepository) ListDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
	limit int,
) ([]*customertracking.Notification, error) {
	if limit <= 0 {
		limit = defaultDueLimit
	}
	limit = min(limit, maxDueLimit)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customertracking.Notification, error) {
			cols := customertrackingcol.NotificationColumns
			entities := make([]*customertracking.Notification, 0, limit)
			err := r.db.DBForContext(ctx).NewSelect().
				Model(&entities).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return customertrackingcol.NotificationScopeTenant(sq, tenantInfo).
						Where(cols.Status.Eq(), customertracking.NotificationStatusPending).
						Where(cols.NextAttemptAt.Lte(), now)
				}).
				Order(cols.NextAttemptAt.OrderAsc(), cols.ID.OrderAsc()).
				Limit(int64(limit)).
				Scan(ctx)
			if err != nil {
				return nil, fmt.Errorf("list due customer status notifications: %w", err)
			}
			return entities, nil
		},
	)
}

func (r *notificationRepository) ListTenantsWithDue(
	ctx context.Context,
	now int64,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "find organizations with customer status emails due")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		cols := customertrackingcol.NotificationColumns
		rows := make([]tenantRow, 0, 8)
		err := r.db.DBForContext(ctx).NewSelect().
			Model((*customertracking.Notification)(nil)).
			ColumnExpr(cols.OrganizationID.As(orgColumn)).
			ColumnExpr(cols.BusinessUnitID.As(buColumn)).
			Where(cols.Status.Eq(), customertracking.NotificationStatusPending).
			Where(cols.NextAttemptAt.Lte(), now).
			GroupExpr(cols.OrganizationID.String()).
			GroupExpr(cols.BusinessUnitID.String()).
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("list tenants with due customer status notifications: %w", err)
		}
		return tenantsOf(rows), nil
	})
}

func (r *notificationRepository) Record(
	ctx context.Context,
	outcome *customertrackingports.DeliveryOutcome,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		cols := customertrackingcol.NotificationColumns
		q := r.db.DBForContext(ctx).NewUpdate().
			Model((*customertracking.Notification)(nil)).
			Set(cols.Status.Set(), outcome.Status).
			Set(cols.Recipients.Set(), pgArray(outcome.Recipients)).
			Set(cols.Attempts.Set(), outcome.Attempts).
			Set(cols.NextAttemptAt.Set(), outcome.NextAttemptAt).
			Set(cols.UpdatedAt.Set(), outcome.At)
		q = setNullable(q, cols.SkipReason.Bare(), string(outcome.SkipReason))
		q = setNullable(q, cols.LastError.Bare(), outcome.LastError)
		q = setNullableID(q, cols.LinkID.Bare(), outcome.LinkID)
		q = setNullableID(q, cols.EmailMessageID.Bare(), outcome.EmailMessageID)
		if outcome.SentAt != nil {
			q = q.Set(cols.SentAt.Set(), *outcome.SentAt)
		} else {
			q = q.Set(cols.SentAt.SetNull())
		}
		result, err := q.
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return customertrackingcol.NotificationScopeTenantUpdate(uq, outcome.TenantInfo).
					Where(cols.ID.Eq(), outcome.ID).
					Where(cols.Status.Eq(), customertracking.NotificationStatusPending)
			}).
			Exec(ctx)
		if err != nil {
			return false, fmt.Errorf("record customer status notification outcome: %w", err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("record customer status notification rows affected: %w", err)
		}
		return rows == 1, nil
	})
}
