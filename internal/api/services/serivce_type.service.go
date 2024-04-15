package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/servicetype"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
)

type ServiceTypeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewServiceTypeService creates a new service type service.
func NewServiceTypeService(s *api.Server) *ServiceTypeService {
	return &ServiceTypeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetServiceTypes gets the service types for an organization.
func (r *ServiceTypeService) GetServiceTypes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.ServiceType, int, error) {
	entityCount, countErr := r.Client.ServiceType.Query().Where(
		servicetype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.ServiceType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			servicetype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateServiceType creates a new service type.
func (r *ServiceTypeService) CreateServiceType(
	ctx context.Context, entity *ent.ServiceType,
) (*ent.ServiceType, error) {
	newEntity := new(ent.ServiceType)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createServiceTypeEntity(ctx, tx, entity)
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

func (r *ServiceTypeService) createServiceTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.ServiceType,
) (*ent.ServiceType, error) {
	createdEntity, err := tx.ServiceType.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create entity")
	}

	return createdEntity, nil
}

// UpdateServiceType updates a service type.
func (r *ServiceTypeService) UpdateServiceType(
	ctx context.Context, entity *ent.ServiceType,
) (*ent.ServiceType, error) {
	updatedEntity := new(ent.ServiceType)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateServiceTypeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *ServiceTypeService) updateServiceTypeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.ServiceType,
) (*ent.ServiceType, error) {
	current, err := tx.ServiceType.Get(ctx, entity.ID)
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
	updateOp := tx.ServiceType.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
