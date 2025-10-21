package workerrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

type ptoCategorizationResult struct {
	newPTOs       []*worker.WorkerPTO
	updatePTOs    []*worker.WorkerPTO
	updatedPTOIDs map[pulid.ID]struct{}
}

func NewRepository(p Params) repositories.WorkerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.worker-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListWorkerRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"wrk",
		req.Filter,
		(*worker.Worker)(nil),
	)

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListWorkerRequest,
) (*pagination.ListResult[*worker.Worker], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	workers := make([]*worker.Worker, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&workers).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan workers", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*worker.Worker]{
		Items: workers,
		Total: total,
	}, nil
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.WorkerFilterOptions,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "addOptions"),
		zap.Any("opts", opts),
	)

	if opts.IncludeProfile {
		q = q.Relation("Profile")
	}

	if opts.IncludePTO {
		q = q.Relation("PTO", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Order("start_date ASC")
		})
	}

	if opts.Status != "" {
		status, err := domain.StatusFromString(opts.Status)
		if err != nil {
			log.Error("invalid status",
				zap.Error(err),
				zap.String("status", opts.Status),
			)
			return q
		}

		q = q.Where("wrk.status = ?", status)
	}

	return q.Order("wrk.created_at DESC", "wrk.first_name ASC", "wrk.last_name ASC")
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetWorkerByIDRequest,
) (*worker.Worker, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	wkr := new(worker.Worker)

	err = db.NewSelect().
		Model(wkr).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where("wrk.id = ?", req.WorkerID).
				Where("wrk.organization_id = ?", req.OrgID).
				Where("wrk.business_unit_id = ?", req.BuID)
		}).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, req.FilterOptions)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Worker")
	}

	return wkr, nil
}

func (r *repository) Create(
	ctx context.Context,
	wrk *worker.Worker,
) (*worker.Worker, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", wrk.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, iErr := tx.NewInsert().
			Model(wrk).
			Returning("*").
			Exec(c); iErr != nil {
			log.Error("failed to insert worker", zap.Error(iErr))
			return iErr
		}

		wrk.Profile.WorkerID = wrk.ID
		wrk.Profile.OrganizationID = wrk.OrganizationID
		wrk.Profile.BusinessUnitID = wrk.BusinessUnitID

		if _, iErr := tx.NewInsert().Model(wrk.Profile).
			Returning("*").
			Exec(c); iErr != nil {
			log.Error("failed to insert worker profile", zap.Error(iErr))
			return iErr
		}

		if err = r.handlePTOOperations(c, tx, wrk, true); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return wrk, nil
}

func (r *repository) Update(
	ctx context.Context,
	wkr *worker.Worker,
) (*worker.Worker, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", wkr.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
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
			log.Error("failed to update worker", zap.Error(rErr))
			return rErr
		}

		roErr := dberror.CheckRowsAffected(results, "Worker", wkr.ID.String())
		if roErr != nil {
			return roErr
		}

		if wkr.Profile != nil {
			if err = r.updateProfile(c, wkr.Profile); err != nil {
				return err
			}
		}

		if len(wkr.PTO) > 0 {
			if err = r.handlePTOOperations(c, tx, wkr, false); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return wkr, nil
}

func (r *repository) updateProfile(
	ctx context.Context,
	profile *worker.WorkerProfile,
) error {
	log := r.l.With(
		zap.String("operation", "UpdateProfile"),
		zap.String("id", profile.ID.String()),
		zap.Int64("version", profile.Version),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := profile.Version

		profile.Version++

		results, rErr := tx.NewUpdate().Model(profile).
			WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
				return q.
					Where("wp.id = ?", profile.ID).
					Where("wp.worker_id = ?", profile.WorkerID).
					Where("wp.organization_id = ?", profile.OrganizationID).
					Where("wp.business_unit_id = ?", profile.BusinessUnitID).
					Where("wp.version = ?", ov)
			}).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error("failed to update worker profile", zap.Error(rErr))
			return rErr
		}

		roErr := dberror.CheckRowsAffected(results, "Worker Profile", profile.ID.String())
		if roErr != nil {
			return roErr
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) handlePTOOperations(
	ctx context.Context,
	tx bun.Tx,
	wkr *worker.Worker,
	isCreate bool,
) error {
	log := r.l.With(
		zap.String("operation", "HandlePTOOperations"),
		zap.String("workerID", wkr.ID.String()),
	)

	if len(wkr.PTO) == 0 && isCreate {
		return nil
	}

	existingPTOs, err := r.fetchExistingPTOs(ctx, tx, wkr, isCreate)
	if err != nil {
		return err
	}

	categorized := r.categorizePTOs(wkr, existingPTOs, isCreate)

	if err = r.bulkInsertPTOs(ctx, tx, categorized.newPTOs, log); err != nil {
		return err
	}

	if err = r.bulkUpdatePTOs(ctx, tx, categorized.updatePTOs, log); err != nil {
		return err
	}

	if !isCreate {
		if deleteErr := r.deleteMissingPTOs(ctx, tx, wkr.ID, existingPTOs, categorized.updatedPTOIDs, log); deleteErr != nil {
			return deleteErr
		}
	}

	return nil
}

