/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
	"github.com/emoss08/trenova/internal/infrastructure/database/postgres/repositories/common"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
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

func (wr *workerRepository) addOptions(
	q *bun.SelectQuery,
	opts repositories.WorkerFilterOptions,
) *bun.SelectQuery {
	// * Include the profile if requested
	if opts.IncludeProfile {
		q = q.Relation("Profile")
	}

	// * Include the PTO if requested
	if opts.IncludePTO {
		q = q.Relation("PTO", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order("start_date ASC")
		})
	}

	if opts.Status != "" {
		status, err := domain.StatusFromString(opts.Status)
		if err != nil {
			wr.l.Error().Err(err).Str("status", opts.Status).Msg("invalid status")
			return q
		}

		q = q.Where("wrk.status = ?", status)
	}

	return q.Order("wrk.created_at DESC", "wrk.first_name ASC", "wrk.last_name ASC")
}

func (wr *workerRepository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListWorkerRequest,
) *bun.SelectQuery {
	qb := querybuilder.NewWithPostgresSearch(
		q,
		"wrk",
		repositories.WorkerFieldConfig,
		(*worker.Worker)(nil),
	)

	qb.ApplyTenantFilters(req.Filter.TenantOpts)

	if req.Filter != nil {
		qb.ApplyFilters(req.Filter.FieldFilters)

		if len(req.Filter.Sort) > 0 {
			qb.ApplySort(req.Filter.Sort)
		}

		if req.Filter.Query != "" {
			qb.ApplyTextSearch(req.Filter.Query, []string{"first_name", "last_name"})
		}

		q = qb.GetQuery()
	}
	q = wr.addOptions(q, req.WorkerFilterOptions)

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

// List returns a list of workers in the database
func (wr *workerRepository) List(
	ctx context.Context,
	req *repositories.ListWorkerRequest,
) (*ports.ListResult[*worker.Worker], error) {
	dba, err := wr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := wr.l.With().
		Str("operation", "List").
		Interface("req", req).
		Logger()

	workers := make([]*worker.Worker, 0, req.Filter.Limit)

	q := dba.NewSelect().Model(&workers)
	q = wr.filterQuery(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan workers")
		return nil, common.HandleNotFoundError(err, "Worker")
	}

	return &ports.ListResult[*worker.Worker]{
		Items: workers,
		Total: total,
	}, nil
}

func (wr *workerRepository) GetByID(
	ctx context.Context,
	req *repositories.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	dba, err := wr.db.ReadDB(ctx)
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

func (wr *workerRepository) Create(
	ctx context.Context,
	wrk *worker.Worker,
) (*worker.Worker, error) {
	dba, err := wr.db.WriteDB(ctx)
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

func (wr *workerRepository) Update(
	ctx context.Context,
	wkr *worker.Worker,
) (*worker.Worker, error) {
	dba, err := wr.db.WriteDB(ctx)
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
			OmitZero().
			WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
				return q.Where("wrk.id = ?", wkr.ID).
					Where("wrk.organization_id = ?", wkr.OrganizationID).
					Where("wrk.business_unit_id = ?", wkr.BusinessUnitID).
					Where("wrk.version = ?", ov)
			}).
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
				fmt.Sprintf(
					"Version mismatch. The worker (%s) has either been updated or deleted since the last request.",
					wkr.ID.String(),
				),
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

func (wr *workerRepository) updateProfile(
	ctx context.Context,
	profile *worker.WorkerProfile,
) error {
	dba, err := wr.db.ReadDB(ctx)
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
				fmt.Sprintf(
					"Version mismatch. The worker profile (%s) has either been updated or deleted since the last request.",
					profile.ID.String(),
				),
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

	existingPTOs := make([]*worker.WorkerPTO, 0)
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

	ptoCount := len(wkr.PTO)
	newPTOs := make([]*worker.WorkerPTO, 0, ptoCount)
	updatePTOs := make([]*worker.WorkerPTO, 0, ptoCount)
	updatedPTOIDs := make(map[pulid.ID]struct{}, ptoCount)
	existingPTOMap := make(map[pulid.ID]*worker.WorkerPTO, len(existingPTOs))

	for _, pto := range existingPTOs {
		existingPTOMap[pto.ID] = pto
	}

	for _, pto := range wkr.PTO {
		pto.WorkerID = wkr.ID
		pto.OrganizationID = wkr.OrganizationID
		pto.BusinessUnitID = wkr.BusinessUnitID

		if isCreate || pto.ID.IsNil() {
			newPTOs = append(newPTOs, pto)
		} else {
			if existing, ok := existingPTOMap[pto.ID]; ok {
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
		ptosToDelete := make([]*worker.WorkerPTO, 0, len(existingPTOs))
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

func (wr *workerRepository) filterWorkerPTOQuery(
	q *bun.SelectQuery,
	req *repositories.ListWorkerPTORequest,
) *bun.SelectQuery {
	qb := querybuilder.NewWithPostgresSearch(
		q,
		"wpto",
		repositories.WorkerPTOFieldConfig,
		(*worker.WorkerPTO)(nil),
	)

	qb.ApplyTenantFilters(req.Filter.TenantOpts)

	if req.Filter != nil {
		qb.ApplyFilters(req.Filter.FieldFilters)

		if len(req.Filter.Sort) > 0 {
			qb.ApplySort(req.Filter.Sort)
		}

		if req.Filter.Query != "" {
			qb.ApplyTextSearch(req.Filter.Query, []string{"type", "status"})
		}

		q = q.Relation("Worker")

		q = qb.GetQuery()
	}

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (wr *workerRepository) ListWorkerPTO(
	ctx context.Context,
	req *repositories.ListWorkerPTORequest,
) (*ports.ListResult[*worker.WorkerPTO], error) {
	dba, err := wr.db.WriteDB(ctx)
	if err != nil {
		return nil, err
	}

	log := wr.l.With().
		Str("operation", "ListWorkerPTO").
		Interface("req", req).
		Logger()

	entities := make([]*worker.WorkerPTO, 0, req.Filter.Limit)

	q := dba.NewSelect().Model(&entities)
	q = wr.filterWorkerPTOQuery(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan worker PTOs")
		return nil, common.HandleNotFoundError(err, "Worker PTO")
	}

	return &ports.ListResult[*worker.WorkerPTO]{
		Items: entities,
		Total: total,
	}, nil
}

func (wr *workerRepository) GetWorkerPTO(
	ctx context.Context,
	req *repositories.GetWorkerPTORequest,
) (*worker.WorkerPTO, error) {
	dba, err := wr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := wr.l.With().
		Str("operation", "GetWorkerPTO").
		Interface("req", req).
		Logger()

	pto := new(worker.WorkerPTO)
	err = dba.NewSelect().Model(pto).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wpto.id = ?", req.PtoID).
				Where("wpto.worker_id = ?", req.WorkerID).
				Where("wpto.business_unit_id = ?", req.BuID).
				Where("wpto.organization_id = ?", req.OrgID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get worker PTO")
		return nil, common.HandleNotFoundError(err, "Worker PTO")
	}

	return pto, nil
}

func (wr *workerRepository) addUpcomingPTOOptions(
	q *bun.SelectQuery,
	opts *repositories.ListUpcomingWorkerPTORequest,
) *bun.SelectQuery {
	if opts.StartDate > 0 {
		q = q.Where("wpto.start_date >= ?", opts.StartDate)
	}

	if opts.EndDate > 0 {
		q = q.Where("wpto.end_date <= ?", opts.EndDate)
	}

	if opts.Type != "" {
		ptoType, err := worker.PTOTypeFromString(opts.Type)
		if err != nil {
			wr.l.Error().
				Err(err).
				Str("type", opts.Type).
				Msg("failed to convert type to PTO type")
			return q
		}

		q = q.Where("wpto.type = ?", ptoType)
	}

	if opts.Status != "" {
		ptoStatus, err := worker.PTOStatusFromString(opts.Status)
		if err != nil {
			wr.l.Error().
				Err(err).
				Str("status", opts.Status).
				Msg("failed to convert status to PTO status")
			return q
		}

		q = q.Where("wpto.status = ?", ptoStatus)
	}

	q = q.Order("wpto.start_date ASC", "wpto.end_date ASC")
	q = q.Relation("Worker")

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (wr *workerRepository) ListUpcomingPTO(
	ctx context.Context,
	req *repositories.ListUpcomingWorkerPTORequest,
) (*ports.ListResult[*worker.WorkerPTO], error) {
	dba, err := wr.db.ReadDB(ctx)
	if err != nil {
		return nil, err
	}

	log := wr.l.With().
		Str("operation", "ListUpcomingPTO").
		Interface("req", req).
		Logger()

	entities := make([]*worker.WorkerPTO, 0, req.Filter.Limit)

	q := dba.NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wpto.organization_id = ?", req.Filter.TenantOpts.OrgID).
				Where("wpto.business_unit_id = ?", req.Filter.TenantOpts.BuID)
		})

	q = wr.addUpcomingPTOOptions(q, req)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get upcoming PTOs")
		return nil, err
	}

	return &ports.ListResult[*worker.WorkerPTO]{
		Items: entities,
		Total: total,
	}, nil
}

func (wr *workerRepository) ApprovePTO(
	ctx context.Context,
	req *repositories.ApprovePTORequest,
) error {
	dba, err := wr.db.WriteDB(ctx)
	if err != nil {
		return err
	}

	log := wr.l.With().
		Str("operation", "ApprovePTO").
		Interface("req", req).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		_, err = tx.NewUpdate().
			Model((*worker.WorkerPTO)(nil)).
			Set("status = ?", worker.PTOStatusApproved).
			Set("approver_id = ?", req.ApproverID).
			Set("version = version + 1").
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("wpto.id = ?", req.PtoID).
					Where("wpto.business_unit_id = ?", req.BuID).
					Where("wpto.organization_id = ?", req.OrgID).
					Where("wpto.status = ?", worker.PTOStatusRequested)
			}).
			Exec(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to approve PTO")
			return err
		}

		return nil
	})

	return nil
}

func (wr *workerRepository) RejectPTO(
	ctx context.Context,
	req *repositories.RejectPTORequest,
) error {
	dba, err := wr.db.WriteDB(ctx)
	if err != nil {
		return err
	}

	log := wr.l.With().
		Str("operation", "RejectPTO").
		Interface("req", req).
		Logger()

	err = dba.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		_, err = tx.NewUpdate().
			Model((*worker.WorkerPTO)(nil)).
			Set("status = ?", worker.PTOStatusRejected).
			Set("rejector_id = ?", req.RejectorID).
			Set("reason = ?", req.Reason).
			Set("version = version + 1").
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("wpto.id = ?", req.PtoID).
					Where("wpto.business_unit_id = ?", req.BuID).
					Where("wpto.organization_id = ?", req.OrgID).
					Where("wpto.status = ?", worker.PTOStatusRequested)
			}).
			Exec(ctx)
		if err != nil {
			log.Error().Err(err).Msg("failed to reject PTO")
			return err
		}

		return nil
	})

	return nil
}
