package tableconfigurationrepository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
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

func NewRepository(p Params) repositories.TableConfigurationRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.tableconfiguration-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.TableConfigurationFilters,
) *bun.SelectQuery {
	qb := querybuilder.New(q, "tc")
	qb.ApplyTenantFilters(req.Filter.TenantOpts)
	qb.ApplyTextSearch(req.Search, []string{"name", "description", "resource"})

	q = qb.GetQuery()

	if req.CreatedBy.IsNotNil() {
		q.Where("user_id = ?", req.CreatedBy)
	}

	if req.Visibility != nil {
		q.Where("visibility = ?", *req.Visibility)
	}

	if !req.IsDefault {
		q.Where("is_default = ?", req.IsDefault)
	}

	if req.IncludeShares {
		q.Relation("Shares")
	}

	if req.IncludeCreator {
		q.Relation("Creator")
	}

	q.Order("tc.created_at DESC")

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.TableConfigurationFilters,
) (*pagination.ListResult[*tableconfiguration.Configuration], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*tableconfiguration.Configuration, 0, req.Filter.Limit)

	total, err := db.NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan table configurations", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tableconfiguration.Configuration]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) ListPublicConfigurations(
	ctx context.Context,
	opts *repositories.TableConfigurationFilters,
) (*pagination.ListResult[*tableconfiguration.Configuration], error) {
	log := r.l.With(
		zap.String("operation", "ListPublicConfigurations"),
		zap.String("userID", opts.Filter.TenantOpts.UserID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*tableconfiguration.Configuration, 0, opts.Filter.Limit)
	total, err := db.NewSelect().Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tc.visibility = ?", tableconfiguration.VisibilityPublic).
				Where("tc.resource = ?", opts.Resource).
				Where("tc.user_id != ?", opts.Filter.TenantOpts.UserID).
				Where("tc.organization_id = ?", opts.Filter.TenantOpts.OrgID).
				Where("tc.business_unit_id = ?", opts.Filter.TenantOpts.BuID)
		}).
		Relation("Creator").
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan table configurations", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tableconfiguration.Configuration]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) filterUserConfigurations(
	q *bun.SelectQuery,
	opts *repositories.ListUserConfigurationRequest,
) *bun.SelectQuery {
	qb := querybuilder.New(q, "tc").
		ApplyTenantFilters(opts.Filter.TenantOpts).
		ApplyTextSearch(opts.Filter.Query, []string{"name", "description", "resource"})

	q = qb.GetQuery()

	q = q.Order("tc.is_default DESC", "tc.created_at ASC")

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (r *repository) ListUserConfigurations(
	ctx context.Context,
	opts *repositories.ListUserConfigurationRequest,
) (*pagination.ListResult[*tableconfiguration.Configuration], error) {
	log := r.l.With(
		zap.String("operation", "ListUserConfigurations"),
		zap.String("userID", opts.Filter.TenantOpts.UserID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*tableconfiguration.Configuration, 0, opts.Filter.Limit)
	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterUserConfigurations(sq, opts)
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan table configurations", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*tableconfiguration.Configuration]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetDefaultOrLatest(
	ctx context.Context,
	resource string,
	opts *repositories.TableConfigurationFilters,
) (*tableconfiguration.Configuration, error) {
	log := r.l.With(
		zap.String("operation", "GetDefaultOrLatest"),
		zap.String("resource", resource),
		zap.String("userID", opts.Filter.TenantOpts.UserID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	config := new(tableconfiguration.Configuration)

	// First try to get the default configuration
	err = db.NewSelect().Model(config).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("resource = ?", resource).
				Where("organization_id = ?", opts.Filter.TenantOpts.OrgID).
				Where("business_unit_id = ?", opts.Filter.TenantOpts.BuID).
				Where("user_id = ?", opts.Filter.TenantOpts.UserID).
				Where("is_default = ?", true)
		}).
		Scan(ctx)

	if err == nil {
		// Found default configuration
		log.Debug("found default configuration", zap.String("configID", config.ID.String()))
		return config, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		// Unexpected error
		log.Error("failed to scan default table configuration", zap.Error(err))
		return nil, err
	}

	// No default found, get the latest configuration
	err = db.NewSelect().Model(config).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("resource = ?", resource).
				Where("organization_id = ?", opts.Filter.TenantOpts.OrgID).
				Where("business_unit_id = ?", opts.Filter.TenantOpts.BuID).
				Where("user_id = ?", opts.Filter.TenantOpts.UserID)
		}).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("no table configuration found for user")
			return nil, nil //nolint:nilnil // Return nil to indicate no configuration exists
		}
		log.Error("failed to scan latest table configuration", zap.Error(err))
		return nil, err
	}

	log.Debug("found latest configuration", zap.String("configID", config.ID.String()))
	return config, nil
}

