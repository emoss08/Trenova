package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tableconfiguration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
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

	DB           db.Connection
	Logger       *logger.Logger
	AuditService services.AuditService
}

type tableConfigurationRepository struct {
	db           db.Connection
	l            *zerolog.Logger
	auditService services.AuditService
}

func NewTableConfigurationRepository(p TableConfigurationRepositoryParams) repositories.TableConfigurationRepository {
	log := p.Logger.With().
		Str("repository", "table_configuration").
		Logger()

	return &tableConfigurationRepository{
		db:           p.DB,
		l:            &log,
		auditService: p.AuditService,
	}
}

func (tcr *tableConfigurationRepository) filterQuery(q *bun.SelectQuery, opts *repositories.TableConfigurationFilters) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query: q,
		// Filter:     opts.Base,
		TableAlias: "tbl_cfg",
	})

	if opts.TableIdentifier != "" {
		q = q.Where("tc.table_identifier = ?", opts.TableIdentifier)
	}

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
		Str("tableIdentifier", filters.TableIdentifier).
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

func (tcr *tableConfigurationRepository) Create(ctx context.Context, config *tableconfiguration.Configuration) error {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "Create").
		Str("orgID", config.OrganizationID.String()).
		Str("buID", config.BusinessUnitID.String()).
		Logger()

	// * if the incoming config is marked as default, then we need to get the existing default and set it to not default
	if config.IsDefault {
		dc := new(tableconfiguration.Configuration)
		_, err = dba.NewUpdate().
			Model(dc).
			Set("is_default = ?", false).
			WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
				return sq.Where("tc.id = ?", dc.ID).
					Where("tc.organization_id = ?", config.OrganizationID).
					Where("tc.business_unit_id = ?", config.BusinessUnitID).
					Where("tc.is_default = ?", true).
					Where("tc.table_identifier = ?", config.TableIdentifier).
					Where("tc.user_id = ?", config.UserID)
			}).
			Returning("*").
			Exec(ctx)

		if err != nil {
			if eris.Is(err, sql.ErrNoRows) {
				log.Info().Msg("no existing default configuration found")
			}

			log.Error().Err(err).Msg("failed to update existing default configuration")
			return oops.In("table_configuration_repository").
				Tags("create").
				With("config", config).
				With("orgID", config.OrganizationID).
				With("buID", config.BusinessUnitID).
				With("userID", config.UserID).
				Time(time.Now()).
				Wrapf(err, "update existing default configuration")
		}
	}

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if err := config.DBValidate(c, tx); err != nil {
			return err
		}

		if _, iErr := tx.NewInsert().Model(config).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("config", config).
				Msg("failed to insert configuration")
			return eris.Wrap(iErr, "insert configuration")
		}
		return nil
	})
	if err != nil {
		return eris.Wrap(err, "failed to create table configuration")
	}

	err = tcr.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTableConfiguration,
			ResourceID:     config.ID.String(),
			Action:         permission.ActionCreate,
			UserID:         config.UserID,
			CurrentState:   jsonutils.MustToJSON(config),
			OrganizationID: config.OrganizationID,
			BusinessUnitID: config.BusinessUnitID,
		},
		audit.WithComment("Table configuration created"),
		audit.WithDiff(nil, config),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log table configuration creation")
	}

	return nil
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

	// Get original for audit
	original, err := tcr.GetByID(ctx, config.ID, &repositories.TableConfigurationFilters{
		Base: &ports.FilterQueryOptions{
			OrgID: config.OrganizationID,
			BuID:  config.BusinessUnitID,
		},
	})
	if err != nil {
		return eris.Wrap(err, "get configuration")
	}

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

	// Log the update
	err = tcr.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTableConfiguration,
			ResourceID:     config.ID.String(),
			Action:         permission.ActionUpdate,
			UserID:         config.UserID,
			CurrentState:   jsonutils.MustToJSON(config),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: config.OrganizationID,
			BusinessUnitID: config.BusinessUnitID,
		},
		audit.WithComment("Table configuration updated"),
		audit.WithDiff(original, config),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log configuration update")
	}

	return nil
}

func (tcr *tableConfigurationRepository) Delete(ctx context.Context, id pulid.ID) error {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "Delete").
		Str("id", id.String()).
		Logger()

	// Get original for audit
	original, err := tcr.GetByID(ctx, id, &repositories.TableConfigurationFilters{})
	if err != nil {
		return eris.Wrap(err, "get configuration")
	}

	result, err := dba.NewDelete().
		Model((*tableconfiguration.Configuration)(nil)).
		Where("id = ?", id).
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

	// Log the deletion
	err = tcr.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceTableConfiguration,
			ResourceID:     id.String(),
			Action:         permission.ActionDelete,
			UserID:         original.UserID,
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: original.OrganizationID,
			BusinessUnitID: original.BusinessUnitID,
		},
		audit.WithComment("Table configuration deleted"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log configuration deletion")
	}

	return nil
}

func (tcr *tableConfigurationRepository) GetUserConfigurations(
	ctx context.Context, tableID string, opts *repositories.TableConfigurationFilters,
) ([]*tableconfiguration.Configuration, error) {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "GetUserConfigurations").
		Str("userID", opts.UserID.String()).
		Str("tableID", tableID).
		Logger()

	configs := make([]*tableconfiguration.Configuration, 0)

	q := dba.NewSelect().
		Model(&configs).
		Where("tc.table_identifier = ?", tableID).
		Where("tc.organization_id = ?", opts.Base.OrgID).
		Where("tc.business_unit_id = ?", opts.Base.BuID)

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get user configurations")
		return nil, eris.Wrap(err, "get user configurations")
	}

	return configs, nil
}

// GetDefaultOrLatestConfiguration returns the default configuration or the latest if no default exists
func (tcr *tableConfigurationRepository) GetDefaultOrLatestConfiguration(ctx context.Context, tableID string, opts *repositories.TableConfigurationFilters) (*tableconfiguration.Configuration, error) {
	dba, err := tcr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tcr.l.With().
		Str("operation", "GetDefaultOrLatestConfiguration").
		Str("tableID", tableID).
		Logger()

	config := new(tableconfiguration.Configuration)
	q := dba.NewSelect().Model(config).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("tc.table_identifier = ?", tableID).
				Where("tc.organization_id = ?", opts.Base.OrgID).
				Where("tc.business_unit_id = ?", opts.Base.BuID)
		})

	// * Query for the default configuration
	q = q.Where("tc.is_default = ?", true)

	// * if the default query returns not found, then query for the latest configuration
	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			q.Where("tc.is_default = ?", false)
			log.Info().Msg("no default configuration found, querying for latest")
		} else {
			return nil, eris.Wrap(err, "get default or latest configuration")
		}
	}

	q = q.Order("tc.created_at DESC")

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get default or latest configuration")
		return nil, eris.Wrap(err, "get default or latest configuration")
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
			Where("id = ? AND visibility = ?",
				share.ConfigurationID,
				tableconfiguration.VisibilityShared).
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
		Where("configuration_id = ? AND shared_with_id = ?", configID, sharedWithID).
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
