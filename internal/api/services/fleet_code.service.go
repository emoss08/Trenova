package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/fleetcode"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

type FleetCodeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewFleetCodeService creates a new fleet code service.
func NewFleetCodeService(s *api.Server) *FleetCodeService {
	return &FleetCodeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetFleetCodes gets the fleet codes for an organization.
func (r *FleetCodeService) GetFleetCodes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.FleetCode, int, error) {
	entityCount, countErr := r.Client.FleetCode.Query().Where(
		fleetcode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.FleetCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			fleetcode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateFleetCode creates a new fleet code.
func (r *FleetCodeService) CreateFleetCode(
	ctx context.Context, entity *ent.FleetCode,
) (*ent.FleetCode, error) {
	newEntity := new(ent.FleetCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createFleetCodeEntity(ctx, tx, entity)
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

func (r *FleetCodeService) createFleetCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.FleetCode,
) (*ent.FleetCode, error) {
	createdEntity, err := tx.FleetCode.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetRevenueGoal(entity.RevenueGoal).
		SetDeadheadGoal(entity.DeadheadGoal).
		SetMileageGoal(entity.MileageGoal).
		SetColor(entity.Color).
		SetNillableManagerID(entity.ManagerID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateFleetCode updates a fleet code.
func (r *FleetCodeService) UpdateFleetCode(
	ctx context.Context, entity *ent.FleetCode,
) (*ent.FleetCode, error) {
	updatedEntity := new(ent.FleetCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateFleetCodeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *FleetCodeService) updateFleetCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.FleetCode,
) (*ent.FleetCode, error) {
	current, err := tx.FleetCode.Get(ctx, entity.ID)
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
	updateOp := tx.FleetCode.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetRevenueGoal(entity.RevenueGoal).
		SetDeadheadGoal(entity.DeadheadGoal).
		SetMileageGoal(entity.MileageGoal).
		SetColor(entity.Color).
		SetNillableManagerID(entity.ManagerID).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
