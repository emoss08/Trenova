/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type TableConfigurationRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type tableConfigurationRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewTableConfigurationRepository(
	p TableConfigurationRepositoryParams,
) repositories.TableConfigurationRepository {
	log := p.Logger.With().
		Str("repository", "table_configuration").
		Logger()

	return &tableConfigurationRepository{
		db: p.DB,
		l:  &log,
	}
}

func (tcr *tableConfigurationRepository) filterQuery(
	b *tableconfiguration.ConfigurationQueryBuilder,
	req *repositories.TableConfigurationFilters,
) *tableconfiguration.ConfigurationQueryBuilder {
	b = b.WhereTenant(req.Base.OrgID, req.Base.BuID)

	if req.CreatedBy.IsNotNil() {
		b = b.WhereUserIDEQ(req.CreatedBy)
	}

	if req.Visibility != nil {
		b = b.WhereVisibilityEQ(*req.Visibility)
	}

	if req.IsDefault != nil {
		b = b.WhereIsDefaultEQ(*req.IsDefault)
	}

	if req.IncludeShares {
		b = b.LoadShares()
	}

	if req.IncludeCreator {
		b = b.LoadCreator()
	}

	if req.Search != "" {
		b = b.Where("tc.name ILIKE ? OR tc.description ILIKE ?",
			"%"+req.Search+"%", "%"+req.Search+"%")
	}

	return b
}

func (tcr *tableConfigurationRepository) List(
	ctx context.Context,
	filters *repositories.TableConfigurationFilters,
) (*ports.ListResult[*tableconfiguration.Configuration], error) {
	dba, err := tcr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := tcr.l.With().
		Str("operation", "List").
		Str("resource", filters.Resource).
		Logger()

	b := tableconfiguration.NewConfigurationQuery(dba)
	b = tcr.filterQuery(b, filters)

	entities, total, err := b.AllWithCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan and count table configurations")
		return nil, err
	}

	return &ports.ListResult[*tableconfiguration.Configuration]{
		Items: entities,
		Total: total,
	}, nil
}

func (tcr *tableConfigurationRepository) Copy(
	ctx context.Context,
	req *repositories.CopyTableConfigurationRequest,
) error {
	log := tcr.l.With().
		Str("operation", "Copy").
		Interface("req", req).
		Logger()

	config, err := tcr.GetByID(ctx, req.ConfigID, &repositories.TableConfigurationFilters{
		Base: &ports.FilterQueryOptions{
			BuID:  req.BuID,
			OrgID: req.OrgID,
		},
		IncludeCreator: true,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get configuration")
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

	if _, err = tcr.Create(ctx, newConfig); err != nil {
		log.Error().Err(err).Msg("failed to create new configuration")
		return err
	}

	return nil
}

func (tcr *tableConfigurationRepository) ListPublicConfigurations(
	ctx context.Context,
	opts *repositories.TableConfigurationFilters,
) (*ports.ListResult[*tableconfiguration.Configuration], error) {
	dba, err := tcr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := tcr.l.With().
		Str("operation", "ListPublicConfigurations").
		Str("userID", opts.Base.UserID.String()).
		Logger()

	entities, total, err := tableconfiguration.NewConfigurationQuery(dba).
		WhereGroup(" AND ", func(cqb *tableconfiguration.ConfigurationQueryBuilder) *tableconfiguration.ConfigurationQueryBuilder {
			return cqb.
				WhereVisibilityEQ(tableconfiguration.VisibilityPublic).
				WhereResourceEQ(opts.Resource).
				WhereUserIDNEQ(opts.Base.UserID).
				WhereTenant(opts.Base.OrgID, opts.Base.BuID)
		}).
		LoadCreator().
		AllWithCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get public configurations")
		return nil, eris.Wrap(err, "get public configurations")
	}

	return &ports.ListResult[*tableconfiguration.Configuration]{
		Items: entities,
		Total: total,
	}, nil
}

func (tcr *tableConfigurationRepository) filterUserConfigurations(
	b *tableconfiguration.ConfigurationQueryBuilder,
	opts *repositories.ListUserConfigurationRequest,
) *tableconfiguration.ConfigurationQueryBuilder {
	b = b.WhereGroup(
		" AND ",
		func(cqb *tableconfiguration.ConfigurationQueryBuilder) *tableconfiguration.ConfigurationQueryBuilder {
			return cqb.
				WhereTenant(opts.Filter.TenantOpts.OrgID, opts.Filter.TenantOpts.BuID).
				WhereUserIDEQ(opts.Filter.TenantOpts.UserID).
				WhereResourceEQ(opts.Resource)
		},
	)

	q := b.Query()

	if opts.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			opts.Filter.Query,
			(*tableconfiguration.Configuration)(nil),
		)
	}

	b = b.
		OrderBy(tableconfiguration.ConfigurationQuery.Field.IsDefault, true).
		OrderBy(tableconfiguration.ConfigurationQuery.Field.CreatedAt, false)

	return b.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (tcr *tableConfigurationRepository) ListUserConfigurations(
	ctx context.Context,
	opts *repositories.ListUserConfigurationRequest,
) (*ports.ListResult[*tableconfiguration.Configuration], error) {
	dba, err := tcr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := tcr.l.With().
		Str("operation", "ListUserConfigurations").
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	b := tableconfiguration.NewConfigurationQuery(dba)
	b = tcr.filterUserConfigurations(b, opts)

	entities, total, err := b.AllWithCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user configurations")
		return nil, err
	}

	return &ports.ListResult[*tableconfiguration.Configuration]{
		Items: entities,
		Total: total,
	}, nil
}

