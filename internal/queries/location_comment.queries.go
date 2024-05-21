package queries

import (
	"context"

	"github.com/emoss08/trenova/internal/api/services/types"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/location"
	"github.com/emoss08/trenova/internal/ent/locationcomment"
	"github.com/google/uuid"
)

// SyncLocationComments synchronizes location comments.
//
// Parameters:
//
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *LocationUpdateRequest: The location update request containing the comment details.
//   - updatedEntity *ent.Location: The updated location entity.
//
// Returns:
//   - error: An error object that indicates why the synchronization failed, nil if no error occurred.
func (r *QueryService) SyncLocationComments(ctx context.Context, tx *ent.Tx, entity *types.LocationUpdateRequest, updatedEntity *ent.Location) error {
	existingComments, err := tx.Location.QueryComments(updatedEntity).Where(
		locationcomment.HasLocationWith(location.IDEQ(entity.ID)),
	).All(ctx)
	if err != nil {
		return err
	}

	// Delete unmatched comments
	if err = r.deleteUnmatchedLocationComments(ctx, tx, entity, existingComments); err != nil {
		return err
	}

	// Update or create new comments
	return r.updateOrCreateLocationComments(ctx, tx, entity)
}

// deleteUnmatchedLocationComments deletes location contacts that are not present in the update request.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *LocationUpdateRequest: The location update request containing the comments details.
//   - existingComments []*ent.LocationComment: A slice of existing location comment.
//
// Returns:
//   - error: An error object that indicates why the deletion failed, nil if no error occurred.
func (r *QueryService) deleteUnmatchedLocationComments(
	ctx context.Context, tx *ent.Tx, entity *types.LocationUpdateRequest, existingComments []*ent.LocationComment,
) error {
	commentPresent := make(map[uuid.UUID]bool)
	for _, comment := range entity.Comments {
		if comment.ID != uuid.Nil {
			commentPresent[comment.ID] = true
		}
	}

	for _, existingComment := range existingComments {
		if !commentPresent[existingComment.ID] {
			if err := tx.LocationComment.DeleteOneID(existingComment.ID).Exec(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// updateOrCreateLocationComments updates existing location comment or creates new ones.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - entity *LocationUpdateRequest: The location update request containing the comments details.
//
// Returns:
//   - error: An error object that indicates why the update or creation failed, nil if no error occurred.
func (r *QueryService) updateOrCreateLocationComments(ctx context.Context, tx *ent.Tx, entity *types.LocationUpdateRequest) error {
	// Builders for new comments
	builders := make([]*ent.LocationCommentCreate, 0, len(entity.Comments))

	for _, comment := range entity.Comments {
		if comment.ID != uuid.Nil {
			if err := tx.LocationComment.UpdateOneID(comment.ID).
				SetComment(comment.Comment).
				SetUserID(comment.UserID).
				SetCommentTypeID(comment.CommentTypeID).
				Exec(ctx); err != nil {
				return err
			}
		} else {
			builder := tx.LocationComment.Create().
				SetLocationID(entity.ID).
				SetComment(comment.Comment).
				SetBusinessUnitID(entity.BusinessUnitID).
				SetOrganizationID(entity.OrganizationID).
				SetUserID(comment.UserID).
				SetCommentTypeID(comment.CommentTypeID)
			builders = append(builders, builder)
		}
	}

	// Create new comments in bulk
	if len(builders) > 0 {
		if err := tx.LocationComment.CreateBulk(builders...).Exec(ctx); err != nil {
			r.Logger.Err(err).Msg("Error creating location comments")
			return err
		}
	}

	return nil
}

// CreateLocationComments creates location comments in bulk.
//
// Parameters:
//
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - tx *ent.Tx: The database transaction to use.
//   - locationID uuid.UUID: The identifier of the location to associate the contacts with.
//   - entity *LocationRequest: The location request containing the contact details.
//
// Returns:
//   - error: An error object that indicates why the creation failed, nil if no error occurred.
func (r *QueryService) CreateLocationComments(ctx context.Context, tx *ent.Tx, locationID uuid.UUID, entity *types.LocationRequest) error {
	builders := make([]*ent.LocationCommentCreate, 0, len(entity.Comments))

	for _, comment := range entity.Comments {
		builder := tx.LocationComment.Create().
			SetLocationID(locationID).
			SetComment(comment.Comment).
			SetBusinessUnitID(entity.BusinessUnitID).
			SetOrganizationID(entity.OrganizationID).
			SetUserID(comment.UserID).
			SetCommentTypeID(comment.CommentTypeID)
		builders = append(builders, builder)
	}

	if err := tx.LocationComment.CreateBulk(builders...).Exec(ctx); err != nil {
		r.Logger.Err(err).Msg("Error creating location comments")
		return err
	}

	return nil
}
