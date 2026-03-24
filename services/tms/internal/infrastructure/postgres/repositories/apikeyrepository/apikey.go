package apikeyrepository

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

func New(p Params) repositories.APIKeyRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.api-key-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListAPIKeysRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(q, "ak", req.Filter, (*apikey.Key)(nil))

	q = q.Relation("Permissions")

	return q.Limit(req.Filter.Pagination.Limit).Offset(req.Filter.Pagination.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAPIKeysRequest,
) (*pagination.ListResult[*apikey.Key], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*apikey.Key, 0, req.Filter.Pagination.Limit)
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count api keys", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*apikey.Key]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*apikey.Key, error) {
	key := new(apikey.Key)
	err := r.db.DB().
		NewSelect().
		Model(key).
		Relation("Permissions").
		Where("ak.id = ?", id).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			query := sq.Where("ak.organization_id = ?", tenantInfo.OrgID)
			if !tenantInfo.BuID.IsNil() {
				query = query.Where("ak.business_unit_id = ?", tenantInfo.BuID)
			}
			return query
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "API Key")
	}
	return key, nil
}

func (r *repository) GetByPrefix(
	ctx context.Context,
	prefix string,
) (*apikey.Key, error) {
	key := new(apikey.Key)
	err := r.db.DB().
		NewSelect().
		Model(key).
		Relation("Permissions").
		Where("ak.key_prefix = ?", prefix).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "API Key")
	}
	return key, nil
}

func (r *repository) Create(ctx context.Context, key *apikey.Key) error {
	_, err := r.db.DB().NewInsert().Model(key).Exec(ctx)
	return err
}

func (r *repository) CreateWithPermissions(
	ctx context.Context,
	key *apikey.Key,
	permissions []*apikey.Permission,
) error {
	return r.db.DB().RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(key).Exec(c); err != nil {
			return err
		}

		return r.replacePermissions(c, tx, key, permissions)
	})
}

func (r *repository) Update(ctx context.Context, key *apikey.Key) error {
	_, err := r.db.DB().
		NewUpdate().
		Model(key).
		WherePK().
		ExcludeColumn("created_at").
		Exec(ctx)
	return err
}

func (r *repository) UpdateWithPermissions(
	ctx context.Context,
	key *apikey.Key,
	permissions []*apikey.Permission,
) error {
	return r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		if _, err := r.db.DBForContext(c).NewUpdate().Model(key).
			WherePK().
			ExcludeColumn("created_at").
			Exec(c); err != nil {
			return err
		}

		return r.replacePermissions(c, tx, key, permissions)
	})
}

func (r *repository) ReplacePermissions(
	ctx context.Context,
	key *apikey.Key,
	permissions []*apikey.Permission,
) error {
	return r.db.DB().RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		return r.replacePermissions(c, tx, key, permissions)
	})
}

func (r *repository) CountActiveByCreator(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (int, error) {
	return r.db.DB().
		NewSelect().
		Model((*apikey.Key)(nil)).
		Where("ak.organization_id = ?", tenantInfo.OrgID).
		Where("ak.business_unit_id = ?", tenantInfo.BuID).
		Where("ak.created_by_id = ?", userID).
		Where("ak.status = ?", apikey.StatusActive).
		Count(ctx)
}

func (r *repository) UpdateUsage(
	ctx context.Context,
	id pulid.ID,
	metadata repositories.APIKeyUsageMetadata,
) error {
	_, err := r.db.DB().
		NewUpdate().
		Model((*apikey.Key)(nil)).
		Set("last_used_at = ?", metadata.LastUsedAt).
		Set("last_used_ip = ?", metadata.LastUsedIP).
		Set("last_used_user_agent = ?", metadata.LastUsedUserAgent).
		Where("id = ?", id).
		Exec(ctx)
	return err
}

func (r *repository) IncrementDailyUsage(
	ctx context.Context,
	id, orgID, buID pulid.ID,
	date time.Time,
	count int64,
) error {
	usage := &apikey.UsageDaily{
		APIKeyID:       id,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		UsageDate:      date.Truncate(24 * time.Hour),
		RequestCount:   count,
	}

	_, err := r.db.DB().
		NewInsert().
		Model(usage).
		On("CONFLICT (api_key_id, usage_date) DO UPDATE").
		Set("request_count = akud.request_count + ?", count).
		Exec(ctx)
	return err
}

func (r *repository) replacePermissions(
	ctx context.Context,
	tx bun.Tx,
	key *apikey.Key,
	permissions []*apikey.Permission,
) error {
	if _, err := tx.NewDelete().
		Model((*apikey.Permission)(nil)).
		Where("api_key_id = ?", key.ID).
		Exec(ctx); err != nil {
		return err
	}

	if len(permissions) == 0 {
		return nil
	}

	for _, permission := range permissions {
		permission.APIKeyID = key.ID
		permission.BusinessUnitID = key.BusinessUnitID
		permission.OrganizationID = key.OrganizationID
	}

	_, err := tx.NewInsert().Model(&permissions).Exec(ctx)
	return err
}
