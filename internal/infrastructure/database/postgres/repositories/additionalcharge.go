package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/db"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type AdditionalChargeRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type additionalChargeRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewAdditionalChargeRepository(
	p AdditionalChargeRepositoryParams,
) repositories.AdditionalChargeRepository {
	log := p.Logger.With().
		Str("repository", "additionalCharge").
		Logger()

	return &additionalChargeRepository{
		db: p.DB,
		l:  &log,
	}
}

func (r *additionalChargeRepository) HandleAdditionalChargeOperations(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
	isCreate bool,
) error {
	// Early return for create with no charges
	if len(shp.AdditionalCharges) == 0 && isCreate {
		return nil
	}

	// Prepare operation data
	opData, err := r.prepareOperationData(ctx, tx, shp, isCreate)
	if err != nil {
		return err
	}

	// Process insert operations
	if err = r.processInserts(ctx, tx, opData.NewCharges); err != nil {
		return err
	}

	// Process update operations
	if err = r.processUpdates(ctx, tx, opData.UpdateCharges); err != nil {
		return err
	}

	// Process delete operations
	if !isCreate {
		if err = r.handleAdditionalChargeDeletions(ctx, tx, &repositories.AdditionalChargeDeletionRequest{
			ExistingAdditionalChargeMap: opData.ExistingMap,
			UpdatedAdditionalChargeIDs:  opData.UpdatedIDs,
			AdditionalChargesToDelete:   opData.DeleteCharges,
		}); err != nil {
			r.l.Error().Err(err).Msg("failed to handle additional charge deletions")
			return err
		}
	}

	r.l.Debug().Int("newAdditionalCharges", len(opData.NewCharges)).
		Int("updatedAdditionalCharges", len(opData.UpdateCharges)).
		Int("deletedAdditionalCharges", len(opData.DeleteCharges)).
		Msg("Additional Charge operations completed")

	return nil
}

// OperationData holds prepared data for CRUD operations
type operationData struct {
	ExistingMap   map[pulid.ID]*shipment.AdditionalCharge
	UpdatedIDs    map[pulid.ID]struct{}
	NewCharges    []*shipment.AdditionalCharge
	UpdateCharges []*shipment.AdditionalCharge
	DeleteCharges []*shipment.AdditionalCharge
}

func (r *additionalChargeRepository) prepareOperationData(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
	isCreate bool,
) (*operationData, error) {
	data := &operationData{
		ExistingMap:   make(map[pulid.ID]*shipment.AdditionalCharge),
		UpdatedIDs:    make(map[pulid.ID]struct{}),
		NewCharges:    make([]*shipment.AdditionalCharge, 0),
		UpdateCharges: make([]*shipment.AdditionalCharge, 0),
		DeleteCharges: make([]*shipment.AdditionalCharge, 0),
	}

	// Get existing charges for updates
	if !isCreate {
		existingCharges, err := r.getExistingAdditionalCharges(ctx, tx, shp)
		if err != nil {
			r.l.Error().Err(err).Msg("failed to get existing additional charges")
			return nil, err
		}

		// Map existing charges by ID
		for _, ac := range existingCharges {
			data.ExistingMap[ac.ID] = ac
		}
	}

	// Categorize charges for operations
	r.categorizeCharges(shp, data, isCreate)

	return data, nil
}

func (r *additionalChargeRepository) categorizeCharges(
	shp *shipment.Shipment,
	data *operationData,
	isCreate bool,
) {
	for _, ac := range shp.AdditionalCharges {
		// Set required fields
		ac.ShipmentID = shp.ID
		ac.OrganizationID = shp.OrganizationID
		ac.BusinessUnitID = shp.BusinessUnitID

		if isCreate || ac.ID.IsNil() {
			data.NewCharges = append(data.NewCharges, ac)
		} else if existing, ok := data.ExistingMap[ac.ID]; ok {
			ac.Version = existing.Version + 1
			data.UpdateCharges = append(data.UpdateCharges, ac)
			data.UpdatedIDs[ac.ID] = struct{}{}
		}
	}
}

func (r *additionalChargeRepository) processInserts(
	ctx context.Context,
	tx bun.IDB,
	newCharges []*shipment.AdditionalCharge,
) error {
	if len(newCharges) == 0 {
		return nil
	}

	if _, err := tx.NewInsert().Model(&newCharges).Exec(ctx); err != nil {
		r.l.Error().Err(err).Msg("failed to bulk insert new additional charges")
		return err
	}

	return nil
}

func (r *additionalChargeRepository) processUpdates(
	ctx context.Context,
	tx bun.IDB,
	updateCharges []*shipment.AdditionalCharge,
) error {
	if len(updateCharges) == 0 {
		return nil
	}

	if err := r.handleBulkUpdate(ctx, tx, updateCharges); err != nil {
		r.l.Error().Err(err).Msg("failed to handle bulk update of additional charges")
		return err
	}

	return nil
}

func (r *additionalChargeRepository) getExistingAdditionalCharges(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
) ([]*shipment.AdditionalCharge, error) {
	additionalCharges := make([]*shipment.AdditionalCharge, 0, len(shp.AdditionalCharges))

	if err := tx.NewSelect().
		Model(&additionalCharges).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where("shipment_id = ?", shp.ID).
				Where("organization_id = ?", shp.OrganizationID).
				Where("business_unit_id = ?", shp.BusinessUnitID)
		}).
		Scan(ctx); err != nil {
		r.l.Error().Err(err).Msg("failed to fetch existing additional charges")
		return nil, err
	}

	return additionalCharges, nil
}

func (r *additionalChargeRepository) handleBulkUpdate(
	ctx context.Context,
	tx bun.IDB,
	additionalCharges []*shipment.AdditionalCharge,
) error {
	values := tx.NewValues(&additionalCharges)

	// * Update the additional charges
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
		r.l.Error().Err(err).Msg("failed to bulk update additional charges")
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		r.l.Error().Err(err).Msg("failed to get rows affected for updated additional charges")
		return err
	}

	if int(rowsAffected) != len(additionalCharges) {
		return errors.NewValidationError(
			"version",
			errors.ErrVersionMismatch,
			"One or more additional charges have been modified since last retrieval",
		)
	}

	r.l.Debug().Int("count", len(additionalCharges)).Msg("bulk updated additional charges")

	return nil
}

func (r *additionalChargeRepository) handleAdditionalChargeDeletions(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.AdditionalChargeDeletionRequest,
) error {
	// * for each existing additional charge, check if it has been updated
	for id, additionalCharge := range req.ExistingAdditionalChargeMap {
		if _, exists := req.UpdatedAdditionalChargeIDs[id]; !exists {
			req.AdditionalChargesToDelete = append(req.AdditionalChargesToDelete, additionalCharge)
		}
	}

	// * if there are any additional charges to delete, delete them
	if len(req.AdditionalChargesToDelete) > 0 {
		_, err := tx.NewDelete().
			Model(&req.AdditionalChargesToDelete).
			WherePK().
			Exec(ctx)
		if err != nil {
			r.l.Error().Err(err).Msg("failed to bulk delete additional charges")
			return err
		}
	}

	return nil
}
