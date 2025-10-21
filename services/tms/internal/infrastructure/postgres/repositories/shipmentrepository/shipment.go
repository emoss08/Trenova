package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/calculator"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB                         *postgres.Connection
	Logger                     *zap.Logger
	Generator                  seqgen.Generator
	Calculator                 *calculator.ShipmentCalculator
	MoveRepository             repositories.ShipmentMoveRepository
	AdditionalChargeRepository repositories.AdditionalChargeRepository
}

type repository struct {
	db                         *postgres.Connection
	l                          *zap.Logger
	generator                  seqgen.Generator
	calculator                 *calculator.ShipmentCalculator
	moveRepository             repositories.ShipmentMoveRepository
	additionalChargeRepository repositories.AdditionalChargeRepository
}

func NewRepository(p Params) repositories.ShipmentRepository {
	return &repository{
		db:                         p.DB,
		l:                          p.Logger.Named("postgres.shipment-repository"),
		generator:                  p.Generator,
		calculator:                 p.Calculator,
		moveRepository:             p.MoveRepository,
		additionalChargeRepository: p.AdditionalChargeRepository,
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListShipmentRequest,
) (*pagination.ListResult[*shipment.Shipment], error) {
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

	entities := make([]*shipment.Shipment, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan shipments", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*shipment.Shipment]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByOrgID(
	ctx context.Context,
	orgID pulid.ID,
) (*pagination.ListResult[*shipment.Shipment], error) {
	log := r.l.With(
		zap.String("operation", "GetByOrgID"),
		zap.String("orgID", orgID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*shipment.Shipment, 0)

	total, err := db.NewSelect().Model(&entities).
		Where("organization_id = ?", orgID).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.addOptions(sq, repositories.ShipmentOptions{
				ExpandShipmentDetails: true,
			})
		}).
		ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan shipments", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*shipment.Shipment]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if err = r.prepareForCreation(ctx, entity, userID); err != nil {
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if _, err = tx.NewInsert().Model(entity).Returning("*").Exec(c); err != nil {
			log.Error("failed to insert shipment", zap.Error(err))
			return err
		}

		if err = r.moveRepository.HandleMoveOperations(c, tx, entity, true); err != nil {
			log.Error("failed to handle move operations", zap.Error(err))
			return err
		}

		if err = r.additionalChargeRepository.HandleOperations(c, tx, entity, true); err != nil {
			log.Error("failed to handle additional charge operations", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to run in transaction", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *shipment.Shipment,
	userID pulid.ID,
) (*shipment.Shipment, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
		zap.String("userID", userID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		ov := entity.Version
		entity.Version++

		results, rErr := tx.NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
				return q.
					Where("sp.id = ?", entity.ID).
					Where("sp.organization_id = ?", entity.OrganizationID).
					Where("sp.business_unit_id = ?", entity.BusinessUnitID).
					Where("sp.version = ?", ov)
			}).
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error("failed to update shipment", zap.Error(rErr))
			return rErr
		}

		roErr := dberror.CheckRowsAffected(results, "Shipment", entity.ID.String())
		if roErr != nil {
			return roErr
		}

		if err = r.moveRepository.HandleMoveOperations(c, tx, entity, false); err != nil {
			log.Error("failed to handle move operations", zap.Error(err))
			return err
		}

		if err = r.additionalChargeRepository.HandleOperations(c, tx, entity, false); err != nil {
			log.Error("failed to handle additional charge operations", zap.Error(err))
			return err
		}

		return nil
	})

	return entity, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req *repositories.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("entityID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(shipment.Shipment)
	q := db.NewSelect().Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("sp.id = ?", req.ID).
				Where("sp.organization_id = ?", req.OrgID).
				Where("sp.business_unit_id = ?", req.BuID)
		})

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.ShipmentOptions)
	})

	if err = q.Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Shipment")
	}

	return entity, nil
}

func (r *repository) BulkDuplicate(
	ctx context.Context,
	req *repositories.DuplicateShipmentRequest,
) ([]*shipment.Shipment, error) {
	log := r.l.With(
		zap.String("operation", "BulkDuplicate"),
		zap.String("shipmentID", req.ShipmentID.String()),
		zap.Int("count", req.Count),
		zap.Bool("overrideDates", req.OverrideDates),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	original, err := r.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:    req.ShipmentID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		log.Error("failed to get original shipment", zap.Error(err))
		return nil, err
	}

	data, err := r.prepareBulkShipmentData(ctx, original, req)
	if err != nil {
		log.Error("failed to prepare bulk shipment data", zap.Error(err))
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		if err = r.bulkInsertShipmentData(c, tx, data, log); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to run in transaction", zap.Error(err))
		return nil, err
	}

	return data.shipments, nil
}

