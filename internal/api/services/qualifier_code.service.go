package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/qualifiercode"
	"github.com/google/uuid"
)

type QualifierCodeService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewQualifierCodeService creates a new qualifier code service.
func NewQualifierCodeService(s *api.Server) *QualifierCodeService {
	return &QualifierCodeService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetQualifierCodes gets the qualifier codes for an organization.
func (r *QualifierCodeService) GetQualifierCodes(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.QualifierCode, int, error) {
	entityCount, countErr := r.Client.QualifierCode.Query().Where(
		qualifiercode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.QualifierCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			qualifiercode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateQualifierCode creates a new qualifier code.
func (r *QualifierCodeService) CreateQualifierCode(
	ctx context.Context, entity *ent.QualifierCode,
) (*ent.QualifierCode, error) {
	newEntity := new(ent.QualifierCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		newEntity, err = r.createQualifierCode(ctx, tx, entity)
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

func (r *QualifierCodeService) createQualifierCode(
	ctx context.Context, tx *ent.Tx, entity *ent.QualifierCode,
) (*ent.QualifierCode, error) {
	createdEntity, err := tx.QualifierCode.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateQualifierCode updates a qualifier code.
func (r *QualifierCodeService) UpdateQualifierCode(
	ctx context.Context, entity *ent.QualifierCode,
) (*ent.QualifierCode, error) {
	updatedEntity := new(ent.QualifierCode)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateQualifierCodeEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *QualifierCodeService) updateQualifierCodeEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.QualifierCode,
) (*ent.QualifierCode, error) {
	current, err := tx.QualifierCode.Get(ctx, entity.ID)
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
	updateOp := tx.QualifierCode.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}
