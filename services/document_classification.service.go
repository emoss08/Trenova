package services

import (
	"context"

	"github.com/emoss08/trenova/ent/documentclassification"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type DocumentClassificationOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewDocumentClassificationOps creates a new document classification service.
func NewDocumentClassificationOps(ctx context.Context) *DocumentClassificationOps {
	return &DocumentClassificationOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetDocumentClassification gets the document classification for an organization.
func (r *DocumentClassificationOps) GetDocumentClassification(limit, offset int, orgID, buID uuid.UUID) ([]*ent.DocumentClassification, int, error) {
	equipManuCount, countErr := r.client.DocumentClassification.Query().Where(
		documentclassification.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	documentClassifications, err := r.client.DocumentClassification.Query().
		Limit(limit).
		Offset(offset).
		Where(
			documentclassification.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return documentClassifications, equipManuCount, nil
}

// CreateDocumentClassification creates a new document classification.
func (r *DocumentClassificationOps) CreateDocumentClassification(newEquipMenu ent.DocumentClassification) (*ent.DocumentClassification, error) {
	documentClassification, err := r.client.DocumentClassification.Create().
		SetOrganizationID(newEquipMenu.OrganizationID).
		SetBusinessUnitID(newEquipMenu.BusinessUnitID).
		SetName(newEquipMenu.Name).
		SetDescription(newEquipMenu.Description).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return documentClassification, nil
}

// UpdateDocumentClassification updates a document classification.
func (r *DocumentClassificationOps) UpdateDocumentClassification(documentClassification ent.DocumentClassification) (*ent.DocumentClassification, error) {
	// Start building the update operation
	updateOp := r.client.DocumentClassification.
		UpdateOneID(documentClassification.ID).
		SetName(documentClassification.Name).
		SetDescription(documentClassification.Description)

	// Execute the update operation
	updateDocClass, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updateDocClass, nil
}
