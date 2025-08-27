/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/queryutils/queryfilters"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

// TractorRepositoryParams defines dependencies required for initializing the TractorRepository.
// This includes database connection and logger.
type TractorRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

// tractorRepository implements the TractorRepository interface
// and provides methods to manage tractor data, including CRUD operations.
type tractorRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

// NewTractorRepository initalizes a new instance of tractorRepository with its dependencies.
//
// Parameters:
//   - p: TractorRepositoryParams containing dependencies.
//
// Returns:
//   - repositories.TractorRepository: A ready-to-use tractor repository instance.
func NewTractorRepository(p TractorRepositoryParams) repositories.TractorRepository {
	log := p.Logger.With().
		Str("repository", "tractor").
		Logger()

	return &tractorRepository{
		db: p.DB,
		l:  &log,
	}
}

// addOptions expands the query with related entities based on TractorFilterOptions.
// This allows eager loading of related data like primary and secondary workers, fleet code, and equipment type.
//
// Parameters:
//   - q: The base select query.
//   - opts: Options to determine which related data to include.
//
// Returns:
//   - *bun.SelectQuery: The updated query with the necessary relations.
func (tr *tractorRepository) addOptions(
	q *bun.SelectQuery,
	opts repositories.TractorFilterOptions,
) *bun.SelectQuery {
	// * Include the worker details if requested
	if opts.IncludeWorkerDetails {
		q = q.RelationWithOpts("PrimaryWorker", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("WorkerProfile")
			},
		})

		q = q.RelationWithOpts("SecondaryWorker", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("WorkerProfile")
			},
		})
	}

	// * Include the fleet details if requested
	if opts.IncludeFleetDetails {
		q = q.Relation("FleetCode")
	}

	// * Include the equipment details if requested
	if opts.IncludeEquipmentDetails {
		q = q.Relation("EquipmentType").Relation("EquipmentManufacturer")
	}

	if opts.Status != "" {
		status, err := domain.EquipmentStatusFromString(opts.Status)
		if err != nil {
			tr.l.Error().
				Str("status", opts.Status).
				Err(err).
				Msg("failed to convert status to equipment status")
			return q
		}

		q = q.Where("tr.status = ?", status)
	}

	return q
}

// filterQuery applies filters and pagination to the tractor query.
// It includes tenant-based filtering and full-text search when provided.
//
// Parameters:
//   - q: The base select query.
//   - req: ListTractorRequest containing filter and pagination details.
//
// Returns:
//   - *bun.SelectQuery: The filtered and paginated query.
func (tr *tractorRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListTractorRequest,
) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		TableAlias: "tr",
		Filter:     req.Filter,
	})

	q = tr.addOptions(q, req.FilterOptions)

	// TODO(Wolfred: Add postgres search support.
	if req.Filter.Query != "" {
		q = q.Where(
			"tr.code ILIKE ? OR tr.vin ILIKE ?",
			"%"+req.Filter.Query+"%",
			"%"+req.Filter.Query+"%",
		)
	}

	return q.Order("tr.code ASC", "tr.created_at ASC").
		Limit(req.Filter.Limit).
		Offset(req.Filter.Offset)
}

// List retrieves a list of tractors based on the previous options.
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: ListTractorRequest containing filter and pagination details.
//
// Returns:
//   - *ports.ListResult[*tractor.Tractor]: A list of tractors.
//   - error: An error if the operation fails.
func (tr *tractorRepository) List(
	ctx context.Context,
	req *repositories.ListTractorRequest,
) (*ports.ListResult[*tractor.Tractor], error) {
	dba, err := tr.db.ReadDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "List").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Str("userID", req.Filter.TenantOpts.UserID.String()).
		Logger()

	entities := make([]*tractor.Tractor, 0, req.Filter.Limit)

	q := dba.NewSelect().Model(&entities)
	q = tr.filterQuery(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan tractors")
		return nil, err
	}

	return &ports.ListResult[*tractor.Tractor]{
		Items: entities,
		Total: total,
	}, nil
}

// GetByID retrieves a tractor by its ID.
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: GetTractorByIDRequest containing the tractor ID and filter options.
//
// Returns:
//   - *tractor.Tractor: The tractor entity.
//   - error: An error if the operation fails.
func (tr *tractorRepository) GetByID(
	ctx context.Context,
	req *repositories.GetTractorByIDRequest,
) (*tractor.Tractor, error) {
	dba, err := tr.db.ReadDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "GetByID").
		Str("tractorID", req.TractorID.String()).
		Logger()

	entity := new(tractor.Tractor)

	query := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("tr.id = ?", req.TractorID).
				Where("tr.organization_id = ?", req.OrgID).
				Where("tr.business_unit_id = ?", req.BuID)
		})

	query = tr.addOptions(query, req.FilterOptions)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Tractor not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get tractor")
		return nil, err
	}

	return entity, nil
}