func (r *repository) GetUserConfigurations(
	ctx context.Context,
	resource string,
	opts *repositories.TableConfigurationFilters,
) ([]*tableconfiguration.Configuration, error) {
	log := r.l.With(
		zap.String("operation", "GetUserConfigurations"),
		zap.String("resource", resource),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*tableconfiguration.Configuration, 0, opts.Filter.Limit)

	err = db.NewSelect().Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tc.organization_id = ?", opts.Filter.TenantOpts.OrgID).
				Where("tc.business_unit_id = ?", opts.Filter.TenantOpts.BuID).
				Where("tc.user_id = ?", opts.Filter.TenantOpts.UserID)
		}).
		Order("tc.created_at DESC").
		Scan(ctx)
	if err != nil {
		log.Error("failed to scan table configurations", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	id pulid.ID,
	opts *repositories.TableConfigurationFilters,
) (*tableconfiguration.Configuration, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("configId", id.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(tableconfiguration.Configuration)
	q := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tc.id = ?", id).
				Where("tc.organization_id = ?", opts.Filter.TenantOpts.OrgID).
				Where("tc.business_unit_id = ?", opts.Filter.TenantOpts.BuID)
		})

	if opts.IncludeShares {
		q.Relation("Shares")
	}

	if opts.IncludeCreator {
		q.Relation("Creator")
	}

	if err = q.Scan(ctx); err != nil {
		log.Error("failed to scan table configuration", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Table Configuration")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	config *tableconfiguration.Configuration,
) (*tableconfiguration.Configuration, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("configId", config.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if config.IsDefault {
		if err = r.unsetExistingDefault(ctx, db, log, config); err != nil {
			log.Error("failed to unset existing default", zap.Error(err))
			return nil, err
		}
	}

	return config, nil
}

func (r *repository) unsetExistingDefault(
	ctx context.Context,
	db bun.IDB,
	log *zap.Logger,
	config *tableconfiguration.Configuration,
) error {
	err := db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		existingDefault := new(tableconfiguration.Configuration)
		qErr := tx.NewSelect().Model(existingDefault).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where("tc.is_default = ?", true).
					Where("tc.resource = ?", config.Resource).
					Where("tc.organization_id = ?", config.OrganizationID).
					Where("tc.business_unit_id = ?", config.BusinessUnitID).
					Where("tc.user_id = ?", config.UserID)
			}).
			For("UPDATE").
			Scan(c)
		if qErr != nil {
			if errors.Is(qErr, sql.ErrNoRows) {
				log.Debug("no existing default found, moving on...")
			} else {
				log.Error("failed to scan existing default configuration", zap.Error(qErr))
				return qErr
			}
		}

		_, err := tx.NewUpdate().
			Model(existingDefault).
			Set("is_default = ?", false).
			Where("tc.id = ?", existingDefault.ID).
			Exec(c)
		if err != nil {
			log.Error("failed to update existing default configuration", zap.Error(err))
			return err
		}

		_, err = tx.NewInsert().Model(config).Exec(c)
		if err != nil {
			log.Error("failed to create table configuration", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to run in transaction", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	config *tableconfiguration.Configuration,
) error {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("configId", config.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if config.IsDefault {
			existingDefault, edErr := r.GetDefaultOrLatest(
				c,
				config.Resource,
				&repositories.TableConfigurationFilters{
					Filter: &pagination.QueryOptions{
						TenantOpts: pagination.TenantOptions{
							OrgID:  config.OrganizationID,
							BuID:   config.BusinessUnitID,
							UserID: config.UserID,
						},
					},
				},
			)
			if edErr != nil {
				log.Error("failed to get existing default configuration", zap.Error(edErr))
				return edErr
			}

			isDefault := existingDefault != nil && existingDefault.IsDefault &&
				existingDefault.ID != config.ID
			if isDefault {
				_, err = tx.NewUpdate().
					Model(existingDefault).
					Set("is_default = ?", false).
					Where("tc.id = ?", existingDefault.ID).
					Exec(c)
				if err != nil {
					log.Error("failed to update existing default configuration", zap.Error(err))
					return err
				}
			}
		}

		ov := config.Version
		config.Version++

		result, rErr := tx.NewUpdate().Model(config).
			WherePK().
			Where("version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error("failed to update table configuration", zap.Error(rErr))
			return rErr
		}

		roErr := dberror.CheckRowsAffected(result, "Table Configuration", config.ID.String())
		if roErr != nil {
			return roErr
		}

		return nil
	})
	if err != nil {
		log.Error("failed to run in transaction", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteUserConfigurationRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("configId", req.ConfigID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := db.NewDelete().
		Model((*tableconfiguration.Configuration)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("tc.id = ?", req.ConfigID).
				Where("tc.organization_id = ?", req.OrgID).
				Where("tc.business_unit_id = ?", req.BuID).
				Where("tc.user_id = ?", req.UserID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete table configuration", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "Table Configuration", req.ConfigID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) Copy(
	ctx context.Context,
	req *repositories.CopyTableConfigurationRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Copy"),
		zap.String("configId", req.ConfigID.String()),
	)

	config, err := r.GetByID(ctx, req.ConfigID, &repositories.TableConfigurationFilters{
		Filter: &pagination.QueryOptions{
			TenantOpts: pagination.TenantOptions{
				BuID:  req.BuID,
				OrgID: req.OrgID,
			},
		},
		IncludeCreator: true,
	})
	if err != nil {
		log.Error("failed to get configuration", zap.Error(err))
		return err
	}
	newConfig := config

	// ! we need to remove the id and version from the new config
	newConfig.ID = pulid.Nil
	newConfig.Version = 0

	newConfig.Visibility = tableconfiguration.VisibilityPrivate
	newConfig.Name = fmt.Sprintf("%s (Copy)", config.Name)
	newConfig.IsDefault = false
	newConfig.Description = fmt.Sprintf("Copy of %s by %s", config.Name, config.Creator.Name)
	newConfig.UserID = req.UserID

	if _, err = r.Create(ctx, newConfig); err != nil {
		log.Error("failed to create new configuration", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Share(
	ctx context.Context,
	share *tableconfiguration.ConfigurationShare,
) error {
	log := r.l.With(
		zap.String("operation", "Share"),
		zap.String("configID", share.ConfigurationID.String()),
		zap.String("sharedWithID", share.SharedWithID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	config := new(tableconfiguration.Configuration)
	err = db.NewSelect().
		Model(config).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tc.id = ?", share.ConfigurationID).
				Where("tc.visibility = ?", tableconfiguration.VisibilityShared).
				Where("tc.organization_id = ?", share.OrganizationID).
				Where("tc.business_unit_id = ?", share.BusinessUnitID)
		}).
		Scan(ctx)
	if err != nil {
		return dberror.HandleNotFoundError(err, "Table Configuration")
	}

	_, err = db.NewInsert().
		Model(share).
		Exec(ctx)
	if err != nil {
		log.Error("failed to create share", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) RemoveShare(
	ctx context.Context,
	configID, sharedWithID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "RemoveShare"),
		zap.String("configID", configID.String()),
		zap.String("sharedWithID", sharedWithID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := db.NewDelete().
		Model((*tableconfiguration.ConfigurationShare)(nil)).
		WhereGroup(" AND ", func(sq *bun.DeleteQuery) *bun.DeleteQuery {
			return sq.Where("configuration_id = ?", configID).
				Where("shared_with_id = ?", sharedWithID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to remove share", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "Table Configuration Share", configID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}
