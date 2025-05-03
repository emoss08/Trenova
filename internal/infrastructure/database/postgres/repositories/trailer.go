package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/postgressearch"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// TrailerRepositoryParams defines dependencies required for initializing the TrailerRepository.
// This includes database connection and logger.
type TrailerRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// trailerRepository implements the TrailerRepository interface
// and provides methods to manage trailer data, including CRUD operations.
type trailerRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewTrailerRepository initalizes a new instance of trailerRepository with its dependencies.
//
// Parameters:
//   - p: TrailerRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.TrailerRepository: A ready-to-use trailer repository instance.
func NewTrailerRepository(p TrailerRepositoryParams) repositories.TrailerRepository {
	log := p.Logger.With().
		Str("repository", "trailer").
		Logger()

	return &trailerRepository{
		db: p.DB,
		l:  &log,
	}
}

// addOptions expands the query with related entities based on TrailerFilterOptions.
// This allows eager loading of related data like equipment type and fleet code.
//
// Parameters:
//   - q: The base select query.
//   - opts: TrailerFilterOptions containing filter options.
//
// Returns:
//   - *bun.SelectQuery: The updated query with the necessary relations.
func (tr *trailerRepository) addOptions(q *bun.SelectQuery, opts repositories.TrailerFilterOptions) *bun.SelectQuery {
	// * Include the equipment details if requested
	if opts.IncludeEquipmentDetails {
		q = q.Relation("EquipmentType").Relation("EquipmentManufacturer")
	}

	// * Include the fleet details if requested
	if opts.IncludeFleetDetails {
		q = q.Relation("FleetCode")
	}

	if opts.Status != "" {
		status, err := domain.EquipmentStatusFromString(opts.Status)
		if err != nil {
			return q
		}

		q = q.Where("tr.status = ?", status)
	}

	return q
}

// filterQuery applies filters and pagination to the trailer query.
// It includes tenant-based filtering and full-text search when provided.
//
// Parameters:
//   - q: The base select query.
//   - opts: ListTrailerOptions containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (tr *trailerRepository) filterQuery(q *bun.SelectQuery, opts *repositories.ListTrailerOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "tr",
		Filter:     opts.Filter,
	})

	q = tr.addOptions(q, opts.FilterOptions)

	if opts.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			opts.Filter.Query,
			(*trailer.Trailer)(nil),
		)
	}

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

// List retrieves a list of trailers based on the previous options.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: ListTrailerOptions containing filter and pagination details.
//
// Returns:
//   - *ports.ListResult[*trailer.Trailer]: A list of trailers.
//   - error: An error if the operation fails.
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

// GetByID retrieves a trailer by its ID.
//
// Parameters:
//   - ctx : The context for the operation.
//   - opts: GetTrailerByIDOptions containing Trailer ID and tentant options.
//
// Returns:
//   - *trailer.Trailer: The trailer entity.
//   - error: An error if the operation fails.
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
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("tr.id = ?", opts.ID).
				Where("tr.organization_id = ?", opts.OrgID).
				Where("tr.business_unit_id = ?", opts.BuID)
		})

	query = tr.addOptions(query, opts.FilterOptions)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Trailer not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get trailer")
		return nil, err
	}

	return entity, nil
}

// Create a trailer
//
// Parameters:
//   - ctx: the context for the operation.
//   - t: The trailer entity to create
//
// Returns:
//   - *trailer.Trailer: The created trailer entity.
//   - error: An error if the operation fails.
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

// Update a trailer
//
// Parameters:
//   - ctx: the context for the operation.
//   - t: The trailer entity to update
//
// Returns:
//   - *trailer.Trailer: The updated trailer entity.
//   - error: An error if the operation fails.
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
