package notificationpreferencerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/buncolgen/notificationcol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxDigestNotifications = 50

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) notificationports.PreferenceRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.notification-preference-repository"),
	}
}

func (r *repository) Get(
	ctx context.Context,
	req notificationports.GetPreferenceRequest,
) (*notification.Preference, bool, error) {
	return dbtx.Read2(
		ctx,
		r.db,
		func(ctx context.Context) (*notification.Preference, bool, error) {
			cols := notificationcol.PreferenceColumns
			entity := new(notification.Preference)
			err := r.db.DBForContext(ctx).
				NewSelect().
				Model(entity).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return notificationcol.PreferenceScopeTenant(sq, req.TenantInfo)
				}).
				Where(cols.UserID.Eq(), req.TenantInfo.UserID).
				Scan(ctx)
			if err != nil {
				if dberror.IsNotFoundError(err) {
					return nil, false, nil
				}
				r.l.Error("failed to get notification preference",
					zap.String("userID", req.TenantInfo.UserID.String()),
					zap.Error(err),
				)
				return nil, false, err
			}

			return entity, true, nil
		},
	)
}

func (r *repository) Upsert(
	ctx context.Context,
	entity *notification.Preference,
) (*notification.Preference, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*notification.Preference, error) {
		cols := notificationcol.PreferenceColumns
		_, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On("CONFLICT (" +
				cols.OrganizationID.Bare() + ", " +
				cols.BusinessUnitID.Bare() + ", " +
				cols.UserID.Bare() + ") DO UPDATE").
			Set(cols.DisabledEventTypes.SetExcluded()).
			Set(cols.DesktopMode.SetExcluded()).
			Set(cols.EmailMode.SetExcluded()).
			Set(cols.EmailDigest.SetExcluded()).
			Set(cols.Sound.SetExcluded()).
			Set(cols.QuietHoursEnabled.SetExcluded()).
			Set(cols.QuietHoursStart.SetExcluded()).
			Set(cols.QuietHoursEnd.SetExcluded()).
			Set(cols.QuietHoursDays.SetExcluded()).
			Set(cols.QuietHoursAllowUrgent.SetExcluded()).
			Set(cols.PausedUntil.SetExcluded()).
			Set(cols.UpdatedAt.SetExcluded()).
			Set(cols.Version.IncConflict(1)).
			Returning("*").
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to save notification preference",
				zap.String("userID", entity.UserID.String()),
				zap.Error(err),
			)
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) ListDigestSubscribers(
	ctx context.Context,
) ([]*notification.Preference, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"find every person, in every organization, who bundles notification emails into a daily digest",
	)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*notification.Preference, error) {
		cols := notificationcol.PreferenceColumns
		entities := make([]*notification.Preference, 0)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Where(cols.EmailMode.Eq(), notification.DeliveryModeAll).
			Where(cols.EmailDigest.IsTrue()).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list digest subscribers", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}

func (r *repository) MarkDigestSent(
	ctx context.Context,
	req notificationports.MarkDigestSentRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := notificationcol.PreferenceColumns
		_, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*notification.Preference)(nil)).
			Set(cols.LastDigestAt.Set(), req.SentAt).
			Apply(func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return notificationcol.PreferenceScopeTenantUpdate(uq, req.TenantInfo)
			}).
			Where(cols.UserID.Eq(), req.TenantInfo.UserID).
			Exec(ctx)
		if err != nil {
			r.l.Error("failed to mark digest sent",
				zap.String("userID", req.TenantInfo.UserID.String()),
				zap.Error(err),
			)
		}
		return err
	})
}

func (r *repository) ListDigestNotifications(
	ctx context.Context,
	req notificationports.ListDigestNotificationsRequest,
) ([]*notification.Notification, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*notification.Notification, error) {
		cols := notificationcol.NotificationColumns
		limit := req.Limit
		if limit <= 0 || limit > maxDigestNotifications {
			limit = maxDigestNotifications
		}

		entities := make([]*notification.Notification, 0, limit)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&entities).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				return notificationcol.NotificationScopeTenant(sq, req.TenantInfo)
			}).
			Where(cols.TargetUserID.Eq(), req.TenantInfo.UserID).
			Where(cols.CreatedAt.Gt(), req.Since).
			Where(cols.CreatedAt.Lte(), req.Until).
			Where(cols.Data.Expr("{}->'delivery'->>'email'")+" = ?", string(notification.EmailDeliveryDigest)).
			Where(cols.DismissedAt.IsNull()).
			Order(cols.CreatedAt.OrderDesc()).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list digest notifications", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
