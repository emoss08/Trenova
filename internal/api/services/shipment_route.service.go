package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/shipmentroute"
	"github.com/google/uuid"
)

type ShipmentRouteService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewShipmentRouteService creates a new shipment route service.
func NewShipmentRouteService(s *api.Server) *ShipmentRouteService {
	return &ShipmentRouteService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetShipmentTypes gets the shipment routes for an organization.
func (r *ShipmentRouteService) GetShipmentTypes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.ShipmentRoute, int, error) {
	entityCount, countErr := r.Client.ShipmentRoute.Query().Where(
		shipmentroute.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.ShipmentRoute.Query().
		Limit(limit).
		Offset(offset).
		Where(
			shipmentroute.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateShipmentType creates a new shipment route.
func (r *ShipmentRouteService) CreateShipmentType(
	ctx context.Context, entity *ent.ShipmentRoute,
) (*ent.ShipmentRoute, error) {
	newEntity := new(ent.ShipmentRoute)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createShipmentTypeEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return newEntity, nil
}

func (r *ShipmentRouteService) createShipmentTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.ShipmentRoute,
) (*ent.ShipmentRoute, error) {
	createdEntity, err := tx.ShipmentRoute.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateShipmentType updates a shipment route.
func (r *ShipmentRouteService) UpdateShipmentType(
	ctx context.Context, entity *ent.ShipmentRoute,
) (*ent.ShipmentRoute, error) {
	updatedEntity := new(ent.ShipmentRoute)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateShipmentTypeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *ShipmentRouteService) updateShipmentTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.ShipmentRoute,
) (*ent.ShipmentRoute, error) {
	current, err := tx.ShipmentRoute.Get(ctx, entity.ID)
	if err != nil {
		return nil, err
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.ShipmentRoute.UpdateOneID(entity.ID).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
