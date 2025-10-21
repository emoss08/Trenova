package notificationrepository

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
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

func NewRepository(p Params) repositories.NotificationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.notification-repository"),
	}
}

func (r *repository) Create(
	ctx context.Context,
	notif *notification.Notification,
) error {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("notification_id", notif.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = db.NewInsert().Model(notif).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to insert notification", zap.Error(err), zap.Any("notification", notif))
		return err
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	notif *notification.Notification,
) error {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("notification_id", notif.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	ov := notif.Version
	notif.Version++

	results, rErr := db.NewUpdate().
		Model(notif).
		WherePK().
		Where("notif.version = ?", ov).
		Returning("*").
		Exec(ctx)

	if rErr != nil {
		log.Error("failed to update notification", zap.Error(rErr))
		return rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Notification", notif.ID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) GetByID(
	ctx context.Context,
	id pulid.ID,
) (*notification.Notification, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("notification_id", id.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	notif := new(notification.Notification)
	err = db.NewSelect().
		Model(notif).
		Where("notif.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Notification")
	}

	return notif, nil
}

func (r *repository) buildUserNotificationsQuery(
	q bun.QueryBuilder,
	req *repositories.GetUserNotificationsRequest,
) bun.QueryBuilder {
	q.WhereGroup(" AND", func(sq bun.QueryBuilder) bun.QueryBuilder {
		return sq.
			Where("notif.organization_id = ?", req.Filter.TenantOpts.OrgID).
			Where("notif.business_unit_id = ?", req.Filter.TenantOpts.BuID).
			Where("notif.target_user_id = ?", req.Filter.TenantOpts.UserID)
	})

	if req.UnreadOnly {
		q = q.Where("notif.read_at IS NULL")
	}

	q = q.Where(
		"(notif.expires_at IS NULL OR notif.expires_at > extract(epoch from current_timestamp)::bigint)",
	)

	return q
}

func (r *repository) GetUserNotifications(
	ctx context.Context,
	req *repositories.GetUserNotificationsRequest,
) (*pagination.ListResult[*notification.Notification], error) {
	log := r.l.With(
		zap.String("operation", "GetUserNotifications"),
		zap.String("user_id", req.Filter.TenantOpts.UserID.String()),
		zap.String("organization_id", req.Filter.TenantOpts.OrgID.String()),
		zap.Bool("unread_only", req.UnreadOnly),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	notifications := make([]*notification.Notification, 0, req.Filter.Limit)

	q := db.NewSelect().
		Model(&notifications).
		ApplyQueryBuilder(func(qb bun.QueryBuilder) bun.QueryBuilder {
			return r.buildUserNotificationsQuery(qb, req)
		})

	q = q.Order("notif.created_at DESC").
		Limit(req.Filter.Limit).
		Offset(req.Filter.Offset)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan notifications", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*notification.Notification]{
		Items: notifications,
		Total: total,
	}, nil
}

func (r *repository) GetUnreadCount(
	ctx context.Context,
	userID pulid.ID,
	organizationID pulid.ID,
) (int, error) {
	log := r.l.With(
		zap.String("operation", "GetUnreadCount"),
		zap.String("user_id", userID.String()),
		zap.String("organization_id", organizationID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return 0, err
	}

	count, err := db.NewSelect().
		Model((*notification.Notification)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				WhereOr("notif.channel = ?", notification.ChannelGlobal).
				WhereOr("notif.channel = ? AND notif.target_user_id = ?", notification.ChannelUser, userID).
				Where("notif.read_at IS NULL")
		}).
		Where("notif.read_at IS NULL").
		Where("notif.organization_id = ?", organizationID).
		Where("(notif.expires_at IS NULL OR notif.expires_at > extract(epoch from current_timestamp)::bigint)").
		Count(ctx)
	if err != nil {
		log.Error("failed to count unread notifications", zap.Error(err))
		return 0, err
	}

	return count, nil
}

func (r *repository) ReadAllNotifications(
	ctx context.Context,
	req repositories.ReadAllNotificationsRequest,
) error {
	log := r.l.With(
		zap.String("operation", "ReadAllNotifications"),
		zap.String("user_id", req.UserID.String()),
		zap.String("organization_id", req.OrgID.String()),
		zap.String("business_unit_id", req.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	now := time.Now().Unix()

	results, rErr := db.NewUpdate().Model((*notification.Notification)(nil)).
		Set("read_at = ?", now).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("notif.organization_id = ?", req.OrgID).
				Where("notif.business_unit_id = ?", req.BuID).
				Where("notif.target_user_id = ?", req.UserID)
		}).
		Exec(ctx)

	if rErr != nil {
		log.Error("failed to read all notifications", zap.Error(rErr))
		return rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Notification", req.UserID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) MarkAsRead(
	ctx context.Context,
	req repositories.MarkAsReadRequest,
) error {
	log := r.l.With(
		zap.String("operation", "MarkAsRead"),
		zap.String("notification_id", req.NotificationID.String()),
		zap.String("user_id", req.UserID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	now := utils.NowUnix()

	result, err := db.NewUpdate().
		Model((*notification.Notification)(nil)).
		Set("read_at = ?", now).
		WhereGroup(" AND", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("notif.id = ?", req.NotificationID).
				Where("notif.organization_id = ?", req.OrgID).
				Where("notif.business_unit_id = ?", req.BuID).
				WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
					return sq.
						WhereOr("notif.channel = ?", notification.ChannelGlobal).
						WhereOr("notif.channel = ? AND notif.target_user_id = ?", notification.ChannelUser, req.UserID)
				}).
				Where("notif.read_at IS NULL")
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to mark notification as read", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "Notification", req.NotificationID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) MarkAsDismissed(
	ctx context.Context,
	req repositories.MarkAsDismissedRequest,
) error {
	log := r.l.With(
		zap.String("operation", "MarkAsDismissed"),
		zap.String("notification_id", req.NotificationID.String()),
		zap.String("user_id", req.UserID.String()),
	)

	dba, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	results, rErr := dba.NewUpdate().
		Model((*notification.Notification)(nil)).
		Set("dismissed_at = ?", utils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.
				Where("notif.id = ?", req.NotificationID).
				Where("notif.organization_id = ?", req.OrgID).
				Where("notif.business_unit_id = ?", req.BuID).
				WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
					return sq.
						WhereOr("notif.channel = ?", notification.ChannelGlobal).
						WhereOr("notif.channel = ? AND notif.target_user_id = ?", notification.ChannelUser, req.UserID).
						Where("notif.read_at IS NULL")
				})
		}).
		Exec(ctx)

	if rErr != nil {
		log.Error("failed to mark notification as dismissed", zap.Error(rErr))
		return rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Notification", req.NotificationID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) MarkAsDelivered(
	ctx context.Context,
	notificationID pulid.ID,
	deliveredAt int64,
) error {
	log := r.l.With(
		zap.String("operation", "MarkAsDelivered"),
		zap.String("notification_id", notificationID.String()),
	)

	dba, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := dba.NewUpdate().
		Model((*notification.Notification)(nil)).
		Set("notif.delivered_at = ?", deliveredAt).
		Set("notif.delivery_status = ?", notification.DeliveryStatusDelivered).
		Where("notif.id = ?", notificationID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to mark notification as delivered", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "Notification", notificationID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) GetPendingRetries(
	ctx context.Context,
	limit int,
) ([]*notification.Notification, error) {
	log := r.l.With(
		zap.String("operation", "GetPendingRetries"),
		zap.Int("limit", limit),
	)

	dba, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	notifications := make([]*notification.Notification, 0, limit)
	err = dba.NewSelect().
		Model(&notifications).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("notif.delivery_status = ?", notification.DeliveryStatusFailed).
				Where("notif.retry_count < notif.max_retries").
				Where("(notif.expires_at IS NULL OR notif.expires_at > extract(epoch from current_timestamp)::bigint)")
		}).
		Order("notif.created_at ASC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get pending retries", zap.Error(err))
		return nil, err
	}

	return notifications, nil
}

func (r *repository) GetExpiredNotifications(
	ctx context.Context,
	limit int,
) ([]*notification.Notification, error) {
	log := r.l.With(
		zap.String("operation", "GetExpiredNotifications"),
		zap.Int("limit", limit),
	)

	dba, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	notifications := make([]*notification.Notification, 0, limit)
	err = dba.NewSelect().
		Model(&notifications).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("notif.expires_at IS NOT NULL").
				Where("notif.expires_at <= extract(epoch from current_timestamp)::bigint").
				Where("notif.delivery_status != ?", notification.DeliveryStatusExpired)
		}).
		Order("notif.expires_at ASC").
		Limit(limit).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get expired notifications", zap.Error(err))
		return nil, err
	}

	return notifications, nil
}

func (r *repository) DeleteOldNotifications(
	ctx context.Context,
	olderThan int64,
) error {
	log := r.l.With(
		zap.String("operation", "DeleteOldNotifications"),
		zap.Int64("older_than", olderThan),
	)

	dba, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	_, err = dba.NewDelete().
		Model((*notification.Notification)(nil)).
		Where("notif.created_at < ?", olderThan).
		Where("(notif.read_at IS NOT NULL OR notif.dismissed_at IS NOT NULL)").
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete old notifications", zap.Error(err))
		return err
	}

	return nil
}