func (r *repository) fetchExistingPTOs(
	ctx context.Context,
	tx bun.Tx,
	wkr *worker.Worker,
	isCreate bool,
) ([]*worker.WorkerPTO, error) {
	if isCreate {
		return []*worker.WorkerPTO{}, nil
	}

	existingPTOs := make([]*worker.WorkerPTO, 0)
	if err := tx.NewSelect().
		Model(&existingPTOs).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where("wpto.worker_id = ?", wkr.ID).
				Where("wpto.organization_id = ?", wkr.OrganizationID).
				Where("wpto.business_unit_id = ?", wkr.BusinessUnitID)
		}).
		Scan(ctx); err != nil {
		r.l.Error("failed to fetch existing PTOs", zap.Error(err))
		return nil, err
	}

	return existingPTOs, nil
}

func (r *repository) categorizePTOs(
	wkr *worker.Worker,
	existingPTOs []*worker.WorkerPTO,
	isCreate bool,
) *ptoCategorizationResult {
	ptoCount := len(wkr.PTO)
	result := &ptoCategorizationResult{
		newPTOs:       make([]*worker.WorkerPTO, 0, ptoCount),
		updatePTOs:    make([]*worker.WorkerPTO, 0, ptoCount),
		updatedPTOIDs: make(map[pulid.ID]struct{}, ptoCount),
	}

	existingPTOMap := make(map[pulid.ID]*worker.WorkerPTO, len(existingPTOs))
	for _, pto := range existingPTOs {
		existingPTOMap[pto.ID] = pto
	}

	for _, pto := range wkr.PTO {
		pto.WorkerID = wkr.ID
		pto.OrganizationID = wkr.OrganizationID
		pto.BusinessUnitID = wkr.BusinessUnitID

		if isCreate || pto.ID.IsNil() {
			result.newPTOs = append(result.newPTOs, pto)
		} else {
			if existing, ok := existingPTOMap[pto.ID]; ok {
				pto.Version = existing.Version + 1
				result.updatePTOs = append(result.updatePTOs, pto)
				result.updatedPTOIDs[pto.ID] = struct{}{}
			}
			// ! Note: PTOs with IDs not found in existing are silently ignored
		}
	}

	return result
}

func (r *repository) bulkInsertPTOs(
	ctx context.Context,
	tx bun.Tx,
	newPTOs []*worker.WorkerPTO,
	log *zap.Logger,
) error {
	if len(newPTOs) == 0 {
		return nil
	}

	if _, err := tx.NewInsert().Model(&newPTOs).Exec(ctx); err != nil {
		log.Error("failed to bulk insert new PTOs", zap.Error(err))
		return err
	}

	log.Debug("successfully inserted new PTOs", zap.Int("count", len(newPTOs)))
	return nil
}

func (r *repository) bulkUpdatePTOs(
	ctx context.Context,
	tx bun.Tx,
	updatePTOs []*worker.WorkerPTO,
	log *zap.Logger,
) error {
	if len(updatePTOs) == 0 {
		return nil
	}

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
		Set("updated_at = ?", utils.NowUnix()).
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				Where("wpto.id = _data.id").
				Where("wpto.version = _data.version - 1"). // Optimistic locking check
				Where("wpto.worker_id = _data.worker_id").
				Where("wpto.organization_id = _data.organization_id").
				Where("wpto.business_unit_id = _data.business_unit_id")
		}).
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update PTOs", zap.Error(err))
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected for PTO updates", zap.Error(err))
		return err
	}

	if int(rowsAffected) != len(updatePTOs) {
		log.Warn("version mismatch detected",
			zap.Int("expected", len(updatePTOs)),
			zap.Int64("actual", rowsAffected))
		return errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"One or more PTOs have been modified since last retrieval",
		)
	}

	log.Debug("successfully updated PTOs", zap.Int("count", len(updatePTOs)))
	return nil
}

func (r *repository) deleteMissingPTOs(
	ctx context.Context,
	tx bun.Tx,
	workerID pulid.ID,
	existingPTOs []*worker.WorkerPTO,
	updatedPTOIDs map[pulid.ID]struct{},
	log *zap.Logger,
) error {
	existingPTOMap := make(map[pulid.ID]*worker.WorkerPTO, len(existingPTOs))
	for _, pto := range existingPTOs {
		existingPTOMap[pto.ID] = pto
	}

	ptoIDsToDelete := make([]pulid.ID, 0, len(existingPTOs))
	for id := range existingPTOMap {
		if _, exists := updatedPTOIDs[id]; !exists {
			ptoIDsToDelete = append(ptoIDsToDelete, id)
		}
	}

	if len(ptoIDsToDelete) == 0 {
		return nil
	}

	_, err := tx.NewDelete().
		Model((*worker.WorkerPTO)(nil)).
		Where("id IN (?)", bun.In(ptoIDsToDelete)).
		Where("worker_id = ?", workerID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk delete PTOs", zap.Error(err))
		return err
	}

	log.Debug("successfully deleted PTOs", zap.Int("count", len(ptoIDsToDelete)))
	return nil
}

