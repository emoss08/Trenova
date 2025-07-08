package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/location"
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

type LocationRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type locationRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewLocationRepository(p LocationRepositoryParams) repositories.LocationRepository {
	log := p.Logger.With().
		Str("repository", "location").
		Logger()

	return &locationRepository{
		db: p.DB,
		l:  &log,
	}
}

func (lr *locationRepository) filterQuery(
	q *bun.SelectQuery,
	opts *repositories.ListLocationOptions,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "loc",
		Filter:     opts.Filter,
	})

	if opts.IncludeCategory {
		q = q.Relation("LocationCategory")
	}

	if opts.IncludeState {
		q = q.Relation("State")
	}

	if opts.Status != "" {
		status, err := domain.StatusFromString(opts.Status)
		if err != nil {
			lr.l.Error().Err(err).Str("status", opts.Status).Msg("invalid status")
			return q
		}

		q = q.Where("loc.status = ?", status)
	}

	if opts.Filter.Query != "" {
		q = q.Where(
			"loc.name ILIKE ? OR loc.code ILIKE ?",
			"%"+opts.Filter.Query+"%",
			"%"+opts.Filter.Query+"%",
		)
	}

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (lr *locationRepository) List(
	ctx context.Context,
	opts *repositories.ListLocationOptions,
) (*ports.ListResult[*location.Location], error) {
	dba, err := lr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := lr.l.With().
		Str("operation", "List").
		Str("buID", opts.Filter.TenantOpts.BuID.String()).
		Str("userID", opts.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*location.Location, 0)

	q := dba.NewSelect().Model(&entities)
	q = lr.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan locations")
		return nil, eris.Wrap(err, "scan locations")
	}

	return &ports.ListResult[*location.Location]{
		Items: entities,
		Total: total,
	}, nil
}

func (lr *locationRepository) GetByID(
	ctx context.Context,
	opts repositories.GetLocationByIDOptions,
) (*location.Location, error) {
	dba, err := lr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := lr.l.With().
		Str("operation", "GetByID").
		Str("locationID", opts.ID.String()).
		Logger()

	entity := new(location.Location)

	query := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("loc.id = ?", opts.ID).
				Where("loc.organization_id = ?", opts.OrgID).
				Where("loc.business_unit_id = ?", opts.BuID)
		})

	if opts.IncludeCategory {
		query = query.Relation("LocationCategory")
	}

	if opts.IncludeState {
		query = query.Relation("State")
	}

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Location not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get location")
		return nil, eris.Wrap(err, "get location")
	}

	return entity, nil
}

func (lr *locationRepository) Create(
	ctx context.Context,
	l *location.Location,
) (*location.Location, error) {
	dba, err := lr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := lr.l.With().
		Str("operation", "Create").
		Str("orgID", l.OrganizationID.String()).
		Str("buID", l.BusinessUnitID.String()).
		Logger()

	// * Check if the location needs to be geocoded
	l = lr.GeocodeIfApplicable(l)

	if _, err = dba.NewInsert().Model(l).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Interface("location", l).
			Msg("failed to insert location")
		return nil, err
	}

	return l, nil
}

func (lr *locationRepository) Update(
	ctx context.Context,
	loc *location.Location,
) (*location.Location, error) {
	dba, err := lr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := lr.l.With().
		Str("operation", "Update").
		Str("id", loc.GetID()).
		Int64("version", loc.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := loc.Version

		loc.Version++

		loc = lr.GeocodeIfApplicable(loc)

		results, rErr := tx.NewUpdate().
			Model(loc).
			Where("loc.version = ?", ov).
			OmitZero().
			WherePK().
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("location", loc).
				Msg("failed to update location")
			return rErr
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("location", loc).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Location (%s) has either been updated or deleted since the last request.",
					loc.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update location")
		return nil, err
	}

	return loc, nil
}

func (lr *locationRepository) GeocodeIfApplicable(l *location.Location) *location.Location {
	if l.PlaceID == "" || l.Latitude == nil || l.Longitude == nil {
		l.IsGeocoded = false
		return l
	}

	l.IsGeocoded = true
	return l
}
