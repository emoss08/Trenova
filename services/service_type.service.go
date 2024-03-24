package services

import (
	"context"

	"github.com/emoss08/trenova/ent/servicetype"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type ServiceTypeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewServiceTypeOps creates a new service type service.
func NewServiceTypeOps(ctx context.Context) *ServiceTypeOps {
	return &ServiceTypeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetServiceTypes gets the service type for an organization.
func (r *ServiceTypeOps) GetServiceTypes(limit, offset int, orgID, buID uuid.UUID) ([]*ent.ServiceType, int, error) {
	serviceTypeCount, countErr := r.client.ServiceType.Query().Where(
		servicetype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	serviceTypes, err := r.client.ServiceType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			servicetype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return serviceTypes, serviceTypeCount, nil
}

// CreateServiceType creates a new service type.
func (r *ServiceTypeOps) CreateServiceType(newServiceType ent.ServiceType) (*ent.ServiceType, error) {
	serviceType, err := r.client.ServiceType.Create().
		SetOrganizationID(newServiceType.OrganizationID).
		SetBusinessUnitID(newServiceType.BusinessUnitID).
		SetStatus(newServiceType.Status).
		SetCode(newServiceType.Code).
		SetDescription(newServiceType.Description).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return serviceType, nil
}

// UpdateServiceType updates a service type.
func (r *ServiceTypeOps) UpdateServiceType(serviceType ent.ServiceType) (*ent.ServiceType, error) {
	// Start building the update operation
	updateOp := r.client.ServiceType.UpdateOneID(serviceType.ID).
		SetStatus(serviceType.Status).
		SetCode(serviceType.Code).
		SetDescription(serviceType.Description)

	// Execute the update operation
	updateserviceType, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updateserviceType, nil
}
