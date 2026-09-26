//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package accountingsyncrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const connectionEntity = "Accounting connection"

type ConnectionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type connectionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewConnectionRepository(p ConnectionParams) repositories.AccountingConnectionRepository {
	return &connectionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.accounting-connection-repository"),
	}
}

func tokenColumns() []string {
	cols := buncolgen.AccountingConnectionColumns
	return []string{cols.AccessTokenCiphertext.String(), cols.RefreshTokenCiphertext.String()}
}

func refreshColumns() []string {
	cols := buncolgen.AccountingConnectionColumns
	return []string{
		cols.ReferenceRefreshStartedAt.String(),
		cols.ReferenceRefreshedAt.String(),
		cols.ReferenceRefreshError.String(),
	}
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
	req repositories.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	entity := new(accountingsync.AccountingConnection)
	cols := buncolgen.AccountingConnectionColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		ExcludeColumn(tokenColumns()...).
		Where(cols.IntegrationType.Eq(), req.IntegrationType).
		Apply(buncolgen.AccountingConnectionApplyTenant(req.TenantInfo)).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, connectionEntity)
	}

	return entity, nil
}

func (r *connectionRepository) GetByID(
	ctx context.Context,
	req repositories.GetAccountingConnectionByIDRequest,
) (*accountingsync.AccountingConnection, error) {
	entity := new(accountingsync.AccountingConnection)
	cols := buncolgen.AccountingConnectionColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		ExcludeColumn(tokenColumns()...).
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.AccountingConnectionApplyTenant(req.TenantInfo)).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, connectionEntity)
	}

	return entity, nil
}

func (r *connectionRepository) ListByTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*accountingsync.AccountingConnection, error) {
	entities := make([]*accountingsync.AccountingConnection, 0, 1)
	cols := buncolgen.AccountingConnectionColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ExcludeColumn(tokenColumns()...).
		Apply(buncolgen.AccountingConnectionApplyTenant(tenantInfo)).
		Order(cols.IntegrationType.OrderAsc()).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *connectionRepository) ListHoldingRealm(
	ctx context.Context,
	req repositories.ListAccountingConnectionsByRealmRequest,
) ([]*accountingsync.AccountingConnection, error) {
	if len(req.RealmIDs) == 0 {
		return []*accountingsync.AccountingConnection{}, nil
	}

	entities := make([]*accountingsync.AccountingConnection, 0, len(req.RealmIDs))
	cols := buncolgen.AccountingConnectionColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ExcludeColumn(tokenColumns()...).
		Where(cols.IntegrationType.Eq(), req.IntegrationType).
		Where(cols.ExternalRealmID.In(), bun.List(req.RealmIDs)).
		Where(cols.Status.NotEq(), accountingsync.ConnectionStatusDisconnected).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *connectionRepository) ListDueForHealthCheck(
	ctx context.Context,
	req repositories.ListDueAccountingConnectionsRequest,
) ([]*accountingsync.AccountingConnection, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	entities := make([]*accountingsync.AccountingConnection, 0, limit)
	cols := buncolgen.AccountingConnectionColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ExcludeColumn(tokenColumns()...).
		Where(cols.Status.In(), bun.List(activeStatuses())).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where(cols.LastCheckedAt.IsNull()).
				WhereOr(cols.LastCheckedAt.Lt(), req.CheckedBefore)
		}).
		OrderExpr(cols.LastCheckedAt.Expr("{} ASC NULLS FIRST")).
		Order(cols.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *connectionRepository) LockByTypeWithTokens(
	ctx context.Context,
	req repositories.GetAccountingConnectionRequest,
) (*accountingsync.AccountingConnection, error) {
	entity := new(accountingsync.AccountingConnection)
	cols := buncolgen.AccountingConnectionColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.IntegrationType.Eq(), req.IntegrationType).
		Apply(buncolgen.AccountingConnectionApplyTenant(req.TenantInfo)).
		For("UPDATE").
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, connectionEntity)
	}

	return entity, nil
}

func (r *connectionRepository) LockWithTokens(
	ctx context.Context,
	req repositories.GetAccountingConnectionByIDRequest,
) (*accountingsync.AccountingConnection, error) {
	entity := new(accountingsync.AccountingConnection)
	cols := buncolgen.AccountingConnectionColumns

	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		Where(cols.ID.Eq(), req.ID).
		Apply(buncolgen.AccountingConnectionApplyTenant(req.TenantInfo)).
		For("UPDATE").
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, connectionEntity)
	}

	return entity, nil
}

func (r *connectionRepository) Create(
	ctx context.Context,
	entity *accountingsync.AccountingConnection,
) (*accountingsync.AccountingConnection, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *connectionRepository) Update(
	ctx context.Context,
	entity *accountingsync.AccountingConnection,
) (*accountingsync.AccountingConnection, error) {
	cols := buncolgen.AccountingConnectionColumns
	ov := entity.Version
	entity.Version++

	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		ExcludeColumn(append(tokenColumns(), refreshColumns()...)...).
		WherePK().
		Where(cols.Version.Eq(), ov).
		Exec(ctx)
	if err != nil {
		entity.Version = ov
		return nil, err
	}
	if err = dberror.CheckRowsAffected(results, connectionEntity, entity.ID.String()); err != nil {
		entity.Version = ov
		return nil, err
	}

	return entity, nil
}

