//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package accountingsyncrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/listquery"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync/accountingsyncdb"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	connectionEntity      = "Accounting connection"
	defaultConnectionPage = 100
)

var connectionErrors = dberror.Mapping{Entity: connectionEntity}

type ConnectionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type connectionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewConnectionRepository(
	p ConnectionParams,
) accountingsyncports.AccountingConnectionRepository {
	return &connectionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-connection-repository"),
	}
}

func withoutTokens() *keel.Query[accountingsync.AccountingConnection] {
	ac := accountingsyncdb.AccountingConnections
	return ac.Omit(ac.AccessTokenCiphertext, ac.RefreshTokenCiphertext, ac.ClientKeyCiphertext)
}

func activeStatuses() []accountingsync.ConnectionStatus {
	return []accountingsync.ConnectionStatus{
		accountingsync.ConnectionStatusConnected,
		accountingsync.ConnectionStatusDegraded,
		accountingsync.ConnectionStatusFailing,
	}
}

func (r *connectionRepository) GetByType(
	ctx context.Context,
	req accountingsyncports.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingConnection, error) {
			ac := accountingsyncdb.AccountingConnections
			entity, err := withoutTokens().
				Where(ac.IntegrationType.Eq(req.IntegrationType)).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, connectionErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *connectionRepository) GetByID(
	ctx context.Context,
	req accountingsyncports.GetAccountingConnectionByIDRequest,
) (*accountingsync.AccountingConnection, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingConnection, error) {
			ac := accountingsyncdb.AccountingConnections
			entity, err := withoutTokens().Where(ac.ID.Eq(req.ID)).One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, connectionErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *connectionRepository) ListByTenant(
	ctx context.Context,
	_ pagination.TenantInfo,
) ([]*accountingsync.AccountingConnection, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingConnection, error) {
			ac := accountingsyncdb.AccountingConnections
			return withoutTokens().Order(ac.IntegrationType.Asc()).All(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *connectionRepository) ListHoldingRealm(
	ctx context.Context,
	req accountingsyncports.ListAccountingConnectionsByRealmRequest,
) ([]*accountingsync.AccountingConnection, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"find every connection holding an accounting company, which spans tenants by design",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingConnection, error) {
			if len(req.RealmIDs) == 0 {
				return []*accountingsync.AccountingConnection{}, nil
			}

			ac := accountingsyncdb.AccountingConnections
			return withoutTokens().
				AllTenants().
				Where(
					ac.IntegrationType.Eq(req.IntegrationType),
					ac.ExternalRealmID.In(req.RealmIDs...),
					ac.Status.Ne(accountingsync.ConnectionStatusDisconnected),
				).
				All(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *connectionRepository) ListDueForHealthCheck(
	ctx context.Context,
	req accountingsyncports.ListDueAccountingConnectionsRequest,
) ([]*accountingsync.AccountingConnection, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"list accounting connections due a health check across every organization",
	)
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingConnection, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultConnectionPage
			}

			ac := accountingsyncdb.AccountingConnections
			return withoutTokens().
				AllTenants().
				Where(
					ac.Status.In(activeStatuses()...),
					keel.Or(ac.LastCheckedAt.IsNull(), ac.LastCheckedAt.Lt(req.CheckedBefore)),
				).
				Order(ac.LastCheckedAt.Asc().NullsFirst(), ac.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
		},
	)
}

func (r *connectionRepository) LockByTypeWithTokens(
	ctx context.Context,
	req accountingsyncports.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingConnection, error) {
			ac := accountingsyncdb.AccountingConnections
			entity, err := ac.Where(ac.IntegrationType.Eq(req.IntegrationType)).
				Lock(keel.ForUpdate).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, connectionErrors.Map(err, "")
			}

			return entity, nil
		},
	)
}

func (r *connectionRepository) LockWithTokens(
	ctx context.Context,
	req accountingsyncports.GetAccountingConnectionByIDRequest,
) (*accountingsync.AccountingConnection, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingConnection, error) {
			ac := accountingsyncdb.AccountingConnections
			entity, err := ac.Where(ac.ID.Eq(req.ID)).
				Lock(keel.ForUpdate).
				One(ctx, r.db.Keel(ctx))
			if err != nil {
				return nil, connectionErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *connectionRepository) Create(
	ctx context.Context,
	entity *accountingsync.AccountingConnection,
) (*accountingsync.AccountingConnection, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingConnection, error) {
			err := accountingsyncdb.AccountingConnections.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				return nil, connectionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *connectionRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingConnection,
) (*accountingsync.AccountingConnection, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*accountingsync.AccountingConnection, error) {
			ac := accountingsyncdb.AccountingConnections
			ov := entity.Version
			err := ac.SaveExcept(
				ctx,
				r.db.Keel(ctx),
				entity,
				ac.AccessTokenCiphertext,
				ac.RefreshTokenCiphertext,
				ac.ClientKeyCiphertext,
				ac.ReferenceRefreshStartedAt,
				ac.ReferenceRefreshedAt,
				ac.ReferenceRefreshError,
			)
			if err != nil {
				entity.Version = ov
				return nil, connectionErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *connectionRepository) StoreTokens(
	ctx context.Context,
	req accountingsyncports.StoreAccountingTokensRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ac := accountingsyncdb.AccountingConnections
		n, err := ac.Where(ac.ID.Eq(req.ID)).Update(
			ctx,
			r.db.Keel(ctx),
			ac.AccessTokenExpiresAt.To(req.AccessTokenExpiresAt),
			ac.RefreshTokenExpiresAt.To(req.RefreshTokenExpiresAt),
			listquery.OptionalInt64(ac.LastRefreshedAt, req.RefreshedAt),
			ac.UpdatedAt.To(req.At),
			stringOrNull(ac.AccessTokenCiphertext, req.AccessTokenCiphertext),
			stringOrNull(ac.RefreshTokenCiphertext, req.RefreshTokenCiphertext),
			stringOrNull(ac.ClientKeyCiphertext, req.ClientKeyCiphertext),
			keepConnectionVersion(),
		)
		if err != nil {
			return err
		}

		return dberror.CheckAffected(n, connectionEntity, req.ID.String())
	})
}