func (tcr *tableConfigurationRepository) GetByID(
	ctx context.Context,
	id pulid.ID,
	opts *repositories.TableConfigurationFilters,
) (*tableconfiguration.Configuration, error) {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "GetByID").
		Str("id", id.String()).
		Logger()

	b := tableconfiguration.NewConfigurationQuery(dba)
	b = b.WhereGroup(
		" AND ",
		func(cqb *tableconfiguration.ConfigurationQueryBuilder) *tableconfiguration.ConfigurationQueryBuilder {
			return cqb.
				WhereIDEQ(id).
				WhereTenant(opts.Base.OrgID, opts.Base.BuID)
		},
	)

	if opts.IncludeShares {
		b = b.LoadShares()
	}

	if opts.IncludeCreator {
		b = b.LoadCreator()
	}

	entity, err := b.First(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Configuration not found")
		}

		log.Error().Err(err).Msg("failed to get configuration")
		return nil, err
	}

	return entity, nil
}

func (tcr *tableConfigurationRepository) Create(
	ctx context.Context,
	config *tableconfiguration.Configuration,
) (*tableconfiguration.Configuration, error) {
	dba, err := tcr.db.ReadDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "Create").
		Str("orgID", config.OrganizationID.String()).
		Str("buID", config.BusinessUnitID.String()).
		Logger()

	//nolint:nestif // This is a nested if statement that is not nested in a larger if statement.
	if config.IsDefault {
		existingDefault := new(tableconfiguration.Configuration)
		qErr := tableconfiguration.NewConfigurationQuery(dba).
			WhereGroup("AND ", func(cqb *tableconfiguration.ConfigurationQueryBuilder) *tableconfiguration.ConfigurationQueryBuilder {
				return cqb.
					WhereResourceEQ(config.Resource).
					WhereTenant(config.OrganizationID, config.BusinessUnitID).
					WhereUserIDEQ(config.UserID).WhereIsDefaultEQ(true)
			}).
			Query().
			For("UPDATE").
			Scan(ctx)
		if qErr != nil {
			if eris.Is(qErr, sql.ErrNoRows) {
				log.Debug().Msg("no existing default configuration found, moving on...")
			} else {
				log.Error().Err(qErr).Msg("failed to get existing default configuration")
				return nil, qErr
			}
		}

		_, err = dba.NewUpdate().Model(existingDefault).
			Set("is_default = ?", false).
			Where("tc.id = ?", existingDefault.ID).
			Exec(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to update existing default configuration")
			return nil, err
		}
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(config).
			Returning("*").
			Exec(c); iErr != nil {
			log.Error().Err(iErr).Msg("failed to create configuration")
			return iErr
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create configuration")
		return nil, err
	}

	return config, nil
}

