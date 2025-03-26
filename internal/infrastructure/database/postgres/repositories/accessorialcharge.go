package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
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

// AccessorialChargeRepositoryParams defines the dependencies required for initializing the AccessorialChargeRepository.
// This includes the database connection and the logger.
type AccessorialChargeRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// accessorialChargeRepository implements the AccessorialChargeRepository interface
// and provides methods to interact with accessorial charges, including CRUD operations.
type accessorialChargeRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewAccessorialChargeRepository initializes a new AccessorialChargeRepository with the provided dependencies.
// It creates a new repository instance and returns it.
func NewAccessorialChargeRepository(p AccessorialChargeRepositoryParams) repositories.AccessorialChargeRepository {
	log := p.Logger.With().
		Str("repository", "accessorialcharge").
		Logger()

	return &accessorialChargeRepository{
		db: p.DB,
		l:  &log,
	}
}

// filterQuery applies filters and pagination to the accessorial charge query.
// It includes tenant-based filtering and full-text search when provided.
//
// Parameters:
//   - q: The base select query.
//   - opts: ListAccessorialChargeRequest containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (ac *accessorialChargeRepository) filterQuery(q *bun.SelectQuery, opts *ports.LimitOffsetQueryOptions) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "acc",
		Filter:     opts,
	})

	if opts.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			opts.Query,
			(*accessorialcharge.AccessorialCharge)(nil),
		)
	}

	return q.Limit(opts.Limit).Offset(opts.Offset)
}

// List retrieves accessorial charges based on filtering and pagination options.
// It returns a list of accessorial charges along with the total count.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: ListAccessorialChargeRequest containing filter and pagination details.
//
// Returns:
//   - *ports.ListResult[*accessorialcharge.AccessorialCharge]: The list of accessorial charges along with the total count.
//   - error: An error if the operation fails.
func (ac *accessorialChargeRepository) List(ctx context.Context, opts *ports.LimitOffsetQueryOptions) (*ports.ListResult[*accessorialcharge.AccessorialCharge], error) {
	dba, err := ac.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ac.l.With().
		Str("operation", "List").
		Str("buID", opts.TenantOpts.BuID.String()).
		Str("userID", opts.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*accessorialcharge.AccessorialCharge, 0)

	q := dba.NewSelect().Model(&entities)
	q = ac.filterQuery(q, opts)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan hazardous materials")
		return nil, eris.Wrap(err, "scan hazardous materials")
	}

	return &ports.ListResult[*accessorialcharge.AccessorialCharge]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID retrieves an accessorial charge by its ID.
// It returns the accessorial charge if found, or an error if it does not exist or the operation fails.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: GetAccessorialChargeByIDRequest containing the ID and tenant details.
//
// Returns:
//   - *accessorialcharge.AccessorialCharge: The accessorial charge if found.
//   - error: An error if the operation fails.s
func (ac *accessorialChargeRepository) GetByID(ctx context.Context, opts repositories.GetAccessorialChargeByIDRequest) (*accessorialcharge.AccessorialCharge, error) {
	dba, err := ac.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ac.l.With().
		Str("operation", "GetByID").
		Str("accessorialChargeID", opts.ID.String()).
		Logger()

	entity := new(accessorialcharge.AccessorialCharge)

	query := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("acc.id = ?", opts.ID).
				Where("acc.organization_id = ?", opts.OrgID).
				Where("acc.business_unit_id = ?", opts.BuID)
		})

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Accessorial Charge not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get accessorial charge")
		return nil, eris.Wrap(err, "get accessorial charge")
	}

	return entity, nil
}

// Create inserts a new accessorial charge into the database.
// It returns the created accessorial charge if successful, or an error if the operation fails.
//
// Parameters:
//   - ctx: The context for the operation.
//   - acc: The accessorial charge to be created.
//
// Returns:
//   - *accessorialcharge.AccessorialCharge: The created accessorial charge.
//   - error: An error if the operation fails.
func (ac *accessorialChargeRepository) Create(ctx context.Context, acc *accessorialcharge.AccessorialCharge) (*accessorialcharge.AccessorialCharge, error) {
	dba, err := ac.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ac.l.With().
		Str("operation", "Create").
		Str("orgID", acc.OrganizationID.String()).
		Str("buID", acc.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().Model(acc).Returning("*").Exec(c); iErr != nil {
			log.Error().Err(iErr).Msg("failed to insert accessorial charge")
			return iErr
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create accessorial charge")
		return nil, eris.Wrap(err, "create accessorial charge")
	}

	return acc, nil
}

// Update updates an existing accessorial charge in the database.
// It returns the updated accessorial charge if successful, or an error if the operation fails.
//
// Parameters:
//   - ctx: The context for the operation.
//   - acc: The accessorial charge to be updated.
//
// Returns:
//   - *accessorialcharge.AccessorialCharge: The updated accessorial charge.
//   - error: An error if the operation fails.
func (ac *accessorialChargeRepository) Update(ctx context.Context, acc *accessorialcharge.AccessorialCharge) (*accessorialcharge.AccessorialCharge, error) {
	dba, err := ac.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := ac.l.With().
		Str("operation", "Update").
		Str("id", acc.GetID()).
		Int64("version", acc.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := acc.Version

		acc.Version++

		results, rErr := tx.NewUpdate().
			Model(acc).
			WherePK().
			Where("acc.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().Err(rErr).Msg("failed to update accessorial charge")
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
				fmt.Sprintf("Version mismatch. The Accessorial Charge (%s) has either been updated or deleted since the last request.", acc.GetID()),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update accessorial charge")
		return nil, eris.Wrap(err, "update accessorial charge")
	}

	return acc, nil
}