func (r *repository) filterWorkerPTOQuery(
	q *bun.SelectQuery,
	req *repositories.ListWorkerPTORequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"wpto",
		req.Filter,
		(*worker.WorkerPTO)(nil),
	)

	q = q.Relation("Worker")

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) ListWorkerPTO(
	ctx context.Context,
	req *repositories.ListWorkerPTORequest,
) (*pagination.ListResult[*worker.WorkerPTO], error) {
	log := r.l.With(
		zap.String("operation", "ListWorkerPTO"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*worker.WorkerPTO, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterWorkerPTOQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan worker PTOs", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*worker.WorkerPTO]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetWorkerPTO(
	ctx context.Context,
	req *repositories.GetWorkerPTORequest,
) (*worker.WorkerPTO, error) {
	log := r.l.With(
		zap.String("operation", "GetWorkerPTO"),
		zap.String("ptoId", req.PtoID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	pto := new(worker.WorkerPTO)
	err = db.NewSelect().Model(pto).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wpto.id = ?", req.PtoID).
				Where("wpto.worker_id = ?", req.WorkerID).
				Where("wpto.business_unit_id = ?", req.BuID).
				Where("wpto.organization_id = ?", req.OrgID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get worker PTO", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "Worker PTO")
	}

	return pto, nil
}

func (r *repository) addUpcomingPTOOptions(
	q *bun.SelectQuery,
	opts *repositories.ListUpcomingWorkerPTORequest,
) *bun.SelectQuery {
	log := r.l.With(
		zap.String("operation", "addUpcomingPTOOptions"),
		zap.Any("opts", opts),
	)

	if opts.StartDate > 0 {
		q = q.Where("wpto.start_date >= ?", opts.StartDate)
	}

	if opts.EndDate > 0 {
		q = q.Where("wpto.end_date <= ?", opts.EndDate)
	}

	if opts.Type != "" {
		ptoType, err := worker.PTOTypeFromString(opts.Type)
		if err != nil {
			log.Error("failed to convert type to PTO type", zap.Error(err))
			return q
		}

		q = q.Where("wpto.type = ?", ptoType)
	}

	if opts.Status != "" {
		ptoStatus, err := worker.PTOStatusFromString(opts.Status)
		if err != nil {
			log.Error("failed to convert status to PTO status", zap.Error(err))
			return q
		}

		q = q.Where("wpto.status = ?", ptoStatus)
	}

	if opts.WorkerID != "" {
		wrkID, err := pulid.MustParse(opts.WorkerID)
		if err != nil {
			log.Error("failed to parse worker ID", zap.Error(err))
			return q
		}

		q = q.Where("wpto.worker_id = ?", wrkID)
	}

	q = q.Order("wpto.start_date ASC", "wpto.end_date ASC")
	q = q.RelationWithOpts("Worker", bun.RelationOpts{
		Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
			if opts.FleetCodeID != "" {
				fleetCodeID, err := pulid.MustParse(opts.FleetCodeID)
				if err != nil {
					log.Error("failed to parse fleet code ID", zap.Error(err))
					return q
				}
				sq = sq.Where("wrk.fleet_code_id = ?", fleetCodeID)
			}

			return sq
		},
	})

	return q.Limit(opts.Filter.Limit).Offset(opts.Filter.Offset)
}

func (r *repository) ListUpcomingPTO(
	ctx context.Context,
	req *repositories.ListUpcomingWorkerPTORequest,
) (*pagination.ListResult[*worker.WorkerPTO], error) {
	log := r.l.With(
		zap.String("operation", "ListUpcomingPTO"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*worker.WorkerPTO, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addUpcomingPTOOptions(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan upcoming PTOs", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*worker.WorkerPTO]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) ApprovePTO(
	ctx context.Context,
	req *repositories.ApprovePTORequest,
) error {
	log := r.l.With(
		zap.String("operation", "ApprovePTO"),
		zap.String("ptoId", req.PtoID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
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
			Exec(c)
		if err != nil {
			log.Error("failed to approve PTO", zap.Error(err))
			return err
		}

		return nil
	})

	return nil
}

func (r *repository) RejectPTO(
	ctx context.Context,
	req *repositories.RejectPTORequest,
) error {
	log := r.l.With(
		zap.String("operation", "RejectPTO"),
		zap.String("ptoId", req.PtoID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
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
			Exec(c)
		if err != nil {
			log.Error("failed to reject PTO", zap.Error(err))
			return err
		}

		return nil
	})

	return nil
}

func (r *repository) CreateWorkerPTO(
	ctx context.Context,
	pto *worker.WorkerPTO,
) (*worker.WorkerPTO, error) {
	log := r.l.With(
		zap.String("operation", "CreateWorkerPTO"),
		zap.String("ptoId", pto.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	_, err = db.NewInsert().Model(pto).Exec(ctx)
	if err != nil {
		log.Error("failed to create worker PTO", zap.Error(err))
		return nil, err
	}

	return pto, nil
}
