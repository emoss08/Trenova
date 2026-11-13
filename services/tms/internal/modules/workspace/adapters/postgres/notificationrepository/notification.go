package notificationrepository

import (
	"context"
	"errors"

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

const (
	openSubjectConstraint = "uq_notifications_open_subject"
	countCap              = 1000
	ascending             = "asc"
	descending            = "desc"
)

func notifications(ctx context.Context, orgID pulid.ID) keel.Query[*notification.Notification] {
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

func open(now int64) keel.Pred {
	n := notificationdb.Notifications

	return keel.And(
		n.DismissedAt.IsNull(),
		keel.Or(n.ExpiresAt.IsNull(), n.ExpiresAt.Gt(now)),
	)
}

func lensFilter(lens notification.Lens, now int64) keel.Pred {
	n := notificationdb.Notifications
	switch lens {
	case notification.LensDone:
		return n.DismissedAt.IsNotNull()
	case notification.LensLater:
		return keel.And(open(now), n.SurfacesAt.Gt(now))
	case notification.LensForYou:
		return keel.And(open(now), n.SurfacesAt.Lte(now), n.Reason.Eq(notification.ReasonDirect))
	case notification.LensTeam:
		return keel.And(open(now), n.SurfacesAt.Lte(now), n.Reason.Eq(notification.ReasonTeam))
	case notification.LensOpen:
		return keel.And(open(now), n.SurfacesAt.Lte(now))
	default:
		return keel.And(open(now), n.SurfacesAt.Lte(now))
	}
}

func lensKeyset(lens notification.Lens) *listquery.Keyset {
	n := notificationdb.Notifications
	switch lens {
	case notification.LensLater:
		return keysetBy("surfacesAt", ascending, n.SurfacesAt.Asc(), n.ID.Asc())
	case notification.LensDone:
		return keysetBy("dismissedAt", descending, n.DismissedAt.Desc(), n.ID.Desc())
	case notification.LensForYou, notification.LensTeam, notification.LensOpen:
		return keysetBy("surfacesAt", descending, n.SurfacesAt.Desc(), n.ID.Desc())
	default:
		return keysetBy("surfacesAt", descending, n.SurfacesAt.Desc(), n.ID.Desc())
	}
}

func keysetBy(field, direction string, at, id keel.Order) *listquery.Keyset {
	return &listquery.Keyset{
		Orders: []keel.Order{at, id},
		Shape: []pagination.CursorSortField{
			{Field: field, Direction: direction},
			{Field: "id", Direction: direction},
		},
	}
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

func (r *repository) NotifiedUsers(
	ctx context.Context,
	req notificationports.ExistsRecentNotificationRequest,
) ([]pulid.ID, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		n := notificationdb.Notifications
		return keel.Into[pulid.ID](ctx, r.db.Keel(ctx), notifications(ctx, req.OrganizationID).
			Pick(n.TargetUserID).
			Where(
				n.BusinessUnitID.Eq(req.BusinessUnitID),
				n.EventType.Eq(req.EventType),
				n.CorrelationID.Eq(req.CorrelationID),
				n.CreatedAt.Gte(req.Since),
				n.TargetUserID.IsNotNull(),
			))
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	created, err := r.createOrAbsorb(ctx, entity)
	if keel.IsUnique(err, openSubjectConstraint) {
		return r.createOrAbsorb(ctx, entity)
	}

	return created, err
}

func (r *repository) createOrAbsorb(
	ctx context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*notification.Notification, error) {
		if !entity.Collapses() {
			entity.FoldOccurrence(nil)

			return r.insert(ctx, entity)
		}

		existing, err := r.openSubject(ctx, entity)
		if errors.Is(err, keel.ErrNotFound) {
			incoming := *entity
			incoming.FoldOccurrence(nil)
			created, createErr := r.insert(ctx, &incoming)
			if createErr != nil {
				return nil, createErr
			}

			return created, nil
		}
		if err != nil {
			return nil, err
		}

		existing.Absorb(entity, timeutils.NowUnix())
		if err = notificationdb.Notifications.Save(ctx, r.db.Keel(ctx), existing); err != nil {
			r.l.Error("failed to update the open notification of a subject", zap.Error(err))
			return nil, err
		}

		return existing, nil
	})
}

func (r *repository) insert(
	ctx context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	if err := notificationdb.Notifications.Create(ctx, r.db.Keel(ctx), entity); err != nil {
		if !keel.IsUnique(err, openSubjectConstraint) {
			r.l.Error("failed to create notification", zap.Error(err))
		}

		return nil, err
	}

	return entity, nil
}

func (r *repository) openSubject(
	ctx context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	n := notificationdb.Notifications
	existing, err := notifications(ctx, entity.OrganizationID).Where(
		n.TargetUserID.Eq(*entity.TargetUserID),
		n.SubjectKey.Eq(entity.SubjectKey),
		n.DismissedAt.IsNull(),
	).Lock(keel.ForUpdate).One(ctx, r.db.Keel(ctx))
	if err != nil {
		if !errors.Is(err, keel.ErrNotFound) {
			r.l.Error("failed to read the open notification of a subject", zap.Error(err))
		}

		return nil, err
	}

	return existing, nil
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
					Where(
						lensFilter(req.Lens, req.Now),
						keel.When(req.UnreadOnly, notificationdb.Notifications.ReadAt.IsNull()),
						audience(req.Filter.TenantInfo, req.PersonalOnly),
					),
				&listquery.Request[notification.Notification]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.NotificationColumns,
					Keyset:  lensKeyset(req.Lens),
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
) keel.Query[*notification.Notification] {
	n := notificationdb.Notifications
	return notifications(ctx, req.TenantInfo.OrgID).
		Where(n.ID.In(req.IDs...), audience(req.TenantInfo, req.PersonalOnly)).
		Where(preds...)
}

func (r *repository) update(
	ctx context.Context,
	operation string,
	q keel.Query[*notification.Notification],
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
			n.ReadAt.ToExpr(n.ReadAt.Coalesce(now)),
		)
	})
}

