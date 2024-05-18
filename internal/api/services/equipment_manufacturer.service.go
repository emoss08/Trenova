package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/equipmentmanufactuer"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

type EquipmentManufacturerService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewEquipmentManufacturerService creates a new equipment manufacturer service.
func NewEquipmentManufacturerService(s *api.Server) *EquipmentManufacturerService {
	return &EquipmentManufacturerService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetEquipmentManufacturers gets the equipment manufacturers for an organization.
func (r *EquipmentManufacturerService) GetEquipmentManufacturers(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.EquipmentManufactuer, int, error) {
	entityCount, countErr := r.Client.EquipmentManufactuer.Query().Where(
		equipmentmanufactuer.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.EquipmentManufactuer.Query().
		Limit(limit).
		Offset(offset).
		Where(
			equipmentmanufactuer.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateEquipmentManufacturer creates a new equipment manufacturer.
func (r *EquipmentManufacturerService) CreateEquipmentManufacturer(
	ctx context.Context, entity *ent.EquipmentManufactuer,
) (*ent.EquipmentManufactuer, error) {
	updatedEntity := new(ent.EquipmentManufactuer)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createEquipmentManufacturerEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *EquipmentManufacturerService) createEquipmentManufacturerEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.EquipmentManufactuer,
) (*ent.EquipmentManufactuer, error) {
	createdEntity, err := tx.EquipmentManufactuer.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetDescription(entity.Description).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateEquipmentManufacturer updates a equipment manufacturer.
func (r *EquipmentManufacturerService) UpdateEquipmentManufacturer(
	ctx context.Context, entity *ent.EquipmentManufactuer,
) (*ent.EquipmentManufactuer, error) {
	updatedEntity := new(ent.EquipmentManufactuer)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateEquipmentManufacturerEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *EquipmentManufacturerService) updateEquipmentManufacturerEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.EquipmentManufactuer,
) (*ent.EquipmentManufactuer, error) {
	current, err := tx.EquipmentManufactuer.Get(ctx, entity.ID)
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
	updateOp := tx.EquipmentManufactuer.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetDescription(entity.Description).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
