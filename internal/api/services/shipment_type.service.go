package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/shipmenttype"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
)

type ShipmentTypeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewShipmentTypeService creates a new shipment type service.
func NewShipmentTypeService(s *api.Server) *ShipmentTypeService {
	return &ShipmentTypeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetShipmentTypes gets the shipment types for an organization.
func (r *ShipmentTypeService) GetShipmentTypes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.ShipmentType, int, error) {
	entityCount, countErr := r.Client.ShipmentType.Query().Where(
		shipmenttype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.ShipmentType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			shipmenttype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateShipmentType creates a new shipment type.
func (r *ShipmentTypeService) CreateShipmentType(
	ctx context.Context, entity *ent.ShipmentType,
) (*ent.ShipmentType, error) {
	newEntity := new(ent.ShipmentType)

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

func (r *ShipmentTypeService) createShipmentTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.ShipmentType,
) (*ent.ShipmentType, error) {
	createdEntity, err := tx.ShipmentType.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetColor(entity.Color).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create entity")
	}

	return createdEntity, nil
}

// UpdateShipmentType updates a shipment type.
func (r *ShipmentTypeService) UpdateShipmentType(
	ctx context.Context, entity *ent.ShipmentType,
) (*ent.ShipmentType, error) {
	updatedEntity := new(ent.ShipmentType)

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

func (r *ShipmentTypeService) updateShipmentTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.ShipmentType,
) (*ent.ShipmentType, error) {
	current, err := tx.ShipmentType.Get(ctx, entity.ID)
	if err != nil {
		return nil, eris.Wrap(err, "failed to retrieve requested entity")
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"code")
	}

	// Start building the update operation
	updateOp := tx.ShipmentType.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetColor(entity.Color).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
