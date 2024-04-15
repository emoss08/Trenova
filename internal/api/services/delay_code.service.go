package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/delaycode"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

type DelayCodeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewDelayCodeService creates a new delay code service.
func NewDelayCodeService(s *api.Server) *DelayCodeService {
	return &DelayCodeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetDelayCodes gets the delay codes for an organization.
func (r *DelayCodeService) GetDelayCodes(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*ent.DelayCode, int, error) {
	entityCount, countErr := r.Client.DelayCode.Query().Where(
		delaycode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.DelayCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			delaycode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateDelayCode creates a new delay code.
func (r *DelayCodeService) CreateDelayCode(
	ctx context.Context, entity *ent.DelayCode,
) (*ent.DelayCode, error) {
	updatedEntity := new(ent.DelayCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createDelayCodeEntity(ctx, tx, entity)
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

func (r *DelayCodeService) createDelayCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.DelayCode,
) (*ent.DelayCode, error) {
	createdEntity, err := tx.DelayCode.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetFCarrierOrDriver(entity.FCarrierOrDriver).
		SetColor(entity.Color).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateDelayCode updates a delay code.
func (r *DelayCodeService) UpdateDelayCode(ctx context.Context, entity *ent.DelayCode) (*ent.DelayCode, error) {
	updatedEntity := new(ent.DelayCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateDelayCodeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *DelayCodeService) updateDelayCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.DelayCode,
) (*ent.DelayCode, error) {
	current, err := tx.DelayCode.Get(ctx, entity.ID)
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
	updateOp := tx.DelayCode.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetFCarrierOrDriver(entity.FCarrierOrDriver).
		SetColor(entity.Color).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
