// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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

func NewShipmentCommodityRepository(
	p ShipmentCommodityRepositoryParams,
) repositories.ShipmentCommodityRepository {
	log := p.Logger.With().
		Str("repository", "shipmentCommodity").
		Logger()

	return &shipmentCommodityRepository{
		db: p.DB,
		l:  &log,
	}
}

func (r *shipmentCommodityRepository) HandleCommodityOperations(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
	isCreate bool,
) error {
	// Early return for create operation with no commodities
	if len(shp.Commodities) == 0 && isCreate {
		return nil
	}

	// Get existing commodities for update operations
	existingCommodityMap := make(map[pulid.ID]*shipment.ShipmentCommodity)
	if !isCreate {
		if err := r.loadExistingCommoditiesMap(ctx, tx, shp, existingCommodityMap); err != nil {
			return err
		}
	}

	// Categorize commodities and prepare for database operations
	newCommodities, updateCommodities, updatedCommodityIDs := r.categorizeCommodities(
		shp,
		existingCommodityMap,
		isCreate,
	)

	// Process database operations
	if err := r.processOperations(ctx, tx, newCommodities, updateCommodities); err != nil {
		return err
	}

	// Handle deletions for update operations
	if !isCreate {
		commoditiesToDelete := make([]*shipment.ShipmentCommodity, 0)
		if err := r.handleCommodityDeletions(ctx, tx, &repositories.CommodityDeletionRequest{
			ExistingCommodityMap: existingCommodityMap,
			UpdatedCommodityIDs:  updatedCommodityIDs,
			CommoditiesToDelete:  commoditiesToDelete,
		}); err != nil {
			r.l.Error().Err(err).Msg("failed to handle commodity deletions")
			return err
		}

		r.l.Debug().Int("newCommodities", len(newCommodities)).
			Int("updatedCommodities", len(updateCommodities)).
			Int("deletedCommodities", len(commoditiesToDelete)).
			Msg("commodity operations completed")
	} else {
		r.l.Debug().Int("newCommodities", len(newCommodities)).
			Int("updatedCommodities", len(updateCommodities)).
			Msg("commodity operations completed")
	}

	return nil
}

// loadExistingCommoditiesMap loads existing commodities into a map for lookup
func (r *shipmentCommodityRepository) loadExistingCommoditiesMap(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
	commodityMap map[pulid.ID]*shipment.ShipmentCommodity,
) error {
	existingCommodities, err := r.getExistingCommodities(ctx, tx, shp)
	if err != nil {
		r.l.Error().Err(err).Msg("failed to get existing commodities")
		return err
	}

	for _, comm := range existingCommodities {
		commodityMap[comm.ID] = comm
	}

	return nil
}

// categorizeCommodities categorizes commodities for different operations
//
// Parameters:
//   - shp: The shipment to categorize commodities for
//   - existingCommodityMap: A map of existing commodities for lookup
//   - isCreate: Whether the operation is a create or update
//
// Returns:
//   - newCommodities: A list of new commodities
//   - updateCommodities: A list of commodities to update
//   - updatedCommodityIDs: A map of updated commodity IDs
func (r *shipmentCommodityRepository) categorizeCommodities(
	shp *shipment.Shipment,
	existingCommodityMap map[pulid.ID]*shipment.ShipmentCommodity,
	isCreate bool,
) (newCommodities, updateCommodities []*shipment.ShipmentCommodity, updatedCommodityIDs map[pulid.ID]struct{}) {
	newCommodities = make([]*shipment.ShipmentCommodity, 0)
	updateCommodities = make([]*shipment.ShipmentCommodity, 0)
	updatedCommodityIDs = make(map[pulid.ID]struct{})

	for _, comm := range shp.Commodities {
		// Set required fields
		comm.ShipmentID = shp.ID
		comm.OrganizationID = shp.OrganizationID
		comm.BusinessUnitID = shp.BusinessUnitID

		if isCreate || comm.ID.IsNil() {
			newCommodities = append(newCommodities, comm)
		} else if existing, ok := existingCommodityMap[comm.ID]; ok {
			comm.Version = existing.Version + 1
			updateCommodities = append(updateCommodities, comm)
			updatedCommodityIDs[comm.ID] = struct{}{}
		}
	}

	return newCommodities, updateCommodities, updatedCommodityIDs
}

// processOperations handles database insert and update operations
func (r *shipmentCommodityRepository) processOperations(
	ctx context.Context,
	tx bun.IDB,
	newCommodities []*shipment.ShipmentCommodity,
	updateCommodities []*shipment.ShipmentCommodity,
) error {
	// Handle bulk insert of new commodities
	if len(newCommodities) > 0 {
		if _, err := tx.NewInsert().Model(&newCommodities).Exec(ctx); err != nil {
			r.l.Error().Err(err).Msg("failed to bulk insert new commodities")
			return err
		}
	}

	// Handle bulk update of commodities
	if len(updateCommodities) > 0 {
		if err := r.handleBulkUpdate(ctx, tx, updateCommodities); err != nil {
			r.l.Error().Err(err).Msg("failed to handle bulk update of commodities")
			return err
		}
	}

	return nil
}

// Get the existing commodities for comparison if this is an update operation
func (r *shipmentCommodityRepository) getExistingCommodities(
	ctx context.Context,
	tx bun.IDB,
	shp *shipment.Shipment,
) ([]*shipment.ShipmentCommodity, error) {
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
func (r *shipmentCommodityRepository) handleBulkUpdate(
	ctx context.Context,
	tx bun.IDB,
	commodities []*shipment.ShipmentCommodity,
) error {
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
func (r *shipmentCommodityRepository) handleCommodityDeletions(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.CommodityDeletionRequest,
) error {
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
