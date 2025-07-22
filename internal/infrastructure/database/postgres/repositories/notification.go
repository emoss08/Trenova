package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type NotificationRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type notificationRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewNotificationRepository(p NotificationRepositoryParams) repositories.NotificationRepository {
	log := p.Logger.With().
		Str("repository", "notification").
		Logger()

	return &notificationRepository{
		db: p.DB,
		l:  &log,
	}
}

func (nr *notificationRepository) Create(
	ctx context.Context,
	notif *notification.Notification,
) error {
	log := nr.l.With().
		Str("operation", "Create").
		Str("notification_id", notif.ID.String()).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return eris.Wrap(err, "get database connection")
	}

	if _, err = dba.NewInsert().Model(notif).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error().Err(err).Interface("notification", notif).Msg("failed to insert notification")
		return eris.Wrap(err, "insert notification")
	}

	log.Info().Msg("notification created successfully")
	return nil
}

func (nr *notificationRepository) Update(
	ctx context.Context,
	notif *notification.Notification,
) error {
	log := nr.l.With().
		Str("operation", "Update").
		Str("notification_id", notif.ID.String()).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return eris.Wrap(err, "get database connection")
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := notif.Version
		notif.Version++

		results, rErr := tx.NewUpdate().
			Model(notif).
			WherePK().
			Where("notif.version = ?", ov).
			OmitZero().
			Returning("*").
			Exec(c)

		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update notification")
			return eris.Wrap(rErr, "update notification")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Notification (%s) has either been updated or deleted since the last request.",
					notif.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		return err
	}

	log.Info().Msg("notification updated successfully")
	return nil
}

func (nr *notificationRepository) GetByID(
	ctx context.Context,
	id pulid.ID,
) (*notification.Notification, error) {
	log := nr.l.With().
		Str("operation", "GetByID").
		Str("notification_id", id.String()).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return nil, eris.Wrap(err, "get database connection")
	}

	notif := new(notification.Notification)
	err = dba.NewSelect().
		Model(notif).
		Where("notif.id = ?", id).
		Scan(ctx)

	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Info().Msg("notification not found")
			return nil, errors.NewNotFoundError("Notification not found")
		}
		log.Error().Err(err).Msg("failed to get notification")
		return nil, eris.Wrap(err, "get notification")
	}

	log.Info().Msg("notification retrieved successfully")
	return notif, nil
}

func (nr *notificationRepository) buildUserNotificationsQuery(
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

func (nr *notificationRepository) GetUserNotifications(
	ctx context.Context,
	req *repositories.GetUserNotificationsRequest,
) (*ports.ListResult[*notification.Notification], error) {
	log := nr.l.With().
		Str("operation", "GetUserNotifications").
		Str("user_id", req.Filter.TenantOpts.UserID.String()).
		Str("organization_id", req.Filter.TenantOpts.OrgID.String()).
		Bool("unread_only", req.UnreadOnly).
		Logger()

	log.Info().Interface("req", req).Msg("Current request")

	dba, err := nr.db.ReadDB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return nil, eris.Wrap(err, "get database connection")
	}

	notifications := make([]*notification.Notification, 0)

	// * Build the notifications query based on the request.
	q := dba.NewSelect().
		Model(&notifications).
		ApplyQueryBuilder(func(qb bun.QueryBuilder) bun.QueryBuilder {
			return nr.buildUserNotificationsQuery(qb, req)
		})

	// * Order by creation date, newest first and apply pagination.
	q = q.Order("notif.created_at DESC").
		Limit(req.Filter.Limit).
		Offset(req.Filter.Offset)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan notifications")
		return nil, eris.Wrap(err, "scan notifications")
	}

	return &ports.ListResult[*notification.Notification]{
		Items: notifications,
		Total: total,
	}, nil
}