func (r *connectionRepository) StoreTokens(
	ctx context.Context,
	req repositories.StoreAccountingTokensRequest,
) error {
	cols := buncolgen.AccountingConnectionColumns

	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*accountingsync.AccountingConnection)(nil)).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AccountingConnectionScopeTenantUpdate(q, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Set(cols.AccessTokenExpiresAt.Set(), req.AccessTokenExpiresAt).
		Set(cols.RefreshTokenExpiresAt.Set(), req.RefreshTokenExpiresAt).
		Set(cols.LastRefreshedAt.Set(), req.RefreshedAt).
		Set(cols.UpdatedAt.Set(), req.At)

	query = setOrNull(query, cols.AccessTokenCiphertext, req.AccessTokenCiphertext)
	query = setOrNull(query, cols.RefreshTokenCiphertext, req.RefreshTokenCiphertext)

	results, err := query.Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckRowsAffected(results, connectionEntity, req.ID.String())
}

func (r *connectionRepository) MarkWebhookReceived(
	ctx context.Context,
	req repositories.MarkAccountingWebhookRequest,
) (int64, error) {
	if len(req.RealmIDs) == 0 {
		return 0, nil
	}

	cols := buncolgen.AccountingConnectionColumns
	results, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*accountingsync.AccountingConnection)(nil)).
		Set(cols.LastWebhookAt.Set(), req.ReceivedAt).
		Where(cols.IntegrationType.Eq(), req.IntegrationType).
		Where(cols.ExternalRealmID.In(), bun.List(req.RealmIDs)).
		Where(cols.Status.NotEq(), accountingsync.ConnectionStatusDisconnected).
		Exec(ctx)
	if err != nil {
		return 0, err
	}

	return results.RowsAffected()
}

func (r *connectionRepository) MarkReferenceRefresh(
	ctx context.Context,
	req repositories.MarkAccountingReferenceRefreshRequest,
) error {
	if req.StartedAt == nil && req.RefreshedAt == nil && req.Error == "" {
		return nil
	}

	cols := buncolgen.AccountingConnectionColumns
	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*accountingsync.AccountingConnection)(nil)).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AccountingConnectionScopeTenantUpdate(q, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		})
	switch {
	case req.StartedAt != nil:
		query = query.
			Set(cols.ReferenceRefreshStartedAt.Set(), *req.StartedAt).
			Set(cols.ReferenceRefreshError.SetNull())
	default:
		query = setOrNull(
			query.Set(cols.ReferenceRefreshStartedAt.SetNull()),
			cols.ReferenceRefreshError,
			req.Error,
		)
		if req.RefreshedAt != nil {
			query = query.Set(cols.ReferenceRefreshedAt.Set(), *req.RefreshedAt)
		}
	}

	results, err := query.Exec(ctx)
	if err != nil {
		return err
	}

	return dberror.CheckRowsAffected(results, connectionEntity, req.ID.String())
}

func (r *connectionRepository) SaveChangeFeed(
	ctx context.Context,
	req *repositories.SaveAccountingChangeFeedRequest,
) (bool, error) {
	cols := buncolgen.AccountingConnectionColumns
	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*accountingsync.AccountingConnection)(nil)).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AccountingConnectionScopeTenantUpdate(q, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		})
	if req.OnlyIfEmpty {
		query = query.Where(cols.ChangeCursor.IsNull())
	}
	if req.Cursor != "" {
		query = query.Set(cols.ChangeCursor.Set(), req.Cursor)
	}
	if req.ReadAt != nil {
		query = query.Set(cols.ChangesReadAt.Set(), *req.ReadAt)
	}
	query = setOrNull(query, cols.ChangesErrorCategory, string(req.ErrorCategory))
	query = setOrNull(query, cols.ChangesErrorMessage, req.ErrorMessage)

	results, err := query.Exec(ctx)
	if err != nil {
		return false, err
	}
	affected, err := results.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (r *connectionRepository) SaveDriftCheck(
	ctx context.Context,
	req *repositories.SaveAccountingDriftCheckRequest,
) error {
	cols := buncolgen.AccountingConnectionColumns
	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*accountingsync.AccountingConnection)(nil)).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.AccountingConnectionScopeTenantUpdate(q, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		})
	if req.CheckedAt != nil {
		query = query.Set(cols.DriftCheckedAt.Set(), *req.CheckedAt)
	}
	query = setOrNull(query, cols.DriftErrorCategory, string(req.ErrorCategory))
	query = setOrNull(query, cols.DriftErrorMessage, req.ErrorMessage)
	if _, err := query.Exec(ctx); err != nil {
		return fmt.Errorf("save the drift check: %w", err)
	}
	return nil
}

func (r *connectionRepository) ListActive(
	ctx context.Context,
	req repositories.ListActiveAccountingConnectionsRequest,
) ([]*accountingsync.AccountingConnection, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	entities := make([]*accountingsync.AccountingConnection, 0, limit)
	cols := buncolgen.AccountingConnectionColumns

	query := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		ExcludeColumn(tokenColumns()...).
		Where(cols.Status.In(), bun.List(activeStatuses())).
		Order(cols.ID.OrderAsc()).
		Limit(limit)
	if !req.AfterID.IsNil() {
		query = query.Where(cols.ID.Gt(), req.AfterID)
	}
	if err := query.Scan(ctx); err != nil {
		return nil, err
	}

	return entities, nil
}

func setOrNull(q *bun.UpdateQuery, col buncolgen.Column, value string) *bun.UpdateQuery {
	if value == "" {
		return q.Set(col.SetNull())
	}
	return q.Set(col.Set(), value)
}
