package pushsubscriptionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification/notificationdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type pushSubscriptionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) notificationports.PushSubscriptionRepository {
	return &pushSubscriptionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.push-subscription-repository"),
	}
}

func subscriptions(ctx context.Context) *keel.Query[notification.PushSubscription] {
	q := notificationdb.PushSubscriptions.Query()
	if dbscope.IsSystem(ctx) {
		q = q.AllTenants()
	}

	return q
}

func (r *pushSubscriptionRepository) Save(
	ctx context.Context,
	req *notificationports.SavePushSubscriptionRequest,
) (*notification.PushSubscription, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*notification.PushSubscription, error) {
		p := notificationdb.PushSubscriptions
		db := r.db.Keel(ctx)
		entity := &notification.PushSubscription{
			BusinessUnitID: req.TenantInfo.BuID,
			OrganizationID: req.TenantInfo.OrgID,
			UserID:         req.TenantInfo.UserID,
			Endpoint:       req.Endpoint,
			P256dh:         req.P256dh,
			Auth:           req.Auth,
			UserAgent:      req.UserAgent,
		}

		inserted, err := p.CreateOrSkip(
			ctx,
			db,
			[]*notification.PushSubscription{entity},
			keel.DynFunc("md5", p.Endpoint),
		)
		if err != nil {
			return nil, fmt.Errorf("save push subscription: %w", err)
		}
		if inserted == 1 {
			return entity, nil
		}

		updated, err := p.Where(
			keel.DynFunc("md5", p.Endpoint).Eq(keel.DynFunc("md5", req.Endpoint)),
			p.UserID.Eq(req.TenantInfo.UserID),
		).UpdateReturning(ctx, db, p.P256dh.To(req.P256dh), p.Auth.To(req.Auth), p.UserAgent.To(req.UserAgent))
		if err != nil {
			return nil, fmt.Errorf("save push subscription: %w", err)
		}
		if len(updated) == 0 {
			return nil, errortypes.NewConflictError(
				"This browser is already registered for notifications by another account",
			)
		}

		return updated[0], nil
	})
}

func (r *pushSubscriptionRepository) ListByUser(
	ctx context.Context,
	userID pulid.ID,
) ([]*notification.PushSubscription, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*notification.PushSubscription, error) {
			p := notificationdb.PushSubscriptions
			items, err := subscriptions(ctx).
				Where(p.UserID.Eq(userID)).
				Order(p.CreatedAt.Desc()).
				All(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, fmt.Errorf("list push subscriptions: %w", err)
			}

			return items, nil
		},
	)
}

func (r *pushSubscriptionRepository) DeleteByEndpoint(
	ctx context.Context,
	userID pulid.ID,
	endpoint string,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := notificationdb.PushSubscriptions
		_, err := subscriptions(ctx).
			Where(p.UserID.Eq(userID), p.Endpoint.Eq(endpoint)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return fmt.Errorf("delete push subscription: %w", err)
		}

		return nil
	})
}

func (r *pushSubscriptionRepository) DeleteByID(
	ctx context.Context,
	id pulid.ID,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := notificationdb.PushSubscriptions
		_, err := subscriptions(ctx).Where(p.ID.Eq(id)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return fmt.Errorf("delete push subscription by id: %w", err)
		}

		return nil
	})
}
