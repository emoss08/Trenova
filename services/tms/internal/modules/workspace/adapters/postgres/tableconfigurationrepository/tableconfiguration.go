package tableconfigurationrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/tableconfiguration/tableconfigurationdb"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/tableconfigurationports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const entityName = "TableConfiguration"

var configurationErrors = dberror.Mapping{Entity: entityName}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) tableconfigurationports.TableConfigurationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.table-configuration-repository"),
	}
}

func (r *repository) visibleTo(
	tenantInfo pagination.TenantInfo,
	resource string,
	visibility string,
) []keel.Pred {
	tc := tableconfigurationdb.TableConfigurations
	preds := []keel.Pred{
		keel.Or(
			keel.And(
				tc.UserID.Eq(tenantInfo.UserID),
				tc.Visibility.Eq(tableconfiguration.VisibilityPrivate),
			),
			tc.Visibility.Eq(tableconfiguration.VisibilityPublic),
		),
		keel.When(resource != "", tc.Resource.Eq(resource)),
	}

	if visibility != "" {
		v, err := tableconfiguration.VisibilityFromString(visibility)
		if err != nil {
			r.l.Error("failed to parse visibility", zap.Error(err))
			return preds
		}
		preds = append(preds, tc.Visibility.Eq(v))
	}

	return preds
}

func (r *repository) List(
	ctx context.Context,
	req *tableconfigurationports.ListTableConfigurationsRequest,
) (*pagination.ListResult[*tableconfiguration.TableConfiguration], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*tableconfiguration.TableConfiguration], error) {
			tc := tableconfigurationdb.TableConfigurations
			result, err := listquery.List(
				ctx,
				r.db.Keel(ctx),
				tc.Where(r.visibleTo(req.Filter.TenantInfo, req.Resource, req.Visibility)...),
				&listquery.Request[tableconfiguration.TableConfiguration]{
					Filter: req.Filter,
					Then:   []keel.Order{tc.IsDefault.Desc(), tc.CreatedAt.Desc()},
				},
			)
			if err != nil {
				r.l.Error("failed to list table configurations", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *tableconfigurationports.ListTableConfigurationConnectionRequest,
) (*pagination.CursorListResult[*tableconfiguration.TableConfiguration], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*tableconfiguration.TableConfiguration], error) {
			tc := tableconfigurationdb.TableConfigurations
			q := tc.Where(r.visibleTo(req.Filter.TenantInfo, req.Resource, req.Visibility)...)
			if req.LimitToResources {
				readable := tc.Resource.In(req.ReadableResources...)
				if len(req.ReadableResources) == 0 {
					readable = keel.Not(keel.True())
				}
				q = q.Where(readable)
			}

			user := tc.User()
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				q,
				&listquery.Request[tableconfiguration.TableConfiguration]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.TableConfigurationColumns,
					Shape: func(
						q *keel.Query[tableconfiguration.TableConfiguration],
					) *keel.Query[tableconfiguration.TableConfiguration] {
						return q.Load(
							user.Pick(user.ID, user.Name, user.EmailAddress, user.ProfilePicURL),
						)
					},
				},
			)
			if err != nil {
				r.l.Error("failed to list table configuration connection", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *tableconfiguration.TableConfiguration,
) (*tableconfiguration.TableConfiguration, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*tableconfiguration.TableConfiguration, error) {
			err := tableconfigurationdb.TableConfigurations.Create(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to create table configuration",
					zap.String("name", entity.Name),
					zap.Error(err),
				)
				return nil, err
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *tableconfiguration.TableConfiguration,
) (*tableconfiguration.TableConfiguration, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*tableconfiguration.TableConfiguration, error) {
			err := tableconfigurationdb.TableConfigurations.Save(ctx, r.db.Keel(ctx), entity)
			if err != nil {
				r.l.Error(
					"failed to update table configuration",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, configurationErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req tableconfigurationports.GetTableConfigurationByIDRequest,
) (*tableconfiguration.TableConfiguration, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*tableconfiguration.TableConfiguration, error) {
			entity, err := tableconfigurationdb.TableConfigurations.Get(
				ctx,
				r.db.Keel(ctx),
				req.ConfigurationID,
			)
			if err != nil {
				r.l.Error(
					"failed to get table configuration",
					zap.String("configurationID", req.ConfigurationID.String()),
					zap.Error(err),
				)
				return nil, configurationErrors.Map(err, req.ConfigurationID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Delete(
	ctx context.Context,
	configurationID pulid.ID,
	_ pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		tc := tableconfigurationdb.TableConfigurations
		rows, err := tc.Where(tc.ID.Eq(configurationID)).Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error(
				"failed to delete table configuration",
				zap.String("configurationID", configurationID.String()),
				zap.Error(err),
			)
			return err
		}

		return dberror.CheckAffected(rows, entityName, configurationID.String())
	})
}

func (r *repository) GetDefaultForResource(
	ctx context.Context,
	req tableconfigurationports.GetDefaultTableConfigurationRequest,
) (*tableconfiguration.TableConfiguration, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*tableconfiguration.TableConfiguration, error) {
			tc := tableconfigurationdb.TableConfigurations
			db := r.db.Keel(ctx)
			entity, err := tc.Where(
				tc.UserID.Eq(req.TenantInfo.UserID),
				tc.Resource.Eq(req.Resource),
				tc.IsDefault.IsTrue(),
			).First(ctx, db)
			if err == nil {
				return entity, nil
			}

			orgDefault, err := tc.Where(
				tc.Resource.Eq(req.Resource),
				tc.IsOrgDefault.IsTrue(),
				tc.Visibility.Eq(tableconfiguration.VisibilityPublic),
			).First(ctx, db)
			if err != nil {
				r.l.Debug("no default table configuration found", zap.Error(err))
				return nil, configurationErrors.Map(err, req.Resource)
			}

			return orgDefault, nil
		},
	)
}

func (r *repository) ClearOrgDefaultForResource(
	ctx context.Context,
	resource string,
	_ pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		tc := tableconfigurationdb.TableConfigurations
		_, err := tc.Where(tc.Resource.Eq(resource), tc.IsOrgDefault.IsTrue()).Update(
			ctx,
			r.db.Keel(ctx),
			tc.IsOrgDefault.To(false),
			tc.Version.ToExpr(tc.Version),
		)
		if err != nil {
			r.l.Error(
				"failed to clear org default table configuration",
				zap.String("resource", resource),
				zap.Error(err),
			)
			return err
		}

		return nil
	})
}

func (r *repository) ClearDefaultForResource(
	ctx context.Context,
	userID pulid.ID,
	resource string,
	_ pagination.TenantInfo,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		tc := tableconfigurationdb.TableConfigurations
		_, err := tc.Where(
			tc.UserID.Eq(userID),
			tc.Resource.Eq(resource),
			tc.IsDefault.IsTrue(),
		).Update(
			ctx,
			r.db.Keel(ctx),
			tc.IsDefault.To(false),
			tc.Version.ToExpr(tc.Version),
		)
		if err != nil {
			r.l.Error(
				"failed to clear default table configuration",
				zap.String("userID", userID.String()),
				zap.String("resource", resource),
				zap.Error(err),
			)
			return err
		}

		return nil
	})
}