func (nr *notificationRepository) GetUnreadCount(
	ctx context.Context,
	userID pulid.ID,
	organizationID pulid.ID,
) (int, error) {
	log := nr.l.With().
		Str("operation", "GetUnreadCount").
		Str("user_id", userID.String()).
		Str("organization_id", organizationID.String()).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return 0, eris.Wrap(err, "get database connection")
	}

	count, err := dba.NewSelect().
		Model((*notification.Notification)(nil)).
		Where("notif.organization_id = ?", organizationID).
		Where("(notif.channel = ? OR "+
			"(notif.channel = ? AND notif.target_user_id = ?) OR "+
			"(notif.channel = ? AND notif.target_user_id = ?))",
			notification.ChannelGlobal,
			notification.ChannelUser, userID,
			notification.ChannelRole, userID).
		Where("notif.read_at IS NULL").
		Where("(notif.expires_at IS NULL OR notif.expires_at > extract(epoch from current_timestamp)::bigint)").
		Count(ctx)

	if err != nil {
		log.Error().Err(err).Msg("failed to count unread notifications")
		return 0, eris.Wrap(err, "count unread notifications")
	}

	log.Info().Int("count", count).Msg("unread count retrieved successfully")
	return count, nil
}

func (nr *notificationRepository) ReadAllNotifications(
	ctx context.Context,
	req repositories.ReadAllNotificationsRequest,
) error {
	log := nr.l.With().
		Str("operation", "ReadAllNotifications").
		Str("user_id", req.UserID.String()).
		Str("organization_id", req.OrgID.String()).
		Str("business_unit_id", req.BuID.String()).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return oops.In("notification_repository").
			With("op", "ReadAllNotifications").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	now := timeutils.NowUnix()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		results, rErr := tx.NewUpdate().Model((*notification.Notification)(nil)).
			Set("read_at = ?", now).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("notif.organization_id = ?", req.OrgID).
					Where("notif.business_unit_id = ?", req.BuID).
					Where("notif.target_user_id = ?", req.UserID)
			}).
			OmitZero().
			Exec(c)

		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to read all notifications")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			log.Warn().Msg("no notifications found to read")
			return nil
		}

		return nil
	})
	if err != nil {
		return oops.In("notification_repository").
			With("op", "ReadAllNotifications").
			Time(time.Now()).
			Wrapf(err, "read all notifications")
	}

	log.Info().Msg("all notifications read successfully")
	return nil
}

func (nr *notificationRepository) MarkAsRead(
	ctx context.Context,
	req repositories.MarkAsReadRequest,
) error {
	log := nr.l.With().
		Str("operation", "MarkAsRead").
		Str("notification_id", req.NotificationID.String()).
		Str("user_id", req.UserID.String()).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return eris.Wrap(err, "get database connection")
	}

	now := timeutils.NowUnix()

	result, err := dba.NewUpdate().
		Model((*notification.Notification)(nil)).
		Set("read_at = ?", now).
		WhereGroup(" AND", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("notif.id = ?", req.NotificationID).
				Where("notif.organization_id = ?", req.OrgID).
				Where("notif.business_unit_id = ?", req.BuID).
				Where("(channel = ? OR "+
					"(channel = ? AND target_user_id = ?) OR "+
					"(channel = ? AND target_user_id = ?))",
					notification.ChannelGlobal,
					notification.ChannelUser, req.UserID,
					notification.ChannelRole, req.UserID)
		}).
		OmitZero().
		Exec(ctx)

	if err != nil {
		log.Error().Err(err).Msg("failed to mark notification as read")
		return oops.In("notification_repository").
			With("op", "MarkAsRead").
			Time(time.Now()).
			Wrapf(err, "mark notification as read")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return err
	}

	if affected == 0 {
		log.Warn().Msg("no notification found to mark as read")
		return errors.NewNotFoundError("Notification not found or not accessible")
	}

	log.Info().Msg("notification marked as read successfully")
	return nil
}