func (r *repository) GetPreviousRates(
	ctx context.Context,
	req *repositories.GetPreviousRatesRequest,
) (*pagination.ListResult[*shipment.Shipment], error) {
	log := r.l.With(
		zap.String("operation", "GetPreviousRates"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*shipment.Shipment, 0, 50)

	originCTE, destCTE := r.buildPreviousRatesCTEs(
		db,
		req.OriginLocationID,
		req.DestinationLocationID,
	)

	q := db.NewSelect().
		Model(&entities).
		With("origin_shipments", originCTE).
		With("dest_shipments", destCTE).
		Relation("ShipmentType").
		Relation("ServiceType").
		Relation("Customer").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sp.organization_id = ?", req.OrgID).
				Where("sp.business_unit_id = ?", req.BuID).
				Where("sp.shipment_type_id = ?", req.ShipmentTypeID).
				Where("sp.service_type_id = ?", req.ServiceTypeID).
				Where("sp.status = ?", shipment.StatusBilled).
				Where("sp.id IN (SELECT shipment_id FROM origin_shipments)").
				Where("sp.id IN (SELECT shipment_id FROM dest_shipments)")
		})

	if req.CustomerID != nil {
		q = q.Where("sp.customer_id = ?", req.CustomerID)
	}

	if req.ExcludeShipmentID != nil {
		q = q.Where("sp.id != ?", req.ExcludeShipmentID)
	}

	q = q.Order("sp.created_at DESC")

	q = q.Limit(50)

	total, err := q.ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count previous rates", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*shipment.Shipment]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) CalculateTotals(
	ctx context.Context,
	shp *shipment.Shipment,
	userID pulid.ID,
) (*repositories.ShipmentTotalsResponse, error) {
	r.calculator.CalculateTotals(ctx, shp, userID)

	baseCharge := r.calculator.CalculateBaseCharge(ctx, shp, userID)
	otherCharge := decimal.Zero
	if shp.OtherChargeAmount.Valid {
		otherCharge = shp.OtherChargeAmount.Decimal
	}

	total := decimal.Zero
	if shp.TotalChargeAmount.Valid {
		total = shp.TotalChargeAmount.Decimal
	}

	return &repositories.ShipmentTotalsResponse{
		BaseCharge:        baseCharge,
		OtherChargeAmount: otherCharge,
		TotalChargeAmount: total,
	}, nil
}

