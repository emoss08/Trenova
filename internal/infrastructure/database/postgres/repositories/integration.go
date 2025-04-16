package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// IntegrationRepositoryParams contains the dependencies for the IntegrationRepository.
type IntegrationRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// integrationRepository implements the IntegrationRepository interface.
type integrationRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewIntegrationRepository initializes a new instance of integrationRepository.
func NewIntegrationRepository(p IntegrationRepositoryParams) repositories.IntegrationRepository {
	log := p.Logger.With().
		Str("repository", "integration").
		Logger()

	return &integrationRepository{
		db: p.DB,
		l:  &log,
	}
}

func (r *integrationRepository) filterQuery(q *bun.SelectQuery, opts *ports.LimitOffsetQueryOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "i",
		Filter:     opts,
	})

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

// List returns a paginated list of integrations.
func (r *integrationRepository) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*integration.Integration], error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "List").
		Logger()

	entities := make([]*integration.Integration, 0)

	q := dba.NewSelect().Model(&entities)

	// * Filter the query
	q = r.filterQuery(q, opts)

	// * Order by name and created at
	q.Order("i.name ASC", "i.created_at DESC")

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to list integrations")
		return nil, err
	}

	return &ports.ListResult[*integration.Integration]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID returns an integration by ID.
func (r *integrationRepository) GetByID(ctx context.Context, opts repositories.GetIntegrationByIDOptions) (*integration.Integration, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByID").
		Str("id", opts.ID.String()).
		Str("orgID", opts.OrgID.String()).
		Str("buID", opts.BuID.String()).
		Logger()

	entity := new(integration.Integration)

	q := dba.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("i.id = ?", opts.ID).
				Where("i.organization_id = ?", opts.OrgID).
				Where("i.business_unit_id = ?", opts.BuID)
		})

	if err = q.Scan(ctx); err != nil {
		// * If the query is [sql.ErrNoRows], return a not found error
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Msg("integration not found")
			return nil, errors.NewNotFoundError("Integration not found")
		}

		log.Error().Err(err).Msg("failed to get integration")
		return nil, eris.Wrap(err, "get integration")
	}

	return entity, nil
}

func (r *integrationRepository) GetByType(ctx context.Context, req repositories.GetIntegrationByTypeRequest) (*integration.Integration, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "GetByType").
		Str("type", string(req.Type)).
		Logger()

	entity := new(integration.Integration)

	q := dba.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("i.type = ?", req.Type).
				Where("i.organization_id = ?", req.OrgID).
				Where("i.business_unit_id = ?", req.BuID)
		})

	if err = q.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			log.Error().Msg("integration not found")
			return nil, errors.NewNotFoundError("Integration not found")
		}

		log.Error().Err(err).Msg("failed to get integration")
		return nil, eris.Wrap(err, "get integration")
	}

	return entity, nil
}

// Update updates an integration.
func (r *integrationRepository) Update(ctx context.Context, i *integration.Integration) (*integration.Integration, error) {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "Update").
		Str("id", i.GetID()).
		Int64("version", i.GetVersion()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		oldVersion := i.Version
		i.Version++

		results, rErr := tx.NewUpdate().
			Model(i).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("i.id = ?", i.GetID()).
					Where("i.organization_id = ?", i.OrganizationID).
					Where("i.business_unit_id = ?", i.BusinessUnitID).
					Where("i.version = ?", oldVersion)
			}).
			// * Just set the configuration field, not the entire integration
			Set("configuration = ?", i.Configuration).
			Returning("*").
			Exec(c)
		if rErr != nil {
			if eris.Is(rErr, sql.ErrNoRows) {
				log.Error().Msg("integration not found")
				return errors.NewNotFoundError("Integration not found")
			}
			log.Error().Err(rErr).Msg("failed to update integration")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().Err(roErr).Msg("failed to get rows affected")
			return roErr
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The integration (%s) has either been updated or deleted since the last request.", i.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update integration")
		return nil, err
	}

	return i, nil
}

// RecordUsage increments the usage count and updates the last used timestamp.
func (r *integrationRepository) RecordUsage(ctx context.Context, id, orgID, buID pulid.ID) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "RecordUsage").
		Str("id", id.String()).
		Str("orgID", orgID.String()).
		Str("buID", buID.String()).
		Logger()

	result, err := dba.NewUpdate().
		Table("integrations").
		Set("usage_count = usage_count + 1").
		Set("last_used = extract(epoch from current_timestamp)::bigint").
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("i.id = ?", id).
				Where("i.organization_id = ?", orgID).
				Where("i.business_unit_id = ?", buID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to record integration usage")
		return eris.Wrap(err, "record integration usage")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return eris.Wrap(err, "get rows affected")
	}

	if rows == 0 {
		return errors.NewNotFoundError("Integration not found")
	}

	return nil
}

// RecordError records an error occurrence.
func (r *integrationRepository) RecordError(ctx context.Context, id, orgID, buID pulid.ID, errorMessage string) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "RecordError").
		Str("id", id.String()).
		Str("orgID", orgID.String()).
		Str("buID", buID.String()).
		Logger()

	// Update status to "error" and increment error count
	result, err := dba.NewUpdate().
		Table("integrations").
		Set("enabled = ?", false).
		Set("error_count = error_count + 1").
		Set("last_error = ?", errorMessage).
		Set("last_error_at = extract(epoch from current_timestamp)::bigint").
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("i.id = ?", id).
				Where("i.organization_id = ?", orgID).
				Where("i.business_unit_id = ?", buID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to record integration error")
		return eris.Wrap(err, "record integration error")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return eris.Wrap(err, "get rows affected")
	}

	if rows == 0 {
		return errors.NewNotFoundError("Integration not found")
	}

	return nil
}

// ClearError clears the error state.
func (r *integrationRepository) ClearError(ctx context.Context, id, orgID, buID pulid.ID) error {
	dba, err := r.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := r.l.With().
		Str("operation", "ClearError").
		Str("id", id.String()).
		Str("orgID", orgID.String()).
		Str("buID", buID.String()).
		Logger()

	// Update status to "active" and clear error info
	result, err := dba.NewUpdate().
		Table("integrations").
		Set("enabled = ?", true).
		Set("last_error = NULL").
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("i.id = ?", id).
				Where("i.organization_id = ?", orgID).
				Where("i.business_unit_id = ?", buID)
		}).
		Exec(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to clear integration error")
		return eris.Wrap(err, "clear integration error")
	}

	rows, err := result.RowsAffected()
	if err != nil {
		log.Error().Err(err).Msg("failed to get rows affected")
		return eris.Wrap(err, "get rows affected")
	}

	if rows == 0 {
		return errors.NewNotFoundError("Integration not found")
	}

	return nil
}