func (r *repository) DismissAll(
	ctx context.Context,
	req notificationports.DismissAllNotificationsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n := notificationdb.Notifications
		return r.update(
			ctx,
			"dismiss every notification of a lens",
			notifications(ctx, req.TenantInfo.OrgID).Where(
				lensFilter(req.Lens, req.Now),
				n.DismissedAt.IsNull(),
				audience(req.TenantInfo, req.PersonalOnly),
			),
			n.DismissedAt.To(req.Now),
			n.ReadAt.ToExpr(n.ReadAt.Coalesce(req.Now)),
		)
	})
}

func (r *repository) Restore(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n := notificationdb.Notifications
		now := timeutils.NowUnix()
		return r.update(
			ctx,
			"restore notifications",
			r.action(ctx, req, n.DismissedAt.IsNotNull()),
			n.DismissedAt.ToNull(),
			n.ResolvedAt.ToNull(),
			n.Resolution.To(""),
			n.SurfacesAt.To(now),
		)
	})
}

func (r *repository) Snooze(
	ctx context.Context,
	req notificationports.SnoozeNotificationsRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n := notificationdb.Notifications
		return r.update(
			ctx,
			"set notifications aside",
			r.action(ctx, req.NotificationActionRequest, n.DismissedAt.IsNull()),
			n.SurfacesAt.To(req.Until),
			n.ReadAt.ToNull(),
		)
	})
}

func (r *repository) Wake(
	ctx context.Context,
	req notificationports.NotificationActionRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		n := notificationdb.Notifications
		now := timeutils.NowUnix()
		return r.update(
			ctx,
			"bring notifications back",
			r.action(ctx, req, n.DismissedAt.IsNull(), n.SurfacesAt.Gt(now)),
			n.SurfacesAt.To(now),
		)
	})
}

func (r *repository) ResolveSubject(
	ctx context.Context,
	req notificationports.ResolveNotificationSubjectRequest,
) (int64, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		n := notificationdb.Notifications
		resolved, err := notifications(ctx, req.OrganizationID).Where(
			n.BusinessUnitID.Eq(req.BusinessUnitID),
			n.SubjectKey.Eq(req.SubjectKey),
			n.DismissedAt.IsNull(),
		).Update(
			ctx,
			r.db.Keel(ctx),
			n.DismissedAt.To(req.Now),
			n.ResolvedAt.To(req.Now),
			n.Resolution.To(req.Resolution),
			n.ReadAt.ToExpr(n.ReadAt.Coalesce(req.Now)),
			n.UpdatedAt.To(req.Now),
		)
		if err != nil {
			r.l.Error("failed to resolve the notifications of a subject", zap.Error(err))
			return 0, err
		}

		return resolved, nil
	})
}

func (r *repository) Counts(
	ctx context.Context,
	req notificationports.NotificationCountsRequest,
) (*notificationports.NotificationCounts, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*notificationports.NotificationCounts, error) {
			n := notificationdb.Notifications
			lens := func(lens notification.Lens, preds ...keel.Pred) keel.Query[*notification.Notification] {
				return notifications(ctx, req.TenantInfo.OrgID).Where(
					lensFilter(lens, req.Now),
					audience(req.TenantInfo, req.PersonalOnly),
				).Where(preds...)
			}

			counts := new(notificationports.NotificationCounts)
			if err := keel.Batch(ctx, r.db.Keel(ctx),
				keel.CountUpToInto(lens(notification.LensForYou), countCap, &counts.ForYou),
				keel.CountUpToInto(
					lens(notification.LensForYou, n.ReadAt.IsNull()),
					countCap,
					&counts.ForYouUnread,
				),
				keel.CountUpToInto(lens(notification.LensTeam), countCap, &counts.Team),
				keel.CountUpToInto(
					lens(notification.LensTeam, n.ReadAt.IsNull()),
					countCap,
					&counts.TeamUnread,
				),
				keel.CountUpToInto(lens(notification.LensLater), countCap, &counts.Later),
			); err != nil {
				r.l.Error("failed to count notifications", zap.Error(err))
				return nil, err
			}

			return counts, nil
		},
	)
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
			lensFilter(notification.LensOpen, timeutils.NowUnix()),
			audience(ti, personalOnly),
		).CountUpTo(ctx, r.db.Keel(ctx), countCap)
		if err != nil {
			r.l.Error("failed to count unread notifications", zap.Error(err))
			return 0, err
		}

		return count, nil
	})
}
