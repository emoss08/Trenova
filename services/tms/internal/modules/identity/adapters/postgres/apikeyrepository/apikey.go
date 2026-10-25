package apikeyrepository

import (
	"context"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey"
	"github.com/emoss08/trenova/internal/modules/identity/domain/apikey/apikeydb"
	"github.com/emoss08/trenova/internal/modules/identity/ports/apikeyports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const resolvePrefixScopeReason = "resolve a bearer API key by prefix before its tenant is known"

var apiKeyErrors = dberror.Mapping{Entity: "API Key"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) apikeyports.APIKeyRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.api-key-repository"),
	}
}

func withPermissions(q *keel.Query[apikey.Key]) *keel.Query[apikey.Key] {
	return q.Load(apikeydb.Keys.Permissions())
}

func (r *repository) List(
	ctx context.Context,
	req *apikeyports.ListAPIKeysRequest,
) (*pagination.ListResult[*apikey.Key], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*apikey.Key], error) {
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				apikeydb.Keys.Query(),
				&listquery.Request[apikey.Key]{Filter: req.Filter, Shape: withPermissions},
			)
			if err != nil {
				r.l.Error("failed to list api keys", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *apikeyports.ListAPIKeyConnectionRequest,
) (*pagination.CursorListResult[*apikey.Key], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*apikey.Key], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				apikeydb.Keys.Query(),
				&listquery.Request[apikey.Key]{
					Filter: req.Filter,
					Cursor: req.Cursor,
					Shape:  withPermissions,
				},
			)
			if err != nil {
				r.l.Error("failed to list api keys", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	_ pagination.TenantInfo,
	id pulid.ID,
) (*apikey.Key, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*apikey.Key, error) {
		k := apikeydb.Keys
		key, err := withPermissions(k.Where(k.ID.Eq(id))).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, apiKeyErrors.Map(err, id.String())
		}
		return key, nil
	})
}

func (r *repository) GetByPrefix(
	ctx context.Context,
	prefix string,
) (*apikey.Key, error) {
	ctx = dbscope.WithSystem(ctx, resolvePrefixScopeReason)
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*apikey.Key, error) {
		k := apikeydb.Keys
		key, err := withPermissions(k.AllTenants().Where(k.KeyPrefix.Eq(prefix))).
			One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, apiKeyErrors.Map(err, prefix)
		}
		return key, nil
	})
}

func (r *repository) Create(ctx context.Context, key *apikey.Key) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return apiKeyErrors.Map(apikeydb.Keys.Create(ctx, r.db.Keel(ctx), key), key.ID.String())
	})
}

func (r *repository) CreateWithPermissions(
	ctx context.Context,
	key *apikey.Key,
	permissions []*apikey.Permission,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := apikeydb.Keys.Create(ctx, r.db.Keel(ctx), key); err != nil {
			return apiKeyErrors.Map(err, key.ID.String())
		}

		return r.replacePermissions(ctx, key, permissions)
	})
}

func (r *repository) Update(ctx context.Context, key *apikey.Key) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return apiKeyErrors.Map(apikeydb.Keys.Save(ctx, r.db.Keel(ctx), key), key.ID.String())
	})
}

func (r *repository) UpdateWithPermissions(
	ctx context.Context,
	key *apikey.Key,
	permissions []*apikey.Permission,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := apikeydb.Keys.Save(ctx, r.db.Keel(ctx), key); err != nil {
			return apiKeyErrors.Map(err, key.ID.String())
		}

		return r.replacePermissions(ctx, key, permissions)
	})
}

func (r *repository) ReplacePermissions(
	ctx context.Context,
	key *apikey.Key,
	permissions []*apikey.Permission,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return r.replacePermissions(ctx, key, permissions)
	})
}

func (r *repository) CountActiveByCreator(
	ctx context.Context,
	_ pagination.TenantInfo,
	userID pulid.ID,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		k := apikeydb.Keys
		n, err := k.Where(
			k.CreatedByID.Eq(userID),
			k.Status.Eq(apikey.StatusActive),
		).Count(ctx, r.db.Keel(ctx))
		return int(n), err
	})
}

func (r *repository) UpdateUsage(
	ctx context.Context,
	id pulid.ID,
	metadata apikeyports.APIKeyUsageMetadata,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		k := apikeydb.Keys
		_, err := k.Where(k.ID.Eq(id)).Update(
			ctx,
			r.db.Keel(ctx),
			k.LastUsedAt.To(metadata.LastUsedAt),
			k.LastUsedIP.To(metadata.LastUsedIP),
			k.LastUsedUserAgent.To(metadata.LastUsedUserAgent),
			k.UpdatedAt.ToExpr(k.UpdatedAt),
		)
		return err
	})
}

func (r *repository) IncrementDailyUsage(
	ctx context.Context,
	id, orgID, buID pulid.ID,
	date time.Time,
	count int64,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		u := apikeydb.UsageDailies
		_, err := u.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*apikey.UsageDaily{{
				APIKeyID:       id,
				OrganizationID: orgID,
				BusinessUnitID: buID,
				UsageDate:      date.Truncate(24 * time.Hour),
				RequestCount:   count,
			}},
			keel.OnConflict(u.APIKeyID, u.UsageDate).Set(u.RequestCount.Add(count)),
		)
		return err
	})
}

func (r *repository) replacePermissions(
	ctx context.Context,
	key *apikey.Key,
	permissions []*apikey.Permission,
) error {
	p := apikeydb.Permissions
	db := r.db.Keel(ctx)
	if _, err := p.Where(p.APIKeyID.Eq(key.ID)).Delete(ctx, db); err != nil {
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

	return p.CreateMany(ctx, db, permissions)
}
