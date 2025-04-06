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

type ShipmentCommodityRepositoryParams struct {
	fx.In

	DB     db.Connection
	Logger *logger.Logger
}

type shipmentCommodityRepository struct {
	db db.Connection
	l  *zerolog.Logger
}

func NewShipmentCommodityRepository(p ShipmentCommodityRepositoryParams) repositories.ShipmentCommodityRepository {
	log := p.Logger.With().
		Str("repository", "shipmentCommodity").
		Logger()

	return &shipmentCommodityRepository{
		db: p.DB,
		l:  &log,
	}
}

func (r *shipmentCommodityRepository) HandleCommodityOperations(ctx context.Context, tx bun.IDB, shp *shipment.Shipment, isCreate bool) error {
	var err error

	// * If there are no commodities and it's a create operation, we can return early
	if len(shp.Commodities) == 0 && isCreate {
		return nil
	}

	// * Get the existing commodities for comparison if this is an update operation
	existingCommodities := make([]*shipment.ShipmentCommodity, 0)
	if !isCreate {
		existingCommodities, err = r.getExistingCommodities(ctx, tx, shp)
		if err != nil {
			r.l.Error().Err(err).Msg("failed to get existing commodities")
			return err
		}
	}

	// * Prepare commodities for operations
	newCommodities := make([]*shipment.ShipmentCommodity, 0)
	updateCommodities := make([]*shipment.ShipmentCommodity, 0)
	existingCommodityMap := make(map[pulid.ID]*shipment.ShipmentCommodity)
	updatedCommodityIDs := make(map[pulid.ID]struct{})
	commoditiesToDelete := make([]*shipment.ShipmentCommodity, 0)

	// * Create map of existing commodities for quick lookup
	for _, comm := range existingCommodities {
		existingCommodityMap[comm.ID] = comm
	}

	// * Categorize commodities for different operations
	for _, comm := range shp.Commodities {
		// *Set required fields
		comm.ShipmentID = shp.ID
		comm.OrganizationID = shp.OrganizationID
		comm.BusinessUnitID = shp.BusinessUnitID

		if isCreate || comm.ID.IsNil() {
			// * Append new commodities
			newCommodities = append(newCommodities, comm)
		} else {
			if existing, ok := existingCommodityMap[comm.ID]; ok {
				// * Increment version for optimistic locking
				comm.Version = existing.Version + 1
				updateCommodities = append(updateCommodities, comm)
				updatedCommodityIDs[comm.ID] = struct{}{}
			}
		}
	}

	// * Handle bulk insert of new commodities
	if len(newCommodities) > 0 {
		if _, err := tx.NewInsert().Model(&newCommodities).Exec(ctx); err != nil {
			r.l.Error().Err(err).Msg("failed to bulk insert new commodities")
			return err
		}
	}

	// * Handle bulk update of new commodities
	if len(updateCommodities) > 0 {
		if err := r.handleBulkUpdate(ctx, tx, updateCommodities); err != nil {
			r.l.Error().Err(err).Msg("failed to handle bulk update of commodities")
			return err
		}
	}

	// * Handle deletion of commodities that are no longer present
	if !isCreate {
		if err := r.handleCommodityDeletions(ctx, tx, &repositories.CommodityDeletionRequest{
			ExistingCommodityMap: existingCommodityMap,
			UpdatedCommodityIDs:  updatedCommodityIDs,
			CommoditiesToDelete:  commoditiesToDelete,
		}); err != nil {
			r.l.Error().Err(err).Msg("failed to handle commodity deletions")
			return err
		}
	}

	r.l.Debug().Int("new_commodities", len(newCommodities)).
		Int("updated_commodities", len(updateCommodities)).
		Int("deleted_commodities", len(commoditiesToDelete)).
		Msg("commodity operations completed")

	return nil
}

// Get the existing commodities for comparison if this is an update operation
func (r *shipmentCommodityRepository) getExistingCommodities(ctx context.Context, tx bun.IDB, shp *shipment.Shipment) ([]*shipment.ShipmentCommodity, error) {
	commodities := make([]*shipment.ShipmentCommodity, 0, len(shp.Commodities))

	// * Fetch the existing commodities
	if err := tx.NewSelect().
		Model(&commodities).
		Where("shipment_id = ?", shp.ID).
		Where("organization_id = ?", shp.OrganizationID).
		Where("business_unit_id = ?", shp.BusinessUnitID).
		Scan(ctx); err != nil {
		r.l.Error().Err(err).Msg("failed to fetch existing commodities")
		return nil, err
	}

	return commodities, nil
}

// Handle bulk update of new commodities
func (r *shipmentCommodityRepository) handleBulkUpdate(ctx context.Context, tx bun.IDB, commodities []*shipment.ShipmentCommodity) error {
	values := tx.NewValues(&commodities)

	// * Update the commodities
	res, err := tx.NewUpdate().
		With("_data", values).
		Model((*shipment.ShipmentCommodity)(nil)).
		TableExpr("_data").
		Set("shipment_id = _data.shipment_id").
		Set("commodity_id = _data.commodity_id").
		Set("weight = _data.weight").
		Set("pieces = _data.pieces").
		Set("version = _data.version").
		Where("sc.id = _data.id").
		Where("sc.version = _data.version - 1").
		Where("sc.organization_id = _data.organization_id").
		Where("sc.business_unit_id = _data.business_unit_id").
		Exec(ctx)
	if err != nil {
		r.l.Error().Err(err).Msg("failed to bulk update commodities")
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		r.l.Error().Err(err).Msg("failed to get rows affected for updated commodities")
		return err
	}

	if int(rowsAffected) != len(commodities) {
		return errors.NewValidationError(
			"version",
			errors.ErrVersionMismatch,
			"One or more commodities have been modified since last retrieval",
		)
	}

	r.l.Debug().Int("count", len(commodities)).Msg("bulk updated commodities")

	return nil
}

// Handle deletion of commodities that are no longer present
func (r *shipmentCommodityRepository) handleCommodityDeletions(ctx context.Context, tx bun.IDB, req *repositories.CommodityDeletionRequest) error {
	// * For each existing commodity, check if it has been updated
	for id, commodity := range req.ExistingCommodityMap {
		if _, exists := req.UpdatedCommodityIDs[id]; !exists {
			req.CommoditiesToDelete = append(req.CommoditiesToDelete, commodity)
		}
	}

	// * If there are any commodities to delete, delete them
	if len(req.CommoditiesToDelete) > 0 {
		_, err := tx.NewDelete().
			Model(&req.CommoditiesToDelete).
			WherePK().
			Exec(ctx)
		if err != nil {
			r.l.Error().Err(err).Msg("failed to bulk delete commodities")
			return err
		}
	}

	return nil
}
