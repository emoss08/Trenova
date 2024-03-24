package services

import (
	"context"

	"github.com/emoss08/trenova/ent/qualifiercode"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type QualifierCodeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewQualifierCodeOps creates a new qualifier code service.
func NewQualifierCodeOps(ctx context.Context) *QualifierCodeOps {
	return &QualifierCodeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetQualifierCodes gets the qualifier code for an organization.
func (r *QualifierCodeOps) GetQualifierCodes(limit, offset int, orgID, buID uuid.UUID) ([]*ent.QualifierCode, int, error) {
	qualifierCodeCount, countErr := r.client.QualifierCode.Query().Where(
		qualifiercode.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	qualifierCodes, err := r.client.QualifierCode.Query().
		Limit(limit).
		Offset(offset).
		Where(
			qualifiercode.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return qualifierCodes, qualifierCodeCount, nil
}

// CreateQualifierCode creates a new qualifier code.
func (r *QualifierCodeOps) CreateQualifierCode(newQualifierCode ent.QualifierCode) (*ent.QualifierCode, error) {
	qualifierCode, err := r.client.QualifierCode.Create().
		SetOrganizationID(newQualifierCode.OrganizationID).
		SetBusinessUnitID(newQualifierCode.BusinessUnitID).
		SetStatus(newQualifierCode.Status).
		SetCode(newQualifierCode.Code).
		SetDescription(newQualifierCode.Description).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return qualifierCode, nil
}

// UpdateQualifierCode updates a qualifier code.
func (r *QualifierCodeOps) UpdateQualifierCode(qualifierCode ent.QualifierCode) (*ent.QualifierCode, error) {
	// Start building the update operation
	updateOp := r.client.QualifierCode.UpdateOneID(qualifierCode.ID).
		SetStatus(qualifierCode.Status).
		SetCode(qualifierCode.Code).
		SetDescription(qualifierCode.Description)

	// Execute the update operation
	updateQualifierCode, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updateQualifierCode, nil
}
