package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/accessorialcharge"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

type AccessorialChargeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewAccessorialChargeService creates a new accessorial charge service.
func NewAccessorialChargeService(s *api.Server) *AccessorialChargeService {
	return &AccessorialChargeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetAccessorialCharges gets the accessorial charges for an organization.
func (r *AccessorialChargeService) GetAccessorialCharges(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.AccessorialCharge, int, error) {
	entityCount, countErr := r.Client.AccessorialCharge.Query().Where(
		accessorialcharge.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.AccessorialCharge.Query().
		Limit(limit).
		Offset(offset).
		Where(
			accessorialcharge.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateAccessorialCharge creates a new accessorial charge.
func (r *AccessorialChargeService) CreateAccessorialCharge(
	ctx context.Context, entity *ent.AccessorialCharge,
) (*ent.AccessorialCharge, error) {
	updatedEntity := new(ent.AccessorialCharge)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createAccessorialChargeEntity(ctx, tx, entity)
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

func (r *AccessorialChargeService) createAccessorialChargeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.AccessorialCharge,
) (*ent.AccessorialCharge, error) {
	createdEntity, err := tx.AccessorialCharge.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetIsDetention(entity.IsDetention).
		SetMethod(entity.Method).
		SetAmount(entity.Amount).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateAccessorialCharge updates a accessorial charge.
func (r *AccessorialChargeService) UpdateAccessorialCharge(ctx context.Context, entity *ent.AccessorialCharge) (*ent.AccessorialCharge, error) {
	updatedEntity := new(ent.AccessorialCharge)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateAccessorialChargeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *AccessorialChargeService) updateAccessorialChargeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.AccessorialCharge,
) (*ent.AccessorialCharge, error) {
	current, err := tx.AccessorialCharge.Get(ctx, entity.ID)
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
	updateOp := tx.AccessorialCharge.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetIsDetention(entity.IsDetention).
		SetMethod(entity.Method).
		SetAmount(entity.Amount).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
