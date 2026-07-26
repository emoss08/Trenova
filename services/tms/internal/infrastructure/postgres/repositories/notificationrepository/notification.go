package notificationrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dbhelper"
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

var cols = buncolgen.NotificationColumns

func (r *repository) scopeFilter(
	q *bun.SelectQuery,
	tenantInfo pagination.TenantInfo,
	personalOnly bool,
) *bun.SelectQuery {
	if personalOnly {
		return q.
			Where(cols.TargetUserID.Eq(), tenantInfo.UserID).
			Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID)
	}

	return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.
			WhereGroup(" AND ", func(inner *bun.SelectQuery) *bun.SelectQuery {
				return inner.
					Where(cols.TargetUserID.Eq(), tenantInfo.UserID).
					Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID)
			}).
			WhereOr(cols.Channel.Eq(), notification.ChannelGlobal)
	})
}

func (r *repository) scopeActionQuery(
	q *bun.UpdateQuery,
	tenantInfo pagination.TenantInfo,
	personalOnly bool,
) *bun.UpdateQuery {
	if personalOnly {
		return q.
			Where(cols.TargetUserID.Eq(), tenantInfo.UserID).
			Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID)
	}

	return q.WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
		return uq.
			WhereGroup(" AND ", func(inner *bun.UpdateQuery) *bun.UpdateQuery {
				return inner.
					Where(cols.TargetUserID.Eq(), tenantInfo.UserID).
					Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID)
			}).
			WhereOr(cols.Channel.Eq(), notification.ChannelGlobal)
	})
}

func (r *repository) ExistsRecent(
	ctx context.Context,
	req repositories.ExistsRecentNotificationRequest,
) (bool, error) {
	return r.db.DBForContext(ctx).
		NewSelect().
		Model((*notification.Notification)(nil)).
		Where(cols.OrganizationID.Eq(), req.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), req.BusinessUnitID).
		Where(cols.EventType.Eq(), req.EventType).
		Where(cols.CorrelationID.Eq(), req.CorrelationID).
		Where(cols.CreatedAt.Gte(), req.Since).
		Exists(ctx)
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
		buncolgen.NotificationTable.Alias,
		req.Filter,
		(*notification.Notification)(nil),
	)

	q = q.Where(cols.OrganizationID.Eq(), req.Filter.TenantInfo.OrgID)
	q = r.scopeFilter(q, req.Filter.TenantInfo, false)
	q = q.Order(cols.CreatedAt.OrderDesc())
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

func applyNotificationColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return q.ColumnExpr(buncolgen.NotificationTable.All())
	}

	return q.Column(columns...)
}

func stateFilter(
	q *bun.SelectQuery,
	req *repositories.ListNotificationConnectionRequest,
) *bun.SelectQuery {
	if req.State == notification.StateArchived {
		q = q.Where(cols.DismissedAt.IsNotNull())
	} else {
		q = q.Where(cols.DismissedAt.IsNull())
	}

	if req.UnreadOnly {
		q = q.Where(cols.ReadAt.IsNull())
	}

	return q
}

func (r *repository) applyCursorPageFilters(
	q *bun.SelectQuery,
	req *repositories.ListNotificationConnectionRequest,
) (*bun.SelectQuery, error) {
	q, err := querybuilder.ApplyCursorFilters(
		q,
		buncolgen.NotificationTable.Alias,
		req.Filter,
		req.Cursor,
		(*notification.Notification)(nil),
	)
	if err != nil {
		return nil, err
	}

	q = q.Where(cols.OrganizationID.Eq(), req.Filter.TenantInfo.OrgID)
	q = stateFilter(q, req)
	return r.scopeFilter(q, req.Filter.TenantInfo, req.PersonalOnly), nil
}

