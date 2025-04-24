package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/worker"
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
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type WorkerRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type workerRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewWorkerRepository(p WorkerRepositoryParams) repositories.WorkerRepository {
	log := p.Logger.With().
		Str("repository", "worker").
		Logger()

	return &workerRepository{
		db: p.DB,
		l:  &log,
	}
}

func (wr *workerRepository) addOptions(q *bun.SelectQuery, opts repositories.WorkerFilterOptions) *bun.SelectQuery {
	// * Include the profile if requested
	if opts.IncludeProfile {
		q = q.Relation("Profile")
	}

	// * Include the PTO if requested
	if opts.IncludePTO {
		wr.l.Info().Msg("Including Paid Time Off")
		q = q.Relation("PTO", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order("start_date ASC")
		})
	}

	if opts.Status != "" {
		status, err := domain.StatusFromString(opts.Status)
		if err != nil {
			return q
		}

		q = q.Where("wrk.status = ?", status)
	}

	return q.Order("wrk.created_at DESC", "wrk.first_name ASC", "wrk.last_name ASC")
}

func (wr *workerRepository) filterQuery(q *bun.SelectQuery, req *repositories.ListWorkerRequest) *bun.SelectQuery {
	q = queryfilters.TenantFilterQuery(&queryfilters.TenantFilterQueryOptions{
		Query:      q,
		Filter:     req.Filter,
		TableAlias: "wrk",
	})

	if req.Filter.Query != "" {
		q = postgressearch.BuildSearchQuery(
			q,
			req.Filter.Query,
			(*worker.Worker)(nil),
		)
	}

	q = wr.addOptions(q, req.FilterOptions)

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

// List returns a list of workers in the database
func (wr *workerRepository) List(ctx context.Context, req *repositories.ListWorkerRequest) (*ports.ListResult[*worker.Worker], error) {
	dba, err := wr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := wr.l.With().
		Str("operation", "List").
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Str("userID", req.Filter.TenantOpts.UserID.String()).
		Logger()

	workers := make([]*worker.Worker, 0)

	q := dba.NewSelect().Model(&workers)
	q = wr.filterQuery(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan workers")
		return nil, eris.Wrap(err, "failed to scan workers")
	}

	return &ports.ListResult[*worker.Worker]{
		Items: workers,
		Total: total,
	}, nil
}

func (wr *workerRepository) GetByID(ctx context.Context, req *repositories.GetWorkerByIDRequest) (*worker.Worker, error) {
	dba, err := wr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := wr.l.With().
		Str("operation", "GetByID").
		Str("id", req.WorkerID.String()).
		Logger()

	wkr := new(worker.Worker)

	query := dba.NewSelect().Model(wkr).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("wrk.id = ?", req.WorkerID).
				Where("wrk.organization_id = ?", req.OrgID).
				Where("wrk.business_unit_id = ?", req.BuID)
		})

	query = wr.addOptions(query, req.FilterOptions)

	if err = query.Scan(ctx); err != nil {
		if eris.Is(err, sql.ErrNoRows) {
			return nil, errors.NewNotFoundError("Worker not found within your organization")
		}

		log.Error().Err(err).Msg("failed to get worker")
		return nil, eris.Wrap(err, "get worker")
	}

	return wkr, nil
}

func (wr *workerRepository) Create(ctx context.Context, wrk *worker.Worker) (*worker.Worker, error) {
	dba, err := wr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := wr.l.With().
		Str("operation", "Create").
		Str("orgID", wrk.OrganizationID.String()).
		Str("buID", wrk.BusinessUnitID.String()).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().
			Model(wrk).
			Returning("*").
			Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("worker", wrk).
				Msg("failed to insert worker")
			return eris.Wrap(iErr, "insert worker")
		}

		log.Info().
			Str("id", wrk.ID.String()).
			Msg("worker created successfully")

		// Assign the proper values to the worker profile
		wrk.Profile.WorkerID = wrk.ID
		wrk.Profile.OrganizationID = wrk.OrganizationID
		wrk.Profile.BusinessUnitID = wrk.BusinessUnitID

		// Insert the worker profile into the database
		if _, iErr := tx.NewInsert().Model(wrk.Profile).
			Returning("*").
			Exec(c); iErr != nil {
			log.Error().
				Err(iErr).
				Interface("workerProfile", wrk.Profile).
				Msg("failed to insert worker profile")
			return eris.Wrap(iErr, "insert worker profile")
		}

		// Handle PTO operations
		if err = wr.handlePTOOperations(c, tx, wrk, true); err != nil {
			return eris.Wrap(err, "handle PTO operations")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return wrk, nil
}

