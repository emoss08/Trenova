package pushsubscriptionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
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

func New(p Params) repositories.PushSubscriptionRepository {
	return &pushSubscriptionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.push-subscription-repository"),
	}
}

func (r *pushSubscriptionRepository) Save(
	ctx context.Context,
	req *repositories.SavePushSubscriptionRequest,
) (*notification.PushSubscription, error) {
	cols := buncolgen.PushSubscriptionColumns
	entity := &notification.PushSubscription{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		UserID:         req.TenantInfo.UserID,
		Endpoint:       req.Endpoint,
		P256dh:         req.P256dh,
		Auth:           req.Auth,
		UserAgent:      req.UserAgent,
	}
	_, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		On("CONFLICT (md5(endpoint)) DO UPDATE").
		Set(cols.UserID.SetExcluded()).
		Set(cols.OrganizationID.SetExcluded()).
		Set(cols.BusinessUnitID.SetExcluded()).
		Set(cols.P256dh.SetExcluded()).
		Set(cols.Auth.SetExcluded()).
		Set(cols.UserAgent.SetExcluded()).
		Set(cols.UpdatedAt.SetExcluded()).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("save push subscription: %w", err)
	}
	return entity, nil
}

func (r *pushSubscriptionRepository) ListByUser(
	ctx context.Context,
	userID pulid.ID,
) ([]*notification.PushSubscription, error) {
	cols := buncolgen.PushSubscriptionColumns
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
}

func (r *pushSubscriptionRepository) DeleteByEndpoint(
	ctx context.Context,
	userID pulid.ID,
	endpoint string,
) error {
	cols := buncolgen.PushSubscriptionColumns
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
}

func (r *pushSubscriptionRepository) DeleteByID(
	ctx context.Context,
	id pulid.ID,
) error {
	cols := buncolgen.PushSubscriptionColumns
	_, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*notification.PushSubscription)(nil)).
		Where(cols.ID.Eq(), id).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete push subscription by id: %w", err)
	}
	return nil
}
