package iamrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam/iamdb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.uber.org/zap"
)

const maxSignInHistory = 50

type authEventRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewAuthEventRepository(p Params) iamports.AuthEventRepository {
	return &authEventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.auth-event-repository"),
	}
}

func NewAuthEventHistoryRepository(p Params) iamports.AuthEventHistoryRepository {
	return &authEventRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.auth-event-history-repository"),
	}
}

func (r *authEventRepository) Create(ctx context.Context, event *iam.AuthEvent) error {
	if event.OrganizationID.IsNil() {
		ctx = dbscope.WithSystem(
			ctx,
			"record an authentication attempt that matched no organization",
		)
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := iamdb.AuthEvents.Create(ctx, r.db.Keel(ctx), event); err != nil {
			r.l.Error("failed to insert auth event", zap.Error(err))
			return err
		}

		return nil
	})
}

func (r *authEventRepository) DeleteBefore(ctx context.Context, before int64) (int64, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"delete authentication events past the retention period for every organization",
	)
	return postgres.DeleteUnderAuditRetention(
		ctx,
		r.db,
		func(ctx context.Context, db *keel.DB) (int64, error) {
			ae := iamdb.AuthEvents
			return ae.AllTenants().Where(ae.OccurredAt.Lt(before)).Delete(ctx, db)
		},
	)
}

func (r *authEventRepository) ListSignInsForUser(
	ctx context.Context,
	req iamports.ListUserSignInsRequest,
) ([]*iam.AuthEvent, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*iam.AuthEvent, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxSignInHistory {
			limit = maxSignInHistory
		}

		ae := iamdb.AuthEvents
		entities, err := ae.Where(
			ae.OrganizationID.Eq(req.TenantInfo.OrgID),
			ae.UserID.Eq(req.TenantInfo.UserID),
			ae.Provider.NotIn(iamports.NonSignInAuthEventProviders()...),
		).
			Order(ae.OccurredAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list sign-ins for user", zap.Error(err))
			return nil, err
		}

		return entities, nil
	})
}
