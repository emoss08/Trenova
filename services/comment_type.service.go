package services

import (
	"context"

	"github.com/emoss08/trenova/ent/commenttype"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type CommentTypeOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewCommentTypeOps creates a new comment type service.
func NewCommentTypeOps(ctx context.Context) *CommentTypeOps {
	return &CommentTypeOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetCommentTypes gets the comment type for an organization.
func (r *CommentTypeOps) GetCommentTypes(limit, offset int, orgID, buID uuid.UUID) ([]*ent.CommentType, int, error) {
	commentTypeCount, countErr := r.client.CommentType.Query().Where(
		commenttype.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	commentTypes, err := r.client.CommentType.Query().
		Limit(limit).
		Offset(offset).
		Where(
			commenttype.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return commentTypes, commentTypeCount, nil
}

// CreateCommentType creates a new comment type.
func (r *CommentTypeOps) CreateCommentType(newCommentType ent.CommentType) (*ent.CommentType, error) {
	commentType, err := r.client.CommentType.Create().
		SetOrganizationID(newCommentType.OrganizationID).
		SetBusinessUnitID(newCommentType.BusinessUnitID).
		SetStatus(newCommentType.Status).
		SetName(newCommentType.Name).
		SetSeverity(newCommentType.Severity).
		SetDescription(newCommentType.Description).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return commentType, nil
}

// UpdateCommentType updates a comment type.
func (r *CommentTypeOps) UpdateCommentType(commentType ent.CommentType) (*ent.CommentType, error) {
	// Start building the update operation
	updateOp := r.client.CommentType.UpdateOneID(commentType.ID).
		SetStatus(commentType.Status).
		SetDescription(commentType.Description).
		SetName(commentType.Name).
		SetSeverity(commentType.Severity)

	// Execute the update operation
	updatedCommentType, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updatedCommentType, nil
}
