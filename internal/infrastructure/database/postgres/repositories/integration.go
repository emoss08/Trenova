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
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type IntegrationRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type integrationRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewIntegrationRepository(p IntegrationRepositoryParams) repositories.IntegrationRepository {
	log := p.Logger.With().
		Str("repository", "integration").
		Logger()

	return &integrationRepository{
		db: p.DB,
		l:  &log,
	}
}

func (r *integrationRepository) filterQuery(
	q *bun.SelectQuery,
	opts *ports.LimitOffsetQueryOptions,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "i",
		Filter:     opts,
	})

	q.Relation("EnabledBy")

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

func (r *integrationRepository) List(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*integration.Integration], error) {
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

func (r *integrationRepository) GetByID(
	ctx context.Context,
	opts repositories.GetIntegrationByIDOptions,
) (*integration.Integration, error) {
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

	q.Relation("EnabledBy")

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

func (r *integrationRepository) GetByType(
	ctx context.Context,
	req repositories.GetIntegrationByTypeRequest,
) (*integration.Integration, error) {
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

func (r *integrationRepository) Update(
	ctx context.Context,
	i *integration.Integration,
) (*integration.Integration, error) {
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
			Set("enabled = ?", i.Enabled).
			Set("enabled_by_id = ?", i.EnabledByID).
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
				fmt.Sprintf(
					"Version mismatch. The integration (%s) has either been updated or deleted since the last request.",
					i.GetID(),
				),
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