func (r *repository) Cancel(
	ctx context.Context,
	req *repositories.CancelShipmentRequest,
) (*shipment.Shipment, error) {
	log := r.l.With(
		zap.String("operation", "Cancel"),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(shipment.Shipment)

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		results, rErr := tx.NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("sp.id = ?", req.ShipmentID).
					Where("sp.organization_id = ?", req.OrgID).
					Where("sp.business_unit_id = ?", req.BuID)
			}).
			Set("status = ?", shipment.StatusCanceled).
			Set("canceled_at = ?", req.CanceledAt).
			Set("canceled_by_id = ?", req.CanceledByID).
			Set("cancel_reason = ?", req.CancelReason).
			Set("version = version + 1").
			Returning("*").
			Exec(c)
		if rErr != nil {
			log.Error("failed to update shipment", zap.Error(rErr))
			return rErr
		}

		roErr := dberror.CheckRowsAffected(results, "Shipment", req.ShipmentID.String())
		if roErr != nil {
			return roErr
		}

		if err = r.cancelShipmentComponents(c, tx, req); err != nil {
			log.Error("failed to cancel shipment components", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		log.Error("failed to run in transaction", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) UnCancel(
	ctx context.Context,
	req *repositories.UnCancelShipmentRequest,
) (*shipment.Shipment, error) {
	log := r.l.With(
		zap.String("operation", "UnCancel"),
		zap.String("shipmentID", req.ShipmentID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	entity := new(shipment.Shipment)
	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		results, rErr := tx.NewUpdate().
			Model(entity).
			Set("status = ?", shipment.StatusNew).
			Set("canceled_at = ?", nil).
			Set("canceled_by_id = ?", pulid.Nil).
			Set("cancel_reason = ?", nil).
			Set("version = version + 1").
			OmitZero().
			WhereGroup(" AND", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where("sp.id = ?", req.ShipmentID).
					Where("sp.organization_id = ?", req.OrgID).
					Where("sp.business_unit_id = ?", req.BuID)
			}).
			Returning("*").
			Exec(c)

		if rErr != nil {
			log.Error("failed to update shipment status", zap.Error(rErr))
			return rErr
		}

		roErr := dberror.CheckRowsAffected(results, "Shipment", req.ShipmentID.String())
		if roErr != nil {
			return roErr
		}

		return nil
	})
	if err != nil {
		log.Error("failed to cancel shipment", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) TransferOwnership(
	ctx context.Context,
	req *repositories.TransferOwnershipRequest,
) (*shipment.Shipment, error) {
	log := r.l.With(
		zap.String("operation", "TransferOwnership"),
		zap.String("shipmentID", req.ShipmentID.String()),
		zap.String("ownerID", req.OwnerID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	entity := new(shipment.Shipment)

	res, err := db.NewUpdate().Model(entity).
		Set("owner_id = ?", req.OwnerID).
		Set("version = version + 1").
		OmitZero().
		WhereGroup(" AND ", func(q *bun.UpdateQuery) *bun.UpdateQuery {
			return q.
				Where("sp.id = ?", req.ShipmentID).
				Where("sp.organization_id = ?", req.OrgID).
				Where("sp.business_unit_id = ?", req.BuID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to transfer ownership", zap.Error(err))
		return nil, err
	}

	roErr := dberror.CheckRowsAffected(res, "Shipment", req.ShipmentID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *repository) GetDelayedShipments(ctx context.Context) ([]*shipment.Shipment, error) {
	log := r.l.With(
		zap.String("operation", "GetDelayedShipments"),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	now := utils.NowUnix()
	entities := make([]*shipment.Shipment, 0)

	stopCte, moveCte := r.buildDelayedShipmentsCTEs(db, now)

	q := db.NewSelect().
		Model(&entities).
		With("stop_cte", stopCte).
		With("move_cte", moveCte).
		Where("sp.id IN (SELECT shipment_id FROM move_cte)").
		Where("sp.status NOT IN (?)", bun.In([]shipment.Status{
			shipment.StatusDelayed,
			shipment.StatusCanceled,
			shipment.StatusCompleted,
			shipment.StatusBilled,
		}))

	if err = q.Scan(ctx); err != nil {
		log.Error("failed to find delayed shipments", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) CheckForDuplicateBOLs(
	ctx context.Context,
	req *repositories.DuplicateBolsRequest,
) ([]*repositories.DuplicateBOLsResult, error) {
	log := r.l.With(
		zap.String("operation", "CheckForDuplicateBOLs"),
		zap.String("bol", req.CurrentBOL),
		zap.String("orgID", req.OrgID.String()),
		zap.String("buID", req.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	query := db.NewSelect().
		Column("sp.id").
		Column("sp.pro_number").
		Model((*shipment.Shipment)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("sp.organization_id = ?", req.OrgID).
				Where("sp.business_unit_id = ?", req.BuID).
				Where("sp.bol = ?", req.CurrentBOL).
				Where("sp.status != ?", shipment.StatusCanceled)
		})

	if req.ExcludeID != nil {
		query = query.Where("sp.id != ?", pulid.ConvertFromPtr(req.ExcludeID))
	}

	duplicates := make([]*repositories.DuplicateBOLsResult, 0)

	if err = query.Scan(ctx, &duplicates); err != nil {
		log.Error("failed to query for duplicate BOLs", zap.Error(err))
		return nil, err
	}

	return duplicates, nil
}

func (r *repository) DelayShipments(ctx context.Context) ([]*shipment.Shipment, error) {
	ct := utils.NowUnix()
	log := r.l.With(
		zap.String("operation", "DelayShipments"),
		zap.Int64("currentTime", ct),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities, err := r.getDelayedShipments(ctx, db, ct)
	if err != nil {
		return nil, err
	}

	if len(entities) == 0 {
		log.Info("no shipments to delay")
		return entities, nil
	}

	err = db.RunInTx(ctx, nil, func(c context.Context, tx bun.Tx) error {
		shipmentIDs := make([]pulid.ID, 0, len(entities))
		for i, shp := range entities {
			shipmentIDs[i] = shp.ID
		}

		if _, err = tx.NewUpdate().Model((*shipment.Shipment)(nil)).
			Set("status = ?", shipment.StatusDelayed).
			Set("updated_at = ?", ct).
			Where("sp.id IN (?)", bun.In(shipmentIDs)).
			Exec(c); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, shp := range entities {
		shp.Status = shipment.StatusDelayed
		shp.UpdatedAt = ct
	}

	return entities, nil
}
