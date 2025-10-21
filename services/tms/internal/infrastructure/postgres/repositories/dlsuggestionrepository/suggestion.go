package dlsuggestionrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
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

func NewRepository(p Params) repositories.DedicatedLaneSuggestionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.dedicatedlane-suggesstion-repository"),
	}
}

func (r *repository) addOptions(
	q *bun.SelectQuery,
	req *repositories.ListDedicatedLaneSuggestionRequest,
) *bun.SelectQuery {
	if req.Status != nil {
		q = q.Where("dls.status = ?", *req.Status)
	}

	if req.CustomerID != nil && !req.CustomerID.IsNil() {
		q = q.Where("dls.customer_id = ?", req.CustomerID.String())
	}

	if !req.IncludeExpired {
		now := utils.NowUnix()
		q = q.Where(
			"dls.expires_at > ? OR dls.status != ?",
			now,
			dedicatedlane.SuggestionStatusPending,
		)
	}

	if !req.IncludeProcessed {
		q = q.Where("dls.status = ?", dedicatedlane.SuggestionStatusPending)
	}

	relations := []string{
		"Customer",
		"OriginLocation",
		"OriginLocation.State",
		"DestinationLocation",
		"DestinationLocation.State",
	}

	for _, rel := range relations {
		q = q.Relation(rel)
	}

	q = q.Order("dls.confidence_score DESC", "dls.created_at DESC")

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListDedicatedLaneSuggestionRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"dls",
		req.Filter,
		(*dedicatedlane.Suggestion)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req)
	})

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListDedicatedLaneSuggestionRequest,
) (*pagination.ListResult[*dedicatedlane.Suggestion], error) {
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

	entities := make([]*dedicatedlane.Suggestion, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan dedicated lanes", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*dedicatedlane.Suggestion]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetDedicatedLaneSuggestionByIDRequest,
) (*dedicatedlane.Suggestion, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(dedicatedlane.Suggestion)
	err = db.NewSelect().Model(entity).
		Relation("Customer").
		Relation("OriginLocation").
		Relation("OriginLocation.State").
		Relation("DestinationLocation").
		Relation("DestinationLocation.State").
		Relation("ServiceType").
		Relation("ShipmentType").
		Relation("TrailerType").
		Relation("TractorType").
		Relation("CreatedDedicatedLane").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("dls.id = ?", req.ID).
				Where("dls.organization_id = ?", req.OrgID).
				Where("dls.business_unit_id = ?", req.BuID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Dedicated Lane Suggestion")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *dedicatedlane.Suggestion,
) (*dedicatedlane.Suggestion, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert dedicated lane suggestion", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *dedicatedlane.Suggestion,
) (*dedicatedlane.Suggestion, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("dls.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update dedicated lane suggestion", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Dedicated Lane Suggestion", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) UpdateStatus(
	ctx context.Context,
	req *repositories.UpdateSuggestionStatusRequest,
) (*dedicatedlane.Suggestion, error) {
	log := r.l.With(
		zap.String("operation", "UpdateStatus"),
		zap.String("entityID", req.SuggestionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(dedicatedlane.Suggestion)

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if err = tx.NewSelect().Model(entity).
			Where("dls.id = ?", req.SuggestionID).
			For("UPDATE").
			Scan(c); err != nil {
			return dberror.HandleNotFoundError(err, "Dedicated Lane Suggestion")
		}

		entity.Status = req.Status
		entity.ProcessedByID = req.ProcessedByID
		entity.ProcessedAt = req.ProcessedAt
		entity.Version++

		if req.RejectReason != nil && entity.PatternDetails == nil {
			entity.PatternDetails = make(map[string]any)
			entity.PatternDetails["rejectReason"] = *req.RejectReason
			entity.PatternDetails["rejectedAt"] = utils.NowUnix()
		}

		if _, err = tx.NewUpdate().Model(entity).WherePK().Where("dls.version = ?", entity.Version).Exec(c); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to update dedicated lane suggestion status", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req *repositories.DeleteDedicatedLaneSuggestionRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := db.NewDelete().
		Model((*dedicatedlane.Suggestion)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("dls.id = ?", req.ID).
				Where("dls.organization_id = ?", req.OrgID).
				Where("dls.business_unit_id = ?", req.BuID)
		}).Exec(ctx)
	if err != nil {
		log.Error("failed to delete dedicated lane suggestion", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "Dedicated Lane Suggestion", req.ID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) ExpireOldSuggestions(
	ctx context.Context,
	orgID, buID pulid.ID,
) (int64, error) {
	log := r.l.With(
		zap.String("operation", "ExpireOldSuggestions"),
		zap.String("orgID", orgID.String()),
		zap.String("buID", buID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return 0, err
	}

	now := utils.NowUnix()

	result, err := db.NewUpdate().Model((*dedicatedlane.Suggestion)(nil)).
		Set("status = ?", dedicatedlane.SuggestionStatusExpired).
		Set("updated_at = ?", now).
		WhereGroup(" AND ", func(sq *bun.UpdateQuery) *bun.UpdateQuery {
			return sq.Where("dls.organization_id = ?", orgID).
				Where("dls.status = ?", dedicatedlane.SuggestionStatusPending).
				Where("dls.business_unit_id = ?", buID).
				Where("dls.expires_at < ?", now)
		}).Exec(ctx)
	if err != nil {
		log.Error("failed to expire old dedicated lane suggestions", zap.Error(err))
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected", zap.Error(err))
		return 0, err
	}

	return rowsAffected, nil
}

func (r *repository) CheckForDuplicatePattern(
	ctx context.Context,
	req *repositories.FindDedicatedLaneByShipmentRequest,
) (*dedicatedlane.Suggestion, error) {
	log := r.l.With(
		zap.String("operation", "CheckForDuplicatePattern"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(dedicatedlane.Suggestion)
	err = db.NewSelect().
		Model(entity).
		Distinct().
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = sq.Where("dls.organization_id = ?", req.OrganizationID).
				Where("dls.business_unit_id = ?", req.BusinessUnitID).
				Where("dls.customer_id = ?", req.CustomerID).
				Where("dls.origin_location_id = ?", req.OriginLocationID).
				Where("dls.destination_location_id = ?", req.DestinationLocationID).
				Where("dls.status = ?", dedicatedlane.SuggestionStatusPending)

			if req.ServiceTypeID != nil && !req.ServiceTypeID.IsNil() {
				sq = sq.Where("dls.service_type_id = ?", req.ServiceTypeID.String())
			} else {
				sq = sq.Where("dls.service_type_id IS NULL")
			}

			if req.ShipmentTypeID != nil && !req.ShipmentTypeID.IsNil() {
				sq = sq.Where("dls.shipment_type_id = ?", req.ShipmentTypeID.String())
			} else {
				sq = sq.Where("dls.shipment_type_id IS NULL")
			}

			if req.TrailerTypeID != nil && !req.TrailerTypeID.IsNil() {
				sq = sq.Where("dls.trailer_type_id = ?", req.TrailerTypeID.String())
			} else {
				sq = sq.Where("dls.trailer_type_id IS NULL")
			}

			if req.TractorTypeID != nil && !req.TractorTypeID.IsNil() {
				sq = sq.Where("dls.tractor_type_id = ?", req.TractorTypeID.String())
			} else {
				sq = sq.Where("dls.tractor_type_id IS NULL")
			}

			return sq
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Dedicated Lane Suggestion")
	}

	return entity, nil
}
