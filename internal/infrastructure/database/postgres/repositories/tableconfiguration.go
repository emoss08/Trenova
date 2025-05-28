package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
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

func NewTableConfigurationRepository(p TableConfigurationRepositoryParams) repositories.TableConfigurationRepository {
	log := p.Logger.With().
		Str("repository", "table_configuration").
		Logger()

	return &tableConfigurationRepository{
		db: p.DB,
		l:  &log,
	}
}

func (tcr *tableConfigurationRepository) filterQuery(q *bun.SelectQuery, opts *repositories.TableConfigurationFilters) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query: q,
		// Filter:     opts.Base,
		TableAlias: "tc",
	})

	if opts.CreatedBy.IsNotNil() {
		q = q.Where("tc.user_id = ?", opts.CreatedBy)
	}

	if opts.Visibility != nil {
		q = q.Where("tc.visibility = ?", opts.Visibility)
	}

	if opts.IsDefault != nil {
		q = q.Where("tc.is_default = ?", opts.IsDefault)
	}

	if opts.Search != "" {
		q = q.Where("tc.name ILIKE ? OR tc.description ILIKE ?",
			"%"+opts.Search+"%", "%"+opts.Search+"%")
	}

	return q
}

func (tcr *tableConfigurationRepository) List(ctx context.Context, filters *repositories.TableConfigurationFilters) (*repositories.ListTableConfigurationResult, error) {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "List").
		Str("resource", filters.Resource).
		Logger()

	configs := make([]*tableconfiguration.Configuration, 0)

	q := dba.NewSelect().Model(&configs)

	if filters.IncludeShares {
		q = q.Relation("Shares")
	}

	if filters.IncludeCreator {
		q = q.Relation("Creator")
	}

	q = tcr.filterQuery(q, filters)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan and count table configurations")
		return nil, eris.Wrap(err, "failed to scan and count table configurations")
	}

	return &repositories.ListTableConfigurationResult{
		Configurations: configs,
		Total:          count,
	}, nil
}

func (tcr *tableConfigurationRepository) filterUserConfigurations(q *bun.SelectQuery, opts *repositories.ListUserConfigurationRequest) *bun.SelectQuery {
	q = q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where("tc.user_id = ?", opts.Filter.TenantOpts.UserID).
			Where("tc.organization_id = ?", opts.Filter.TenantOpts.OrgID).
			Where("tc.resource = ?", opts.Resource).
			Where("tc.business_unit_id = ?", opts.Filter.TenantOpts.BuID)
	})

	if opts.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			opts.Filter.Query,
			(*tableconfiguration.Configuration)(nil),
		)
	}

	q = q.Order("tc.is_default DESC", "tc.created_at DESC")

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (tcr *tableConfigurationRepository) ListUserConfigurations(ctx context.Context, opts *repositories.ListUserConfigurationRequest) (*ports.ListResult[*tableconfiguration.Configuration], error) {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "ListUserConfigurations").
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	configs := make([]*tableconfiguration.Configuration, 0)

	q := dba.NewSelect().Model(&configs)

	q = tcr.filterUserConfigurations(q, opts)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user configurations")
		return nil, eris.Wrap(err, "get user configurations")
	}

	return &ports.ListResult[*tableconfiguration.Configuration]{
		Items: configs,
		Total: count,
	}, nil
}

func (tcr *tableConfigurationRepository) GetByID(ctx context.Context, id pulid.ID, opts *repositories.TableConfigurationFilters) (*tableconfiguration.Configuration, error) {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "GetByID").
		Str("id", id.String()).
		Logger()

	config := new(tableconfiguration.Configuration)

	q := dba.NewSelect().Model(config).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tc.id = ?", id).
				Where("tc.organization_id = ?", opts.Base.OrgID).
				Where("tc.business_unit_id = ?", opts.Base.BuID)
		})

	if opts.IncludeShares {
		q = q.Relation("Shares")
	}

	if opts.IncludeCreator {
		q = q.Relation("Creator")
	}

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Configuration not found")
		}

		log.Error().Err(err).Msg("failed to get configuration")
		return nil, err
	}

	return config, nil
}