// Update updates the configuration
//
//nolint:gocognit // This is a complex function
func (tcr *tableConfigurationRepository) Update(
	ctx context.Context,
	config *tableconfiguration.Configuration,
) error {
	dba, err := tcr.db.WriteDB(ctx)
	if err != nil {
		return err
	}

	log := tcr.l.With().
		Str("operation", "Update").
		Str("id", config.ID.String()).
		Int64("version", config.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if err := config.DBValidate(c, tx); err != nil {
			return err
		}

		// TODO(wolfred): this is a temporary fix we should clean this up.
		// * if the incoming config is marked as default, then we need to get the existing default and set it to not default
		if config.IsDefault {
			existingDefault, edErr := tcr.GetDefaultOrLatestConfiguration(
				ctx,
				config.Resource,
				&repositories.TableConfigurationFilters{
					Base: &ports.FilterQueryOptions{
						OrgID: config.OrganizationID,
						BuID:  config.BusinessUnitID,
					},
				},
			)
			if edErr != nil {
				// ! if there is no default configuration we don't need to do anything.
				log.Debug().Msg("no existing default configuration found, moving on...")
			}

			if existingDefault != nil {
				_, err = tx.NewUpdate().Model(existingDefault).
					Set("is_default = ?", false).
					Where("tc.id = ?", existingDefault.ID).
					Exec(c)
				if err != nil {
					log.Error().Err(err).Msg("failed to update existing default configuration")
					return err
				}
			}
		}

		ov := config.Version
		config.Version++

		results, rErr := tx.NewUpdate().
			Model(config).
			WherePK().
			Where("version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("config", config).
				Msg("failed to update configuration")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("config", config).
				Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The configuration (%s) has been updated since your last request.",
					config.ID.String(),
				),
			)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (tcr *tableConfigurationRepository) Delete(
	ctx context.Context,
	req repositories.DeleteUserConfigurationRequest,
) error {
	dba, err := tcr.db.WriteDB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "Delete").
		Str("configID", req.ConfigID.String()).
		Str("userID", req.UserID.String()).
		Logger()

	result, err := dba.NewDelete().
		Model((*tableconfiguration.Configuration)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("tc.id = ?", req.ConfigID).
				Where("tc.organization_id = ?", req.OrgID).
				Where("tc.business_unit_id = ?", req.BuID).
				Where("tc.user_id = ?", req.UserID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete configuration")
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return err
	}

	if rows == 0 {
		return errors.NewNotFoundError("Configuration not found")
	}

	return nil
}

func (tcr *tableConfigurationRepository) GetUserConfigurations(
	ctx context.Context,
	resource string,
	opts *repositories.TableConfigurationFilters,
) ([]*tableconfiguration.Configuration, error) {
	dba, err := tcr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := tcr.l.With().
		Str("operation", "GetUserConfigurations").
		Str("userID", opts.UserID.String()).
		Str("resource", resource).
		Logger()

	entities, err := tableconfiguration.NewConfigurationQuery(dba).
		WhereGroup(" AND ", func(cqb *tableconfiguration.ConfigurationQueryBuilder) *tableconfiguration.ConfigurationQueryBuilder {
			return cqb.
				WhereUserIDEQ(opts.UserID).
				WhereTenant(opts.Base.OrgID, opts.Base.BuID)
		}).
		OrderBy(tableconfiguration.ConfigurationQuery.Field.CreatedAt, true).
		All(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user configurations")
		return nil, err
	}

	return entities, nil
}

// GetDefaultOrLatestConfiguration returns the default configuration or the latest if no default exists
func (tcr *tableConfigurationRepository) GetDefaultOrLatestConfiguration(
	ctx context.Context,
	resource string,
	opts *repositories.TableConfigurationFilters,
) (*tableconfiguration.Configuration, error) {
	dba, err := tcr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := tcr.l.With().
		Str("operation", "GetDefaultOrLatestConfiguration").
		Str("resource", resource).
		Logger()

	config := new(tableconfiguration.Configuration)
	q := tableconfiguration.NewConfigurationQuery(dba).
		Model(config).
		WhereGroup(" AND ", func(cqb *tableconfiguration.ConfigurationQueryBuilder) *tableconfiguration.ConfigurationQueryBuilder {
			return cqb.WhereResourceEQ(resource).
				WhereUserIDEQ(opts.Base.UserID).
				WhereTenant(opts.Base.OrgID, opts.Base.BuID)
		}).
		Query()

	err = q.Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Info().Msg("no default configuration found, getting latest")
		} else {
			log.Error().Err(err).Msg("failed to get default or latest configuration")
			return nil, err
		}
	}

	if config.IsDefault {
		return config, nil
	}

	// * Query for the default configuration
	q = q.Where("tc.is_default = ?", true)

	err = q.Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Info().Msg("no default configuration found, getting latest")
		} else {
			log.Error().Err(err).Msg("failed to get default configuration")
			return nil, err
		}
	}

	return config, nil
}

func (tcr *tableConfigurationRepository) ShareConfiguration(
	ctx context.Context,
	share *tableconfiguration.ConfigurationShare,
) error {
	dba, err := tcr.db.WriteDB(ctx)
	if err != nil {
		return err
	}

	log := tcr.l.With().
		Str("operation", "ShareConfiguration").
		Str("configID", share.ConfigurationID.String()).
		Str("sharedWithID", share.SharedWithID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		config := new(tableconfiguration.Configuration)
		err = tx.NewSelect().
			Model(config).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where("tc.id = ?", share.ConfigurationID).
					Where("tc.visibility = ?", tableconfiguration.VisibilityShared).
					Where("tc.organization_id = ?", share.OrganizationID).
					Where("tc.business_unit_id = ?", share.BusinessUnitID)
			}).
			Scan(c)
		if err != nil {
			if eris.Is(err, sql.ErrNoRows) {
				return errors.NewNotFoundError("Configuration not found or not shareable")
			}
			return err
		}

		_, err = tx.NewInsert().
			Model(share).
			Exec(c)
		if err != nil {
			log.Error().Err(err).Msg("failed to create share")
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (tcr *tableConfigurationRepository) RemoveShare(
	ctx context.Context,
	configID, sharedWithID pulid.ID,
) error {
	dba, err := tcr.db.WriteDB(ctx)
	if err != nil {
		return err
	}

	log := tcr.l.With().
		Str("operation", "RemoveShare").
		Str("configID", configID.String()).
		Str("sharedWithID", sharedWithID.String()).
		Logger()

	result, err := dba.NewDelete().
		Model((*tableconfiguration.ConfigurationShare)(nil)).
		WhereGroup(" AND ", func(sq *bun.DeleteQuery) *bun.DeleteQuery {
			return sq.Where("configuration_id = ?", configID).
				Where("shared_with_id = ?", sharedWithID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to remove share")
		return eris.Wrap(err, "remove share")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return eris.Wrap(err, "get rows affected")
	}

	if rows == 0 {
		return errors.NewNotFoundError("Share not found")
	}

	return nil
}
