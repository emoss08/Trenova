package services

import (
	"context"

	"github.com/emoss08/trenova/ent/hazardousmaterialsegregation"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type HazardousMaterialSegregationOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewHazardousMaterialSegregationOps creates a new fleet code service.
func NewHazardousMaterialSegregationOps(ctx context.Context) *HazardousMaterialSegregationOps {
	return &HazardousMaterialSegregationOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetHazmatSegRules gets the fleet code for an organization.
func (r *HazardousMaterialSegregationOps) GetHazmatSegRules(limit, offset int, orgID, buID uuid.UUID) ([]*ent.HazardousMaterialSegregation, int, error) {
	entityCount, countErr := r.client.HazardousMaterialSegregation.Query().Where(
		hazardousmaterialsegregation.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.HazardousMaterialSegregation.Query().
		Limit(limit).
		Offset(offset).
		Where(
			hazardousmaterialsegregation.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateHazmatSegRule creates a new accessorial charge.
func (r *HazardousMaterialSegregationOps) CreateHazmatSegRule(entity ent.HazardousMaterialSegregation) (*ent.HazardousMaterialSegregation, error) {
	newEntity, err := r.client.HazardousMaterialSegregation.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetClassA(entity.ClassA).
		SetClassB(entity.ClassB).
		SetSegregationType(entity.SegregationType).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return newEntity, nil
}

// UpdateHazmatSegRule updates a fleet code.
func (r *HazardousMaterialSegregationOps) UpdateHazmatSegRule(entity ent.HazardousMaterialSegregation) (*ent.HazardousMaterialSegregation, error) {
	// Start building the update operation
	updateOp := r.client.HazardousMaterialSegregation.UpdateOneID(entity.ID).
		SetClassA(entity.ClassA).
		SetClassB(entity.ClassB).
		SetSegregationType(entity.SegregationType)

	// Execute the update operation
	updatedEntity, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