func (r *connectionRepository) MarkWebhookReceived(
	ctx context.Context,
	req accountingsyncports.MarkAccountingWebhookRequest,
) (int64, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"stamp every connection holding the accounting company a webhook named",
	)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int64, error) {
		if len(req.RealmIDs) == 0 {
			return 0, nil
		}

		ac := accountingsyncdb.AccountingConnections
		return ac.AllTenants().
			Where(
				ac.IntegrationType.Eq(req.IntegrationType),
				ac.ExternalRealmID.In(req.RealmIDs...),
				ac.Status.Ne(accountingsync.ConnectionStatusDisconnected),
			).
			Update(
				ctx,
				r.db.Keel(ctx),
				ac.LastWebhookAt.To(req.ReceivedAt),
				keepConnectionVersion(),
			)
	})
}

func (r *connectionRepository) MarkReferenceRefresh(
	ctx context.Context,
	req accountingsyncports.MarkAccountingReferenceRefreshRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if req.StartedAt == nil && req.RefreshedAt == nil && req.Error == "" {
			return nil
		}

		ac := accountingsyncdb.AccountingConnections
		sets := []keel.Assignment{keepConnectionVersion()}
		switch {
		case req.StartedAt != nil:
			sets = append(sets,
				ac.ReferenceRefreshStartedAt.To(*req.StartedAt),
				ac.ReferenceRefreshError.ToNull(),
			)
		default:
			sets = append(sets,
				ac.ReferenceRefreshStartedAt.ToNull(),
				stringOrNull(ac.ReferenceRefreshError, req.Error),
			)
			if req.RefreshedAt != nil {
				sets = append(sets, ac.ReferenceRefreshedAt.To(*req.RefreshedAt))
			}
		}

		n, err := ac.Where(ac.ID.Eq(req.ID)).Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return err
		}

		return dberror.CheckAffected(n, connectionEntity, req.ID.String())
	})
}

func (r *connectionRepository) SaveChangeFeed(
	ctx context.Context,
	req *accountingsyncports.SaveAccountingChangeFeedRequest,
) (bool, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (bool, error) {
		ac := accountingsyncdb.AccountingConnections
		sets := []keel.Assignment{
			keepConnectionVersion(),
			stringOrNull(ac.ChangesErrorCategory.StringCol, string(req.ErrorCategory)),
			stringOrNull(ac.ChangesErrorMessage, req.ErrorMessage),
		}
		if req.Cursor != "" {
			sets = append(sets, ac.ChangeCursor.To(req.Cursor))
		}
		if req.ReadAt != nil {
			sets = append(sets, ac.ChangesReadAt.To(*req.ReadAt))
		}

		n, err := ac.Where(
			ac.ID.Eq(req.ID),
			keel.When(req.OnlyIfEmpty, ac.ChangeCursor.IsNull()),
		).Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return false, err
		}

		return n > 0, nil
	})
}

func (r *connectionRepository) SaveDriftCheck(
	ctx context.Context,
	req *accountingsyncports.SaveAccountingDriftCheckRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		ac := accountingsyncdb.AccountingConnections
		sets := []keel.Assignment{
			keepConnectionVersion(),
			stringOrNull(ac.DriftErrorCategory.StringCol, string(req.ErrorCategory)),
			stringOrNull(ac.DriftErrorMessage, req.ErrorMessage),
		}
		if req.CheckedAt != nil {
			sets = append(sets, ac.DriftCheckedAt.To(*req.CheckedAt))
		}
		if _, err := ac.Where(ac.ID.Eq(req.ID)).Update(ctx, r.db.Keel(ctx), sets...); err != nil {
			return fmt.Errorf("save the drift check: %w", err)
		}
		return nil
	})
}

func (r *connectionRepository) ListActive(
	ctx context.Context,
	req accountingsyncports.ListActiveAccountingConnectionsRequest,
) ([]*accountingsync.AccountingConnection, error) {
	ctx = dbscope.WithSystem(ctx, "list active accounting connections across every organization")
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*accountingsync.AccountingConnection, error) {
			limit := req.Limit
			if limit <= 0 {
				limit = defaultConnectionPage
			}

			ac := accountingsyncdb.AccountingConnections
			return withoutTokens().
				AllTenants().
				Where(
					ac.Status.In(activeStatuses()...),
					keel.When(!req.AfterID.IsNil(), ac.ID.Gt(req.AfterID)),
				).
				Order(ac.ID.Asc()).
				Limit(limit).
				List(ctx, r.db.Keel(ctx))
		},
	)
}

func keepConnectionVersion() keel.Assignment {
	ac := accountingsyncdb.AccountingConnections
	return ac.Version.ToExpr(ac.Version)
}

func stringOrNull(col keel.StringCol, value string) keel.Assignment {
	if value == "" {
		return col.ToNull()
	}
	return col.To(value)
}