func (wr *workerRepository) Update(ctx context.Context, wkr *worker.Worker) (*worker.Worker, error) {
	dba, err := wr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	log := wr.l.With().
		Str("operation", "Update").
		Str("id", wkr.GetID()).
		Int64("version", wkr.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := wkr.Version

		wkr.Version++

		results, rErr := tx.NewUpdate().
			Model(wkr).
			Where("wrk.id = ?", wkr.ID).
			Where("wrk.organization_id = ?", wkr.OrganizationID).
			Where("wrk.business_unit_id = ?", wkr.BusinessUnitID).
			Where("wrk.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("worker", wkr).
				Msg("failed to update worker")
			return eris.Wrap(rErr, "update worker")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("worker", wkr).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The worker (%s) has either been updated or deleted since the last request.", wkr.ID.String()),
			)
		}

		// Update the worker profile
		if wkr.Profile != nil {
			if err = wr.updateProfile(c, wkr.Profile); err != nil {
				return err
			}
		}

		// Handle PTO operations
		if len(wkr.PTO) > 0 {
			if err = wr.handlePTOOperations(c, tx, wkr, false); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, eris.Wrap(err, "update worker")
	}

	return wkr, nil
}

func (wr *workerRepository) updateProfile(ctx context.Context, profile *worker.WorkerProfile) error {
	dba, err := wr.db.DB(ctx)
	if err != nil {
		return eris.Wrap(err, "get database connection")
	}

	log := wr.l.With().
		Str("operation", "UpdateProfile").
		Str("id", profile.ID.String()).
		Int64("version", profile.Version).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := profile.Version

		profile.Version++

		results, rErr := tx.NewUpdate().Model(profile).
			Where("wp.id = ?", profile.ID).
			Where("wp.worker_id = ?", profile.WorkerID).
			Where("wp.organization_id = ?", profile.OrganizationID).
			Where("wp.business_unit_id = ?", profile.BusinessUnitID).
			Where("wp.version = ?", ov).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error().
				Err(rErr).
				Interface("workerProfile", profile).
				Msg("failed to update worker profile")
			return eris.Wrap(rErr, "update worker profile")
		}

		rows, roErr := results.RowsAffected()
		if roErr != nil {
			log.Error().
				Err(roErr).
				Interface("workerProfile", profile).
				Msg("failed to get rows affected")
			return eris.Wrap(roErr, "get rows affected")
		}

		if rows == 0 {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				fmt.Sprintf("Version mismatch. The worker profile (%s) has either been updated or deleted since the last request.", profile.ID.String()),
			)
		}

		return nil
	})
	if err != nil {
		return eris.Wrap(err, "update worker profile")
	}

	return nil
}

