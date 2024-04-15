package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/trailer"
	"github.com/google/uuid"
)

type TrailerService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewTrailerService creates a new trailer service.
func NewTrailerService(s *api.Server) *TrailerService {
	return &TrailerService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetTrailers gets the trailers for an organization.
func (r *TrailerService) GetTrailers(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.Trailer, int, error) {
	entityCount, countErr := r.Client.Trailer.Query().Where(
		trailer.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.Trailer.Query().
		Limit(limit).
		Offset(offset).
		WithEquipmentManufacturer().
		WithState().
		WithRegistrationState().
		WithEquipmentType().
		WithFleetCode().
		Where(
			trailer.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateTrailer creates a new trailer.
func (r *TrailerService) CreateTrailer(
	ctx context.Context, entity *ent.Trailer,
) (*ent.Trailer, error) {
	newEntity := new(ent.Trailer)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createTrailerEntity(ctx, tx, entity)
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

func (r *TrailerService) createTrailerEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Trailer,
) (*ent.Trailer, error) {
	createdEntity, err := tx.Trailer.Create().
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
		SetFleetCodeID(entity.FleetCodeID).
		SetLastInspectionDate(entity.LastInspectionDate).
		SetRegistrationNumber(entity.RegistrationNumber).
		SetNillableRegistrationStateID(entity.RegistrationStateID).
		SetRegistrationExpirationDate(entity.RegistrationExpirationDate).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateTrailer updates a trailer.
func (r *TrailerService) UpdateTrailer(
	ctx context.Context, entity *ent.Trailer,
) (*ent.Trailer, error) {
	updatedEntity := new(ent.Trailer)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateTrailerEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *TrailerService) updateTrailerEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.Trailer,
) (*ent.Trailer, error) {
	current, err := tx.Trailer.Get(ctx, entity.ID)
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
	updateOp := tx.Trailer.UpdateOneID(entity.ID).
		SetCode(entity.Code).
		SetStatus(entity.Status).
		SetEquipmentTypeID(entity.EquipmentTypeID).
		SetLicensePlateNumber(entity.LicensePlateNumber).
		SetVin(entity.Vin).
		SetNillableEquipmentManufacturerID(entity.EquipmentManufacturerID).
		SetModel(entity.Model).
		SetNillableYear(entity.Year).
		SetNillableStateID(entity.StateID).
		SetFleetCodeID(entity.FleetCodeID).
		SetLastInspectionDate(entity.LastInspectionDate).
		SetRegistrationNumber(entity.RegistrationNumber).
		SetNillableRegistrationStateID(entity.RegistrationStateID).
		SetRegistrationExpirationDate(entity.RegistrationExpirationDate).
		SetVersion(entity.Version + 1) // Increment the version

	// If registration state id is nil clear the assocation.
	if entity.RegistrationStateID == nil {
		updateOp.ClearRegistrationState()
	}

	// If the equipment manufacturer id is nil clear the association.
	if entity.EquipmentManufacturerID == nil {
		updateOp.ClearEquipmentManufacturer()
	}

	// If the registration state id is nil clear the association.
	if entity.RegistrationStateID == nil {
		updateOp.ClearRegistrationState()
	}

	// If the state id is nil clear the association.
	if entity.StateID == nil {
		updateOp.ClearState()
	}

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
