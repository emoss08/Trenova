package notificationrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification/notificationdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) notificationports.NotificationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.notification-repository"),
	}
}

func notifications(ctx context.Context, orgID pulid.ID) *keel.Query[notification.Notification] {
	n := notificationdb.Notifications
	if dbscope.IsSystem(ctx) {
		return n.AllTenants().Where(n.OrganizationID.Eq(orgID))
	}

	return n.Query()
}

func audience(tenantInfo pagination.TenantInfo, personalOnly bool) keel.Pred {
	n := notificationdb.Notifications
	personal := keel.And(
		n.TargetUserID.Eq(tenantInfo.UserID),
		n.BusinessUnitID.Eq(tenantInfo.BuID),
	)
	if personalOnly {
		return personal
	}

	return keel.Or(personal, n.Channel.Eq(notification.ChannelGlobal))
}

func stateFilter(req *notificationports.ListNotificationConnectionRequest) []keel.Pred {
	n := notificationdb.Notifications
	dismissed := n.DismissedAt.IsNull()
	if req.State == notification.StateArchived {
		dismissed = n.DismissedAt.IsNotNull()
	}

	return []keel.Pred{dismissed, keel.When(req.UnreadOnly, n.ReadAt.IsNull())}
}

func (r *repository) ExistsRecent(
	ctx context.Context,
	req notificationports.ExistsRecentNotificationRequest,
) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		n := notificationdb.Notifications
		return notifications(ctx, req.OrganizationID).Where(
			n.BusinessUnitID.Eq(req.BusinessUnitID),
			n.EventType.Eq(req.EventType),
			n.CorrelationID.Eq(req.CorrelationID),
			n.CreatedAt.Gte(req.Since),
		).Exists(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*notification.Notification, error) {
		if err := notificationdb.Notifications.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create notification", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) List(
	ctx context.Context,
	req *notificationports.ListNotificationsRequest,
) (*pagination.ListResult[*notification.Notification], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*notification.Notification], error) {
			n := notificationdb.Notifications
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				notifications(ctx, req.Filter.TenantInfo.OrgID).
					Where(audience(req.Filter.TenantInfo, false)),
				&listquery.Request[notification.Notification]{
					Filter: req.Filter,
					Then:   []keel.Order{n.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list notifications", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *notificationports.ListNotificationConnectionRequest,
) (*pagination.CursorListResult[*notification.Notification], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*notification.Notification], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				notifications(ctx, req.Filter.TenantInfo.OrgID).
					Where(stateFilter(req)...).
					Where(audience(req.Filter.TenantInfo, req.PersonalOnly)),
				&listquery.Request[notification.Notification]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.NotificationColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list notification connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) action(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
	preds ...keel.Pred,
) *keel.Query[notification.Notification] {
	n := notificationdb.Notifications
	return notifications(ctx, req.TenantInfo.OrgID).
		Where(n.ID.In(req.IDs...), audience(req.TenantInfo, req.PersonalOnly)).
		Where(preds...)
}

func (r *repository) update(
	ctx context.Context,
	operation string,
	q *keel.Query[notification.Notification],
	sets ...keel.Assignment,
) error {
	n := notificationdb.Notifications
	sets = append(sets, n.UpdatedAt.ToExpr(n.UpdatedAt))
	if _, err := q.Update(ctx, r.db.Keel(ctx), sets...); err != nil {
		r.l.Error("failed to "+operation, zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) MarkAsRead(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n := notificationdb.Notifications
		return r.update(
			ctx,
			"mark notifications as read",
			r.action(ctx, req, n.ReadAt.IsNull()),
			n.ReadAt.To(timeutils.NowUnix()),
		)
	})
}

func (r *repository) MarkAsUnread(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n := notificationdb.Notifications
		return r.update(
			ctx,
			"mark notifications as unread",
			r.action(ctx, req, n.ReadAt.IsNotNull()),
			n.ReadAt.ToNull(),
		)
	})
}

func (r *repository) Dismiss(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n := notificationdb.Notifications
		now := timeutils.NowUnix()
		return r.update(
			ctx,
			"dismiss notifications",
			r.action(ctx, req, n.DismissedAt.IsNull()),
			n.DismissedAt.To(now),
			n.ReadAt.ToExpr(keel.Raw("COALESCE(?, ?::bigint)", n.ReadAt, now)),
		)
	})
}

func (r *repository) Restore(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n := notificationdb.Notifications
		return r.update(
			ctx,
			"restore notifications",
			r.action(ctx, req, n.DismissedAt.IsNotNull()),
			n.DismissedAt.ToNull(),
		)
	})
}

func (r *repository) MarkAllAsRead(
	ctx context.Context,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
	personalOnly bool,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n := notificationdb.Notifications
		ti := tenantInfo
		ti.UserID = userID

		return r.update(
			ctx,
			"mark all notifications as read",
			notifications(ctx, tenantInfo.OrgID).Where(
				n.BusinessUnitID.Eq(tenantInfo.BuID),
				audience(ti, personalOnly),
				n.ReadAt.IsNull(),
				n.DismissedAt.IsNull(),
			),
			n.ReadAt.To(timeutils.NowUnix()),
		)
	})
}

func (r *repository) CountUnread(
	ctx context.Context,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
	personalOnly bool,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		n := notificationdb.Notifications
		ti := tenantInfo
		ti.UserID = userID

		count, err := notifications(ctx, tenantInfo.OrgID).Where(
			n.BusinessUnitID.Eq(tenantInfo.BuID),
			n.ReadAt.IsNull(),
			n.DismissedAt.IsNull(),
			audience(ti, personalOnly),
		).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count unread notifications", zap.Error(err))
			return 0, err
		}

		return count, nil
	})
}