func (wr *workerRepository) handlePTOOperations( //nolint:funlen,gocognit,cyclop // I know it's long but it's a lot of logic
	ctx context.Context,
	tx bun.Tx,
	wkr *worker.Worker,
	isCreate bool,
) error {
	log := wr.l.With().
		Str("operation", "handlePTOOperations").
		Str("workerID", wkr.ID.String()).
		Logger()

	// If there are no PTOs and it's a create operation, we can return early
	if len(wkr.PTO) == 0 && isCreate {
		return nil
	}

	// Get existing PTOs for comparison if this is an update
	var existingPTOs []*worker.WorkerPTO
	if !isCreate {
		if err := tx.NewSelect().
			Model(&existingPTOs).
			Where("wpto.worker_id = ?", wkr.ID).
			Where("wpto.organization_id = ?", wkr.OrganizationID).
			Where("wpto.business_unit_id = ?", wkr.BusinessUnitID).
			Scan(ctx); err != nil {
			log.Error().Err(err).Msg("failed to fetch existing PTOs")
			return eris.Wrap(err, "fetch existing PTOs")
		}
	}

	// Prepare PTOs for operations
	newPTOs := make([]*worker.WorkerPTO, 0)
	updatePTOs := make([]*worker.WorkerPTO, 0)
	existingPTOMap := make(map[pulid.ID]*worker.WorkerPTO)
	updatedPTOIDs := make(map[pulid.ID]struct{})

	// Create map of existing PTOs for quick lookup
	for _, pto := range existingPTOs {
		existingPTOMap[pto.ID] = pto
	}

	// Categorize PTOs for different operations
	for _, pto := range wkr.PTO {
		// Set required fields
		pto.WorkerID = wkr.ID
		pto.OrganizationID = wkr.OrganizationID
		pto.BusinessUnitID = wkr.BusinessUnitID

		if isCreate || pto.ID.IsNil() {
			newPTOs = append(newPTOs, pto)
		} else {
			if existing, ok := existingPTOMap[pto.ID]; ok {
				// Increment version for optimistic locking
				pto.Version = existing.Version + 1
				updatePTOs = append(updatePTOs, pto)
				updatedPTOIDs[pto.ID] = struct{}{}
			}
		}
	}

	// Handle bulk insert of new PTOs
	if len(newPTOs) > 0 {
		if _, err := tx.NewInsert().Model(&newPTOs).Exec(ctx); err != nil {
			log.Error().Err(err).Msg("failed to bulk insert new PTOs")
			return eris.Wrap(err, "bulk insert PTOs")
		}
		log.Debug().Int("count", len(newPTOs)).Msg("bulk inserted new PTOs")
	}

	// Handle bulk update of existing PTOs
	if len(updatePTOs) > 0 {
		values := tx.NewValues(&updatePTOs)
		res, err := tx.NewUpdate().
			With("_data", values).
			Model((*worker.WorkerPTO)(nil)).
			TableExpr("_data").
			Set("start_date = _data.start_date").
			Set("end_date = _data.end_date").
			Set("reason = _data.reason").
			Set("status = _data.status").
			Set("type = _data.type").
			Set("version = _data.version").
			Where("wpto.id = _data.id").
			Where("wpto.version = _data.version - 1").
			Where("wpto.worker_id = _data.worker_id").
			Where("wpto.organization_id = _data.organization_id").
			Where("wpto.business_unit_id = _data.business_unit_id").Exec(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to bulk update PTOs")
			return eris.Wrap(err, "bulk update PTOs")
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			log.Error().Err(err).Msg("failed to get rows affected for PTO updates")
			return eris.Wrap(err, "get rows affected for PTO updates")
		}

		if int(rowsAffected) != len(updatePTOs) {
			return errors.NewValidationError(
				"version",
				errors.ErrVersionMismatch,
				"One or more PTOs have been modified since last retrieval",
			)
		}
		log.Debug().Int("count", len(updatePTOs)).Msg("bulk updated PTOs")
	}

	// Handle deletion of PTOs that are no longer present
	if !isCreate {
		ptosToDelete := make([]*worker.WorkerPTO, 0)
		for id, pto := range existingPTOMap {
			if _, exists := updatedPTOIDs[id]; !exists {
				ptosToDelete = append(ptosToDelete, pto)
			}
		}

		if len(ptosToDelete) > 0 {
			_, err := tx.NewDelete().
				Model(&ptosToDelete).
				WherePK().
				Exec(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to bulk delete PTOs")
				return eris.Wrap(err, "bulk delete PTOs")
			}
			log.Debug().Int("count", len(ptosToDelete)).Msg("bulk deleted PTOs")
		}
	}

	return nil
}

func (wr *workerRepository) GetWorkerPTO(ctx context.Context, ptoID, workerID, buID, orgID pulid.ID) (*worker.WorkerPTO, error) {
	dba, err := wr.db.DB(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get database connection")
	}

	pto := new(worker.WorkerPTO)
	err = dba.NewSelect().Model(pto).
		Where("wpto.id = ?", ptoID).
		Where("wpto.worker_id = ?", workerID).
		Where("wpto.business_unit_id = ?", buID).
		Where("wpto.organization_id = ?", orgID).
		Scan(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "get worker PTO")
	}

	return pto, nil
}
