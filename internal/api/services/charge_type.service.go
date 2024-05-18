package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/chargetype"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ChargeTypeService is the service for charge type.
type ChargeTypeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewChargeTypeService creates a new charge type service.
func NewChargeTypeService(s *api.Server) *ChargeTypeService {
	return &ChargeTypeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetChargeTypes gets the charge types for an organization.
func (r *ChargeTypeService) GetChargeTypes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.ChargeType, int, error) {
	entityCount, countErr := r.Client.ChargeType.Query().Where(
		chargetype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.ChargeType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			chargetype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateChargeType creates a new charge type.
func (r *ChargeTypeService) CreateChargeType(
	ctx context.Context, entity *ent.ChargeType,
) (*ent.ChargeType, error) {
	updatedEntity := new(ent.ChargeType)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createCreateTypeEntity(ctx, tx, entity)
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

func (r *ChargeTypeService) createCreateTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.ChargeType,
) (*ent.ChargeType, error) {
	createdEntity, err := tx.ChargeType.Create().
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

// UpdateChargeType updates a charge type.
func (r *ChargeTypeService) UpdateChargeType(
	ctx context.Context, entity *ent.ChargeType,
) (*ent.ChargeType, error) {
	updatedEntity := new(ent.ChargeType)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateChargeTypeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *ChargeTypeService) updateChargeTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.ChargeType,
) (*ent.ChargeType, error) {
	current, err := tx.ChargeType.Get(ctx, entity.ID)
	if err != nil {
		return nil, err
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"name")
	}

	// Start building the update operation
	updateOp := tx.ChargeType.UpdateOneID(entity.ID).
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