func (tcr *tableConfigurationRepository) Create(ctx context.Context, config *tableconfiguration.Configuration) (*tableconfiguration.Configuration, error) {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "Create").
		Str("orgID", config.OrganizationID.String()).
		Str("buID", config.BusinessUnitID.String()).
		Logger()

	// * if the incoming config is marked as default, then we need to get the existing default and set it to not default
	//nolint:nestif // This is a nested if statement that is not nested in a larger if statement.
	if config.IsDefault {
		existingDefault := new(tableconfiguration.Configuration)
		err = dba.NewSelect().Model(existingDefault).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where("tc.organization_id = ?", config.OrganizationID).
					Where("tc.resource = ?", config.Resource).
					Where("tc.business_unit_id = ?", config.BusinessUnitID).
					Where("tc.user_id = ?", config.UserID).
					Where("tc.is_default = ?", true)
			}).
			For("UPDATE").
			Scan(ctx)
		if err != nil {
			if eris.Is(err, sql.ErrNoRows) {
				// ! if there is no default configuration we don't need to do anything.
				log.Debug().Msg("no existing default configuration found, moving on...")
			} else {
				log.Error().Err(err).Msg("failed to get existing default configuration")
				return nil, err
			}
		}

		// * we need to update the existing default configuration to not be the default
		_, err = dba.NewUpdate().Model(existingDefault).
			Set("is_default = ?", false).
			Where("tc.id = ?", existingDefault.ID).
			Exec(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to update existing default configuration")
			return nil, err
		}
	}

	// * Now we can create the new configuration
	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(config).Returning("*").Exec(c); iErr != nil {
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

func (tcr *tableConfigurationRepository) Update(ctx context.Context, config *tableconfiguration.Configuration) error {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
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
			return eris.Wrap(rErr, "update configuration")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("config", config).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The configuration (%s) has been updated since your last request.", config.ID.String()),
			)
		}

		return nil
	})
	if err != nil {
		return eris.Wrap(err, "update configuration")
	}

	return nil
}

func (tcr *tableConfigurationRepository) Delete(ctx context.Context, req repositories.DeleteUserConfigurationRequest) error {
	dba, err := tcr.db.DB(ctx)
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
		return eris.Wrap(err, "delete configuration")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return eris.Wrap(err, "get rows affected")
	}

	if rows == 0 {
		return errors.NewNotFoundError("Configuration not found")
	}

	return nil
}

func (tcr *tableConfigurationRepository) GetUserConfigurations(ctx context.Context, resource string, opts *repositories.TableConfigurationFilters) ([]*tableconfiguration.Configuration, error) {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "GetUserConfigurations").
		Str("userID", opts.UserID.String()).
		Str("resource", resource).
		Logger()

	configs := make([]*tableconfiguration.Configuration, 0)

	q := dba.NewSelect().
		Model(&configs).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tc.resource = ?", resource).
				Where("tc.organization_id = ?", opts.Base.OrgID).
				Where("tc.business_unit_id = ?", opts.Base.BuID)
		})

	// * if the default query returns not found, then query for the latest configuration
	q = q.Order("tc.created_at DESC")

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get user configurations")
		return nil, eris.Wrap(err, "get user configurations")
	}

	return configs, nil
}

// GetDefaultOrLatestConfiguration returns the default configuration or the latest if no default exists
func (tcr *tableConfigurationRepository) GetDefaultOrLatestConfiguration(ctx context.Context, resource string, opts *repositories.TableConfigurationFilters) (*tableconfiguration.Configuration, error) {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "GetDefaultOrLatestConfiguration").
		Str("resource", resource).
		Logger()

	config := new(tableconfiguration.Configuration)
	q := dba.NewSelect().Model(config).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tc.resource = ?", resource).
				Where("tc.organization_id = ?", opts.Base.OrgID).
				Where("tc.business_unit_id = ?", opts.Base.BuID)
		})

	// * scan initially to see if there are any configurations that may not be default
	err = q.Scan(ctx)
	if err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Info().Msg("no default configuration found, getting latest")
		} else {
			log.Error().Err(err).Msg("failed to get default or latest configuration")
			return nil, oops.In("table_configuration_repository").
				Tags("get_default_or_latest_configuration").
				With("resource", resource).
				With("orgID", opts.Base.OrgID).
				With("buID", opts.Base.BuID).
				Time(time.Now()).
				Wrapf(err, "get default or latest configuration")
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
			return nil, oops.In("table_configuration_repository").
				Tags("get_default_or_latest_configuration").
				With("resource", resource).
				With("orgID", opts.Base.OrgID).
				With("buID", opts.Base.BuID).
				Time(time.Now()).
				Wrapf(err, "get default or latest configuration")
		}
	}

	return config, nil
}

func (tcr *tableConfigurationRepository) ShareConfiguration(ctx context.Context, share *tableconfiguration.ConfigurationShare) error {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "ShareConfiguration").
		Str("configID", share.ConfigurationID.String()).
		Str("sharedWithID", share.SharedWithID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		// First verify the configuration exists and is shareable
		config := new(tableconfiguration.Configuration)
		err = tx.NewSelect().
			Model(config).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where("id = ?", share.ConfigurationID).
					Where("visibility = ?", tableconfiguration.VisibilityShared)
			}).
			Scan(c)
		if err != nil {
			if eris.Is(err, sql.ErrNoRows) {
				return errors.NewNotFoundError("Configuration not found or not shareable")
			}
			return eris.Wrap(err, "verify configuration")
		}

		// Insert the share
		_, err = tx.NewInsert().
			Model(share).
			Exec(c)
		if err != nil {
			log.Error().Err(err).Msg("failed to create share")
			return eris.Wrap(err, "create share")
		}

		return nil
	})
	if err != nil {
		return eris.Wrap(err, "share configuration")
	}

	return nil
}

func (tcr *tableConfigurationRepository) RemoveShare(ctx context.Context, configID pulid.ID, sharedWithID pulid.ID) error {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
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