func (r *repository) applyTotalCountFilters(
	q *bun.SelectQuery,
	req *repositories.ListNotificationConnectionRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		buncolgen.NotificationTable.Alias,
		req.Filter,
		(*notification.Notification)(nil),
	)

	q = q.Where(cols.OrganizationID.Eq(), req.Filter.TenantInfo.OrgID)
	q = stateFilter(q, req)
	return r.scopeFilter(q, req.Filter.TenantInfo, req.PersonalOnly)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListNotificationConnectionRequest,
) (*pagination.CursorListResult[*notification.Notification], error) {
	log := r.l.With(zap.String("operation", "ListConnection"))

	dba := r.db.DBForContext(ctx)
	total, err := dba.
		NewSelect().
		Model((*notification.Notification)(nil)).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.applyTotalCountFilters(sq, req)
		}).
		Count(ctx)
	if err != nil {
		log.Error("failed to count notifications", zap.Error(err))
		return nil, err
	}

	result, err := dbhelper.CursorList(
		ctx,
		dbhelper.CursorListParams[*notification.Notification]{
			Filter:     req.Filter,
			Cursor:     req.Cursor,
			TotalCount: &total,
			Query: func(entities *[]*notification.Notification) *bun.SelectQuery {
				return dba.
					NewSelect().
					Model(entities).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return applyNotificationColumns(sq, req.NotificationColumns)
					})
			},
			Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
				return r.applyCursorPageFilters(sq, req)
			},
		})
	if err != nil {
		log.Error("failed to scan notifications", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (r *repository) actionQuery(req repositories.NotificationActionRequest) *bun.UpdateQuery {
	q := r.db.DB().
		NewUpdate().
		Model((*notification.Notification)(nil)).
		Where(cols.ID.In(), bun.List(req.IDs)).
		Where(cols.OrganizationID.Eq(), req.TenantInfo.OrgID)
	return r.scopeActionQuery(q, req.TenantInfo, req.PersonalOnly)
}

func (r *repository) MarkAsRead(
	ctx context.Context,
	req repositories.NotificationActionRequest,
) error {
	log := r.l.With(zap.String("operation", "MarkAsRead"))

	_, err := r.actionQuery(req).
		Set(cols.ReadAt.Set(), timeutils.NowUnix()).
		Where(cols.ReadAt.IsNull()).
		Exec(ctx)
	if err != nil {
		log.Error("failed to mark notifications as read", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) MarkAsUnread(
	ctx context.Context,
	req repositories.NotificationActionRequest,
) error {
	log := r.l.With(zap.String("operation", "MarkAsUnread"))

	_, err := r.actionQuery(req).
		Set(cols.ReadAt.SetNull()).
		Where(cols.ReadAt.IsNotNull()).
		Exec(ctx)
	if err != nil {
		log.Error("failed to mark notifications as unread", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Dismiss(
	ctx context.Context,
	req repositories.NotificationActionRequest,
) error {
	log := r.l.With(zap.String("operation", "Dismiss"))

	now := timeutils.NowUnix()
	_, err := r.actionQuery(req).
		Set(cols.DismissedAt.Set(), now).
		Set(cols.ReadAt.SetExpr("COALESCE({}, ?)"), now).
		Where(cols.DismissedAt.IsNull()).
		Exec(ctx)
	if err != nil {
		log.Error("failed to dismiss notifications", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Restore(
	ctx context.Context,
	req repositories.NotificationActionRequest,
) error {
	log := r.l.With(zap.String("operation", "Restore"))

	_, err := r.actionQuery(req).
		Set(cols.DismissedAt.SetNull()).
		Where(cols.DismissedAt.IsNotNull()).
		Exec(ctx)
	if err != nil {
		log.Error("failed to restore notifications", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) MarkAllAsRead(
	ctx context.Context,
	userID pulid.ID,
	tenantInfo pagination.TenantInfo,
	personalOnly bool,
) error {
	log := r.l.With(zap.String("operation", "MarkAllAsRead"))

	q := r.db.DB().
		NewUpdate().
		Model((*notification.Notification)(nil)).
		Set(cols.ReadAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.NotificationScopeTenantUpdate(uq, tenantInfo)
		})

	ti := tenantInfo
	ti.UserID = userID
	q = r.scopeActionQuery(q, ti, personalOnly)

	_, err := q.
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.
				Where(cols.ReadAt.IsNull()).
				Where(cols.DismissedAt.IsNull())
		}).
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
	personalOnly bool,
) (int64, error) {
	log := r.l.With(zap.String("operation", "CountUnread"))

	ti := tenantInfo
	ti.UserID = userID

	q := r.db.DB().
		NewSelect().
		Model((*notification.Notification)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.NotificationScopeTenant(sq, ti).
				Where(cols.ReadAt.IsNull()).
				Where(cols.DismissedAt.IsNull())
		})

	q = r.scopeFilter(q, ti, personalOnly)

	count, err := q.Count(ctx)
	if err != nil {
		log.Error("failed to count unread notifications", zap.Error(err))
		return 0, err
	}

	return int64(count), nil
}