func (nr *notificationRepository) MarkAsDismissed(
	ctx context.Context,
	req repositories.MarkAsDismissedRequest,
) error {
	log := nr.l.With().
		Str("operation", "MarkAsDismissed").
		Str("notification_id", req.NotificationID.String()).
		Str("user_id", req.UserID.String()).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return eris.Wrap(err, "get database connection")
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		results, rErr := tx.NewUpdate().
			Model((*notification.Notification)(nil)).
			Set("dismissed_at = ?", timeutils.NowUnix()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.
					Where("notif.id = ?", req.NotificationID).
					Where("notif.organization_id = ?", req.OrgID).
					Where("notif.business_unit_id = ?", req.BuID).
					Where("(notif.channel = ? OR "+
						"(notif.channel = ? AND notif.target_user_id = ?) OR "+
						"(notif.channel = ? AND notif.target_user_id = ?))",
						notification.ChannelGlobal,
						notification.ChannelUser, req.UserID,
						notification.ChannelRole, req.UserID)
			}).
			OmitZero().
			Exec(c)

		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to mark notification as dismissed")
			return eris.Wrap(rErr, "mark notification as dismissed")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return err
		}

		if rows == 0 {
			return errors.NewNotFoundError("Notification not found or not accessible")
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to mark notification as dismissed")
		return err
	}

	return nil
}

func (nr *notificationRepository) MarkAsDelivered(
	ctx context.Context,
	notificationID pulid.ID,
	deliveredAt int64,
) error {
	log := nr.l.With().
		Str("operation", "MarkAsDelivered").
		Str("notification_id", notificationID.String()).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return oops.In("notification_repository").
			With("op", "MarkAsDismissed").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	result, err := dba.NewUpdate().
		Model((*notification.Notification)(nil)).
		Set("notif.delivered_at = ?", deliveredAt).
		Set("notif.delivery_status = ?", notification.DeliveryStatusDelivered).
		Where("notif.id = ?", notificationID).
		Exec(ctx)

	if err != nil {
		log.Error().Err(err).Msg("failed to mark notification as delivered")
		return eris.Wrap(err, "mark notification as delivered")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return err
	}

	if affected == 0 {
		log.Warn().Msg("no notification found to mark as delivered")
		return errors.NewNotFoundError("Notification not found")
	}

	log.Info().Msg("notification marked as delivered successfully")
	return nil
}

func (nr *notificationRepository) GetPendingRetries(
	ctx context.Context,
	limit int,
) ([]*notification.Notification, error) {
	log := nr.l.With().
		Str("operation", "GetPendingRetries").
		Int("limit", limit).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return nil, oops.In("notification_repository").
			With("op", "GetPendingRetries").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	notifications := make([]*notification.Notification, 0)
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
		log.Error().Err(err).Msg("failed to get pending retries")
		return nil, oops.In("notification_repository").
			With("op", "GetPendingRetries").
			Time(time.Now()).
			Wrapf(err, "get pending retries")
	}

	log.Info().Int("count", len(notifications)).Msg("pending retries retrieved successfully")
	return notifications, nil
}

func (nr *notificationRepository) GetExpiredNotifications(
	ctx context.Context,
	limit int,
) ([]*notification.Notification, error) {
	log := nr.l.With().
		Str("operation", "GetExpiredNotifications").
		Int("limit", limit).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return nil, eris.Wrap(err, "get database connection")
	}

	notifications := make([]*notification.Notification, 0)
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
		log.Error().Err(err).Msg("failed to get expired notifications")
		return nil, eris.Wrap(err, "get expired notifications")
	}

	log.Info().Int("count", len(notifications)).Msg("expired notifications retrieved successfully")
	return notifications, nil
}

func (nr *notificationRepository) DeleteOldNotifications(
	ctx context.Context,
	olderThan int64,
) error {
	log := nr.l.With().
		Str("operation", "DeleteOldNotifications").
		Int64("older_than", olderThan).
		Logger()

	dba, err := nr.db.DB(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get database connection")
		return oops.In("notification_repository").
			With("op", "DeleteOldNotifications").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	result, err := dba.NewDelete().
		Model((*notification.Notification)(nil)).
		Where("notif.created_at < ?", olderThan).
		Where("(notif.read_at IS NOT NULL OR notif.dismissed_at IS NOT NULL)").
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete old notifications")
		return oops.In("notification_repository").
			With("op", "DeleteOldNotifications").
			Time(time.Now()).
			Wrapf(err, "delete old notifications")
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return err
	}

	log.Info().Int64("deleted_count", affected).Msg("old notifications deleted successfully")
	return nil
}
