package additionalchargerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
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

func NewRepository(p Params) repositories.AdditionalChargeRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.additionalcharge-repository"),
	}
}

type operationData struct {
	ExistingMap   map[pulid.ID]*shipment.AdditionalCharge
	UpdatedIDs    map[pulid.ID]struct{}
	NewCharges    []*shipment.AdditionalCharge
	UpdateCharges []*shipment.AdditionalCharge
	DeleteCharges []*shipment.AdditionalCharge
}

func (r *repository) HandleOperations(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
	isCreate bool,
) error {
	if len(entity.AdditionalCharges) == 0 && isCreate {
		return nil
	}

	opData, err := r.prepareOperationData(ctx, tx, entity, isCreate)
	if err != nil {
		return err
	}

	if err = r.processInserts(ctx, tx, opData.NewCharges); err != nil {
		return err
	}

	if err = r.processUpdates(ctx, tx, opData.UpdateCharges); err != nil {
		return err
	}

	if !isCreate {
		if err = r.handleChargeDeletions(ctx, tx, &repositories.AdditionalChargeDeletionRequest{
			ExistingAdditionalChargeMap: opData.ExistingMap,
			UpdatedAdditionalChargeIDs:  opData.UpdatedIDs,
			AdditionalChargesToDelete:   opData.DeleteCharges,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) getExistingCharges(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
) ([]*shipment.AdditionalCharge, error) {
	log := r.l.With(
		zap.String("operation", "getExistingCharges"),
		zap.String("shipmentID", entity.ID.String()),
		zap.String("businessUnitID", entity.BusinessUnitID.String()),
		zap.String("organizationID", entity.OrganizationID.String()),
	)

	ac := make([]*shipment.AdditionalCharge, 0, len(entity.AdditionalCharges))

	if err := tx.NewSelect().Model(&ac).WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where("ac.shipment_id = ?", entity.ID).
			Where("ac.business_unit_id = ?", entity.BusinessUnitID).
			Where("ac.organization_id = ?", entity.OrganizationID)
	}).Scan(ctx); err != nil {
		log.Error("failed to get existing charges", zap.Error(err))
		return nil, err
	}

	return ac, nil
}

func (r *repository) prepareOperationData(
	ctx context.Context,
	tx bun.IDB,
	entity *shipment.Shipment,
	isCreate bool,
) (*operationData, error) {
	log := r.l.With(
		zap.String("operation", "prepareOperationData"),
		zap.String("shipmentID", entity.ID.String()),
		zap.String("businessUnitID", entity.BusinessUnitID.String()),
		zap.String("organizationID", entity.OrganizationID.String()),
		zap.Bool("isCreate", isCreate),
	)

	data := &operationData{
		ExistingMap:   make(map[pulid.ID]*shipment.AdditionalCharge),
		UpdatedIDs:    make(map[pulid.ID]struct{}),
		NewCharges:    make([]*shipment.AdditionalCharge, 0),
		UpdateCharges: make([]*shipment.AdditionalCharge, 0),
		DeleteCharges: make([]*shipment.AdditionalCharge, 0),
	}

	if !isCreate {
		existingCharges, err := r.getExistingCharges(ctx, tx, entity)
		if err != nil {
			log.Error("failed to get existing charges", zap.Error(err))
			return nil, err
		}

		for _, ac := range existingCharges {
			data.ExistingMap[ac.ID] = ac
		}
	}

	r.categorizeCharges(entity, data, isCreate)

	return data, nil
}

func (r *repository) categorizeCharges(
	entity *shipment.Shipment,
	data *operationData,
	isCreate bool,
) {
	for _, ac := range entity.AdditionalCharges {
		ac.ShipmentID = entity.ID
		ac.OrganizationID = entity.OrganizationID
		ac.BusinessUnitID = entity.BusinessUnitID

		if isCreate || ac.ID.IsNil() {
			data.NewCharges = append(data.NewCharges, ac)
		} else if existing, ok := data.ExistingMap[ac.ID]; ok {
			ac.Version = existing.Version + 1
			data.UpdateCharges = append(data.UpdateCharges, ac)
			data.UpdatedIDs[ac.ID] = struct{}{}
		}
	}
}

func (r *repository) processInserts(
	ctx context.Context,
	tx bun.IDB,
	entities []*shipment.AdditionalCharge,
) error {
	log := r.l.With(
		zap.String("operation", "processInserts"),
		zap.Int("count", len(entities)),
	)

	if len(entities) == 0 {
		log.Debug("no new charges to process")
		return nil
	}

	if _, err := tx.NewInsert().Model(&entities).Exec(ctx); err != nil {
		log.Error("failed to bulk insert additional charges", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) processUpdates(
	ctx context.Context,
	tx bun.IDB,
	entities []*shipment.AdditionalCharge,
) error {
	log := r.l.With(
		zap.String("operation", "processUpdates"),
		zap.Int("count", len(entities)),
	)

	if len(entities) == 0 {
		log.Debug("no update charges to process")
		return nil
	}

	if err := r.handleBulkUpdate(ctx, tx, entities); err != nil {
		log.Error("failed to handle bulk update of additional charges", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) handleBulkUpdate(
	ctx context.Context,
	tx bun.IDB,
	entities []*shipment.AdditionalCharge,
) error {
	log := r.l.With(
		zap.String("operation", "handleBulkUpdate"),
		zap.Int("count", len(entities)),
	)

	values := tx.NewValues(&entities)

	res, err := tx.NewUpdate().
		With("_data", values).
		Model((*shipment.AdditionalCharge)(nil)).
		TableExpr("_data").
		Set("shipment_id = _data.shipment_id").
		Set("accessorial_charge_id = _data.accessorial_charge_id").
		Set("unit = _data.unit").
		Set("method = _data.method").
		Set("amount = _data.amount").
		Set("version = _data.version").
		Where("ac.id = _data.id").
		Where("ac.version = _data.version - 1").
		Where("ac.organization_id = _data.organization_id").
		Where("ac.business_unit_id = _data.business_unit_id").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update additional charges", zap.Error(err))
		return err
	}

	ra, err := res.RowsAffected()
	if err != nil {
		log.Error("failed to get rows affected", zap.Error(err))
		return err
	}

	if int(ra) != len(entities) {
		return errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"One or more additional charges have been modified since last retrieval",
		)
	}

	return nil
}

func (r *repository) handleChargeDeletions(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.AdditionalChargeDeletionRequest,
) error {
	log := r.l.With(
		zap.String("operation", "handleChargeDeletions"),
		zap.Any("req", req),
	)

	for id, ac := range req.ExistingAdditionalChargeMap {
		if _, exists := req.UpdatedAdditionalChargeIDs[id]; !exists {
			req.AdditionalChargesToDelete = append(req.AdditionalChargesToDelete, ac)
		}
	}

	if len(req.AdditionalChargesToDelete) > 0 {
		_, err := tx.NewDelete().
			Model(&req.AdditionalChargesToDelete).
			WhereGroup(" AND ", func(q *bun.DeleteQuery) *bun.DeleteQuery {
				return q.Where("ac.id = ?", req.AdditionalChargesToDelete[0].ID).
					Where("ac.organization_id = ?", req.AdditionalChargesToDelete[0].OrganizationID).
					Where("ac.business_unit_id = ?", req.AdditionalChargesToDelete[0].BusinessUnitID)
			}).
			Exec(ctx)
		if err != nil {
			log.Error("failed to delete additional charges", zap.Error(err))
			return err
		}
	}

	return nil
}
