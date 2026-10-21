package pushsubscriptionrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/pkg/buncolgen/notificationcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
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

func (r *pushSubscriptionRepository) Save(
	ctx context.Context,
	req *notificationports.SavePushSubscriptionRequest,
) (*notification.PushSubscription, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*notification.PushSubscription, error) {
		cols := notificationcol.PushSubscriptionColumns
		entity := &notification.PushSubscription{
			BusinessUnitID: req.TenantInfo.BuID,
			OrganizationID: req.TenantInfo.OrgID,
			UserID:         req.TenantInfo.UserID,
			Endpoint:       req.Endpoint,
			P256dh:         req.P256dh,
			Auth:           req.Auth,
			UserAgent:      req.UserAgent,
		}
		result, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On("CONFLICT (md5(endpoint)) DO UPDATE").
			Set(cols.P256dh.SetExcluded()).
			Set(cols.Auth.SetExcluded()).
			Set(cols.UserAgent.SetExcluded()).
			Set(cols.UpdatedAt.SetExcluded()).
			Where(cols.UserID.Expr("{} = EXCLUDED." + cols.UserID.Bare())).
			Where(cols.OrganizationID.Expr("{} = EXCLUDED." + cols.OrganizationID.Bare())).
			Where(cols.BusinessUnitID.Expr("{} = EXCLUDED." + cols.BusinessUnitID.Bare())).
			Returning(strings.Join([]string{cols.ID.Bare(), cols.CreatedAt.Bare()}, ", ")).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("save push subscription: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("save push subscription: %w", err)
		}
		if affected == 0 {
			return nil, errortypes.NewConflictError(
				"This browser is already registered for notifications by another account",
			)
		}
		return entity, nil
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
			cols := notificationcol.PushSubscriptionColumns
			items := make([]*notification.PushSubscription, 0, 4)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(&items).
				Where(cols.UserID.Eq(), userID).
				Order(cols.CreatedAt.OrderDesc()).
				Scan(ctx)
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
		cols := notificationcol.PushSubscriptionColumns
		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*notification.PushSubscription)(nil)).
			Where(cols.UserID.Eq(), userID).
			Where(cols.Endpoint.Eq(), endpoint).
			Exec(ctx)
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
		cols := notificationcol.PushSubscriptionColumns
		_, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*notification.PushSubscription)(nil)).
			Where(cols.ID.Eq(), id).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("delete push subscription by id: %w", err)
		}
		return nil
	})
}
