package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
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

type TrailerRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type trailerRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewTrailerRepository(p TrailerRepositoryParams) repositories.TrailerRepository {
	log := p.Logger.With().
		Str("repository", "trailer").
		Logger()

	return &trailerRepository{
		db: p.DB,
		l:  &log,
	}
}

func (tr *trailerRepository) filterQuery(q *bun.SelectQuery, opts *repositories.ListTrailerOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "tr",
		Filter:     opts.Filter,
	})

	if opts.IncludeEquipmentDetails {
		q = q.Relation("EquipmentType").Relation("EquipmentManufacturer")
	}

	if opts.IncludeFleetDetails {
		q = q.Relation("FleetCode")
	}

	if opts.Filter.Query != "" {
		q = q.Where("tr.code ILIKE ?", "%"+opts.Filter.Query+"%")
	}

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (tr *trailerRepository) List(ctx context.Context, opts *repositories.ListTrailerOptions) (*ports.ListResult[*trailer.Trailer], error) {
	dba, err := tr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "List").
		Str("buID", opts.Filter.TenantOpts.BuID.String()).
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*trailer.Trailer, 0)

	q := dba.NewSelect().Model(&entities)
	q = tr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan trailers")
		return nil, err
	}

	return &ports.ListResult[*trailer.Trailer]{
		Items: entities,
		Total: total,
	}, nil
}

func (tr *trailerRepository) GetByID(ctx context.Context, opts repositories.GetTrailerByIDOptions) (*trailer.Trailer, error) {
	dba, err := tr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "GetByID").
		Str("trailerID", opts.ID.String()).
		Logger()

	entity := new(trailer.Trailer)

	query := dba.NewSelect().Model(entity).
		Where("tr.id = ? AND tr.organization_id = ? AND tr.business_unit_id = ?", opts.ID, opts.OrgID, opts.BuID)

	if opts.IncludeEquipmentDetails {
		query = query.Relation("EquipmentType").Relation("EquipmentManufacturer")
	}

	if opts.IncludeFleetDetails {
		query = query.Relation("FleetCode")
	}

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Trailer not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get trailer")
		return nil, err
	}

	return entity, nil
}

func (tr *trailerRepository) Create(ctx context.Context, t *trailer.Trailer) (*trailer.Trailer, error) {
	dba, err := tr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "Create").
		Str("orgID", t.OrganizationID.String()).
		Str("buID", t.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(t).Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("trailer", t).
				Msg("failed to insert trailer")
			return err
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create trailer")
		return nil, err
	}

	return t, nil
}

func (tr *trailerRepository) Update(ctx context.Context, t *trailer.Trailer) (*trailer.Trailer, error) {
	dba, err := tr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "Update").
		Str("id", t.GetID()).
		Int64("version", t.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := t.Version

		t.Version++

		results, rErr := tx.NewUpdate().
			Model(t).
			WherePK().
			Where("tr.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("trailer", t).
				Msg("failed to update trailer")
			return err
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("trailer", t).
				Msg("failed to get rows affected")
			return err
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The Trailer (%s) has either been updated or deleted since the last request.", t.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update trailer")
		return nil, err
	}

	return t, nil
}
