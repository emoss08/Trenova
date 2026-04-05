package notificationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.NotificationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.notification-repository"),
	}
}

func (r *repository) userOrGlobalFilter(q *bun.SelectQuery, tenantInfo pagination.TenantInfo) *bun.SelectQuery {
	return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.
			WhereGroup(" OR ", func(inner *bun.SelectQuery) *bun.SelectQuery {
				return inner.
					Where("notif.target_user_id = ? AND notif.business_unit_id = ?", tenantInfo.UserID, tenantInfo.BuID).
					WhereOr("notif.channel = ?", notification.ChannelGlobal)
			})
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	log := r.l.With(zap.String("operation", "Create"))

	_, err := r.db.DB().NewInsert().Model(entity).Exec(ctx)
	if err != nil {
		log.Error("failed to create notification", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListNotificationsRequest,
) (*pagination.ListResult[*notification.Notification], error) {
	log := r.l.With(zap.String("operation", "List"))

	entities := make([]*notification.Notification, 0, req.Filter.Pagination.SafeLimit())
	q := r.db.DB().
		NewSelect().
		Model(&entities)

	q = querybuilder.ApplyFilters(
		q,
		"notif",
		req.Filter,
		(*notification.Notification)(nil),
	)

	q = q.Where("notif.organization_id = ?", req.Filter.TenantInfo.OrgID)
	q = r.userOrGlobalFilter(q, req.Filter.TenantInfo)
	q = q.Order("notif.created_at DESC")
	q = q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to list notifications", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*notification.Notification]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) MarkAsRead(
	ctx context.Context,
	req repositories.MarkNotificationsReadRequest,
) error {
	log := r.l.With(zap.String("operation", "MarkAsRead"))

	_, err := r.db.DB().
		NewUpdate().
		Model((*notification.Notification)(nil)).
		Set("read_at = ?", timeutils.NowUnix()).
		Where("notif.id IN (?)", bun.List(req.IDs)).
		Where("notif.organization_id = ?", req.TenantInfo.OrgID).
		Where("((notif.target_user_id = ? AND notif.business_unit_id = ?) OR notif.channel = ?)",
			req.TenantInfo.UserID, req.TenantInfo.BuID, notification.ChannelGlobal).
		Where("notif.read_at IS NULL").
		Exec(ctx)
	if err != nil {
		log.Error("failed to mark notifications as read", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) MarkAllAsRead(
	ctx context.Context,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	log := r.l.With(zap.String("operation", "MarkAllAsRead"))

	_, err := r.db.DB().
		NewUpdate().
		Model((*notification.Notification)(nil)).
		Set("read_at = ?", timeutils.NowUnix()).
		Where("notif.organization_id = ?", tenantInfo.OrgID).
		Where("((notif.target_user_id = ? AND notif.business_unit_id = ?) OR notif.channel = ?)",
			userID, tenantInfo.BuID, notification.ChannelGlobal).
		Where("notif.read_at IS NULL").
		Exec(ctx)
	if err != nil {
		log.Error("failed to mark all notifications as read", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) CountUnread(
	ctx context.Context,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (int64, error) {
	log := r.l.With(zap.String("operation", "CountUnread"))

	q := r.db.DB().
		NewSelect().
		Model((*notification.Notification)(nil)).
		Where("notif.organization_id = ?", tenantInfo.OrgID).
		Where("notif.read_at IS NULL")

	q = r.userOrGlobalFilter(q, tenantInfo)

	count, err := q.Count(ctx)
	if err != nil {
		log.Error("failed to count unread notifications", zap.Error(err))
		return 0, err
	}

	return int64(count), nil
}