// GetByPrimaryWorkerID retrieves a tractor by its primary worker ID.
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: GetByPrimaryWorkerIDRequest containing the worker ID and organization ID.
//
// Returns:
//   - *tractor.Tractor: The tractor entity.
//   - error: An error if the operation fails.
func (tr *tractorRepository) GetByPrimaryWorkerID(
	ctx context.Context,
	req repositories.GetTractorByPrimaryWorkerIDRequest,
) (*tractor.Tractor, error) {
	dba, err := tr.db.ReadDB(ctx)
	if err != nil {
		return nil, oops.In("tractor_repository").
			With("op", "get_by_primary_worker_id").
			With("req", req).
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "GetByPrimaryWorkerID").
		Str("workerID", req.WorkerID.String()).
		Logger()

	entity := new(tractor.Tractor)

	query := dba.NewSelect().Model(entity).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("tr.primary_worker_id = ?", req.WorkerID).
				Where("tr.organization_id = ?", req.OrgID).
				Where("tr.business_unit_id = ?", req.BuID)
		})

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Tractor not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get tractor")
		return nil, err
	}

	return entity, nil
}

// Create creates a new tractor.
//
// Parameters:
//   - ctx: The context for the operation.
//   - t: The tractor entity to create.
//
// Returns:
//   - *tractor.Tractor: The created tractor entity.
//   - error: An error if the operation fails.
func (tr *tractorRepository) Create(
	ctx context.Context,
	t *tractor.Tractor,
) (*tractor.Tractor, error) {
	dba, err := tr.db.WriteDB(ctx)
	if err != nil {
		return nil, oops.
			In("tractor_repository").
			With("op", "create").
			Time(time.Now()).
			Wrapf(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "Create").
		Str("orgID", t.OrganizationID.String()).
		Str("buID", t.BusinessUnitID.String()).
		Logger()

	if _, err = dba.NewInsert().Model(t).Exec(ctx); err != nil {
		log.Error().
			Err(err).
			Interface("tractor", t).
			Msg("failed to insert tractor")
		return nil, err
	}

	return t, nil
}

// Update updates a tractor.
//
// Parameters:
//   - ctx: The context for the operation.
//   - t: The tractor entity to update.
//
// Returns:
//   - *tractor.Tractor: The updated tractor entity.
//   - error: An error if the operation fails.
func (tr *tractorRepository) Update(
	ctx context.Context,
	t *tractor.Tractor,
) (*tractor.Tractor, error) {
	dba, err := tr.db.WriteDB(ctx)
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
			OmitZero().
			Where("tr.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("tractor", t).
				Msg("failed to update tractor")
			return err
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("tractor", t).
				Msg("failed to get rows affected")
			return err
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf(
					"Version mismatch. The Tractor (%s) has either been updated or deleted since the last request.",
					t.GetID(),
				),
			)
		}

		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to update tractor")
		return nil, err
	}

	return t, nil
}

// Assignment assigns a primary and secondary worker to a tractor.
//
// Parameters:
//   - ctx: The context for the operation.
//   - opts: AssignmentOptions containing the tractor ID and organization ID.
//
// Returns:
//   - *repositories.AssignmentResponse: The assignment response.
//   - error: An error if the operation fails.
func (tr *tractorRepository) Assignment(
	ctx context.Context,
	opts repositories.TractorAssignmentRequest,
) (*repositories.AssignmentResponse, error) {
	dba, err := tr.db.ReadDB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := tr.l.With().
		Str("operation", "Assignment").
		Str("tractorID", opts.TractorID.String()).
		Logger()

	entity := new(tractor.Tractor)

	q := dba.NewSelect().Model(entity).
		Column("tr.primary_worker_id", "tr.secondary_worker_id").
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("tr.id = ?", opts.TractorID).
				Where("tr.organization_id = ?", opts.OrgID).
				Where("tr.business_unit_id = ?", opts.BuID)
		})

	if err = q.Scan(ctx); err != nil {
		log.Error().Err(err).Msg("failed to get tractor")
	}

	return &repositories.AssignmentResponse{
		PrimaryWorkerID:   entity.PrimaryWorkerID,
		SecondaryWorkerID: entity.SecondaryWorkerID,
	}, nil
}
