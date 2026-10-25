package customertrackingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking/customertrackingdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var notificationErrors = dberror.Mapping{Entity: "Customer status notification"}

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
		n := customertrackingdb.Notifications
		inserted, err := n.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			[]*customertracking.Notification{entity},
			n.OrganizationID,
			n.BusinessUnitID,
			n.ShipmentID,
			n.Event,
			n.SubjectKey,
		)
		if err != nil {
			return false, fmt.Errorf(
				"insert customer status notification: %w",
				notificationErrors.Map(err, entity.ID.String()),
			)
		}
		return inserted == 1, nil
	})
}

func (r *notificationRepository) GetByID(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*customertracking.Notification, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*customertracking.Notification, error) {
		entity, err := customertrackingdb.Notifications.Get(ctx, r.db.Keel(ctx), id)
		if err != nil {
			return nil, notificationErrors.Map(err, id.String())
		}
		return entity, nil
	})
}

func (r *notificationRepository) ListForShipment(
	ctx context.Context,
	_ pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*customertracking.Notification, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customertracking.Notification, error) {
			n := customertrackingdb.Notifications
			entities, err := n.Where(n.ShipmentID.Eq(shipmentID)).
				Order(n.OccurredAt.Desc(), n.ID.Desc()).
				Limit(maxShipmentLedger).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list customer status notifications: %w", err)
			}
			return entities, nil
		},
	)
}

func (r *notificationRepository) ListDue(
	ctx context.Context,
	_ pagination.TenantInfo,
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
			n := customertrackingdb.Notifications
			entities, err := n.Where(
				n.Status.Eq(customertracking.NotificationStatusPending),
				n.NextAttemptAt.Lte(now),
			).
				Order(n.NextAttemptAt.Asc(), n.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
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
		n := customertrackingdb.Notifications
		rows, err := keel.Into[tenantRow](ctx, r.db.Keel(ctx), n.AllTenants().
			Group(n.OrganizationID, n.BusinessUnitID).
			Pick(n.OrganizationID, n.BusinessUnitID).
			Where(
				n.Status.Eq(customertracking.NotificationStatusPending),
				n.NextAttemptAt.Lte(now),
			))
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
		n := customertrackingdb.Notifications
		recipients := outcome.Recipients
		if recipients == nil {
			recipients = []string{}
		}
		sentAt := n.SentAt.ToNull()
		if outcome.SentAt != nil {
			sentAt = n.SentAt.To(*outcome.SentAt)
		}

		rows, err := n.Where(
			n.ID.Eq(outcome.ID),
			n.Status.Eq(customertracking.NotificationStatusPending),
		).Update(
			ctx,
			r.db.Keel(ctx),
			n.Status.To(outcome.Status),
			n.Recipients.To(recipients),
			n.Attempts.To(outcome.Attempts),
			n.NextAttemptAt.To(outcome.NextAttemptAt),
			n.UpdatedAt.To(outcome.At),
			n.SkipReason.To(outcome.SkipReason),
			n.LastError.To(outcome.LastError),
			n.LinkID.To(outcome.LinkID),
			n.EmailMessageID.To(outcome.EmailMessageID),
			sentAt,
		)
		if err != nil {
			return false, fmt.Errorf("record customer status notification outcome: %w", err)
		}
		return rows == 1, nil
	})
}
