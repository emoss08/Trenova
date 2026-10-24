package notificationpreferencerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification/notificationdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
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
			p := notificationdb.Preferences
			entity, err := p.Where(p.UserID.Eq(req.TenantInfo.UserID)).One(ctx, r.db.Keel(ctx))
			if dberror.IsNotFoundError(err) {
				return nil, false, nil
			}
			if err != nil {
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
		p := notificationdb.Preferences
		db := r.db.Keel(ctx)
		inserted, err := p.Upsert(
			ctx,
			db,
			[]*notification.Preference{entity},
			keel.OnConflict(p.OrganizationID, p.BusinessUnitID, p.UserID).Update(
				p.DisabledEventTypes,
				p.DesktopMode,
				p.EmailMode,
				p.EmailDigest,
				p.Sound,
				p.QuietHoursEnabled,
				p.QuietHoursStart,
				p.QuietHoursEnd,
				p.QuietHoursDays,
				p.QuietHoursAllowUrgent,
				p.PausedUntil,
			),
		)
		if err == nil && !inserted[0] {
			var stored *notification.Preference
			stored, err = p.Where(p.UserID.Eq(entity.UserID)).One(ctx, db)
			if err == nil {
				entity = stored
			}
		}
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
		p := notificationdb.Preferences
		entities, err := p.AllTenants().
			Where(p.EmailMode.Eq(notification.DeliveryModeAll), p.EmailDigest.IsTrue()).
			All(ctx, r.db.Keel(ctx))
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
		p := notificationdb.Preferences
		_, err := p.Where(p.UserID.Eq(req.TenantInfo.UserID)).Update(
			ctx,
			r.db.Keel(ctx),
			p.LastDigestAt.To(req.SentAt),
			p.Version.ToExpr(p.Version),
			p.UpdatedAt.ToExpr(p.UpdatedAt),
		)
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
		n := notificationdb.Notifications
		limit := req.Limit
		if limit <= 0 || limit > maxDigestNotifications {
			limit = maxDigestNotifications
		}

		entities, err := n.Where(
			n.TargetUserID.Eq(req.TenantInfo.UserID),
			n.CreatedAt.Gt(req.Since),
			n.CreatedAt.Lte(req.Until),
			n.Data.Text("delivery", "email").Eq(string(notification.EmailDeliveryDigest)),
			n.DismissedAt.IsNull(),
		).
			Order(n.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list digest notifications", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
