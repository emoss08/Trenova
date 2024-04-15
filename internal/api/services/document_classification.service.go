package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/documentclassification"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
	"github.com/rotisserie/eris"
)

type DocumentClassificationService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewDocumentClassificationService creates a new document classification service.
func NewDocumentClassificationService(s *api.Server) *DocumentClassificationService {
	return &DocumentClassificationService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetDocumentClassifications gets the document classifications for an organization.
func (r *DocumentClassificationService) GetDocumentClassifications(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.DocumentClassification, int, error) {
	entityCount, countErr := r.Client.DocumentClassification.Query().Where(
		documentclassification.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.DocumentClassification.Query().
		Limit(limit).
		Offset(offset).
		Where(
			documentclassification.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateDocumentClassification creates a new document classification.
func (r *DocumentClassificationService) CreateDocumentClassification(
	ctx context.Context, entity *ent.DocumentClassification,
) (*ent.DocumentClassification, error) {
	updatedEntity := new(ent.DocumentClassification)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createDocumentClassificationEntity(ctx, tx, entity)
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

func (r *DocumentClassificationService) createDocumentClassificationEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.DocumentClassification,
) (*ent.DocumentClassification, error) {
	createdEntity, err := tx.DocumentClassification.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetColor(entity.Color).
		Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create entity")
	}

	return createdEntity, nil
}

// UpdateDocumentClassification updates a document classification.
func (r *DocumentClassificationService) UpdateDocumentClassification(
	ctx context.Context, entity *ent.DocumentClassification,
) (*ent.DocumentClassification, error) {
	updatedEntity := new(ent.DocumentClassification)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateDocumentClassificationEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *DocumentClassificationService) updateDocumentClassificationEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.DocumentClassification,
) (*ent.DocumentClassification, error) {
	current, err := tx.DocumentClassification.Get(ctx, entity.ID)
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
	updateOp := tx.DocumentClassification.
		UpdateOneID(entity.ID).
		SetCode(entity.Code).
		SetDescription(entity.Description).
		SetStatus(entity.Status).
		SetColor(entity.Color).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, eris.Wrap(err, "failed to update entity")
	}

	return updatedEntity, nil
}
