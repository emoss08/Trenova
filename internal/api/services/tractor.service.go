package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/tractor"
	"github.com/google/uuid"
)

type TractorService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewTractorService creates a new tractor service.
func NewTractorService(s *api.Server) *TractorService {
	return &TractorService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetTractors gets the tractors for an organization.
func (r *TractorService) GetTractors(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.Tractor, int, error) {
	entityCount, countErr := r.Client.Tractor.Query().Where(
		tractor.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.Tractor.Query().
		Limit(limit).
		Offset(offset).
		WithEquipmentType().
		WithPrimaryWorker().
		WithSecondaryWorker().
		WithFleetCode().
		Where(
			tractor.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateTractor creates a new tractor.
func (r *TractorService) CreateTractor(
	ctx context.Context, entity *ent.Tractor,
) (*ent.Tractor, error) {
	newEntity := new(ent.Tractor)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createTractorEntity(ctx, tx, entity)
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

func (r *TractorService) createTractorEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Tractor,
) (*ent.Tractor, error) {
	createdEntity, err := tx.Tractor.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetEquipmentTypeID(entity.EquipmentTypeID).
		SetLicensePlateNumber(entity.LicensePlateNumber).
		SetVin(entity.Vin).
		SetNillableEquipmentManufacturerID(entity.EquipmentManufacturerID).
		SetModel(entity.Model).
		SetNillableYear(entity.Year).
		SetNillableStateID(entity.StateID).
		SetLeased(entity.Leased).
		SetLeasedDate(entity.LeasedDate).
		SetPrimaryWorkerID(entity.PrimaryWorkerID).
		SetNillableSecondaryWorkerID(entity.SecondaryWorkerID).
		SetFleetCodeID(entity.FleetCodeID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateTractor updates a tractor.
func (r *TractorService) UpdateTractor(
	ctx context.Context, entity *ent.Tractor,
) (*ent.Tractor, error) {
	updatedEntity := new(ent.Tractor)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateTractorEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *TractorService) updateTractorEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Tractor,
) (*ent.Tractor, error) {
	current, err := tx.Tractor.Get(ctx, entity.ID)
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
	updateOp := tx.Tractor.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetEquipmentTypeID(entity.EquipmentTypeID).
		SetLicensePlateNumber(entity.LicensePlateNumber).
		SetVin(entity.Vin).
		SetNillableEquipmentManufacturerID(entity.EquipmentManufacturerID).
		SetModel(entity.Model).
		SetNillableYear(entity.Year).
		SetNillableStateID(entity.StateID).
		SetLeased(entity.Leased).
		SetLeasedDate(entity.LeasedDate).
		SetPrimaryWorkerID(entity.PrimaryWorkerID).
		SetNillableSecondaryWorkerID(entity.SecondaryWorkerID).
		SetFleetCodeID(entity.FleetCodeID).
		SetVersion(entity.Version + 1) // Increment the version

	// If the secondary worker ID is nil, clear the association.
	if entity.SecondaryWorkerID == nil {
		updateOp = updateOp.ClearSecondaryWorker()
	}

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
