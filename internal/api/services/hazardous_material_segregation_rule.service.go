package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/hazardousmaterialsegregation"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
)

type HazardousMaterialSegregationService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewHazardousMaterialSegregationService creates a new Hazardous Material Segregation Rule service.
func NewHazardousMaterialSegregationService(s *api.Server) *HazardousMaterialSegregationService {
	return &HazardousMaterialSegregationService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetHazmatSegRules gets the Hazardous Material Segregation Rule for an organization.
func (r *HazardousMaterialSegregationService) GetHazmatSegregationRules(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.HazardousMaterialSegregation, int, error) {
	entityCount, countErr := r.Client.HazardousMaterialSegregation.Query().Where(
		hazardousmaterialsegregation.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.HazardousMaterialSegregation.Query().
		Limit(limit).
		Offset(offset).
		Where(
			hazardousmaterialsegregation.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateHazmatSegRule creates a new Hazardous Material Segregation Rule entity.
func (r *HazardousMaterialSegregationService) CreateHazmatSegregationRule(
	ctx context.Context, entity *ent.HazardousMaterialSegregation,
) (*ent.HazardousMaterialSegregation, error) {
	updatedEntity := new(ent.HazardousMaterialSegregation)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createHazmatSegregationRuleEntity(ctx, tx, entity)
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

func (r *HazardousMaterialSegregationService) createHazmatSegregationRuleEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.HazardousMaterialSegregation,
) (*ent.HazardousMaterialSegregation, error) {
	createdEntity, err := tx.HazardousMaterialSegregation.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetClassA(entity.ClassA).
		SetClassB(entity.ClassB).
		SetSegregationType(entity.SegregationType).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

func (r *HazardousMaterialSegregationService) UpdateHazmatSegregationRule(
	ctx context.Context, entity *ent.HazardousMaterialSegregation,
) (*ent.HazardousMaterialSegregation, error) {
	var updatedEntity *ent.HazardousMaterialSegregation

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateHazmatSegregationRuleEntity(ctx, tx, entity)
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

func (r *HazardousMaterialSegregationService) updateHazmatSegregationRuleEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.HazardousMaterialSegregation,
) (*ent.HazardousMaterialSegregation, error) {
	// Start building the update operation
	updateOp := tx.HazardousMaterialSegregation.UpdateOneID(entity.ID).
		SetClassA(entity.ClassA).
		SetClassB(entity.ClassB).
		SetSegregationType(entity.SegregationType).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
