package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/user"
	"github.com/emoss08/trenova/ent/userfavorite"
	"github.com/google/uuid"
)

// UserFavoriteOps is the service for user.
type UserFavoriteOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewFavoriteOps creates a new user favorite service.
func NewUserFavoriteOps(ctx context.Context) *UserFavoriteOps {
	return &UserFavoriteOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetUserFavorites returns all the favorites for a user along with the count of favorites.
func (r *UserFavoriteOps) GetUserFavorites(userID uuid.UUID) ([]*ent.UserFavorite, int, error) {
	// count of how many favorites the user has
	count, countErr := r.client.UserFavorite.
		Query().
		Where(userfavorite.HasUserWith(user.IDEQ(userID))).
		Count(r.ctx)
	if countErr != nil {
		return nil, 0, countErr
	}

	uf, err := r.client.UserFavorite.
		Query().
		Where(userfavorite.HasUserWith(user.IDEQ(userID))).
		All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return uf, count, nil
}

// UserFavoriteCreate creates a new user favorite.
func (r *UserFavoriteOps) UserFavoriteCreate(userFavorite ent.UserFavorite) (*ent.UserFavorite, error) {
	newUF, err := r.client.UserFavorite.Create().
		SetPageLink(userFavorite.PageLink).
		SetUserID(userFavorite.UserID).
		SetBusinessUnitID(userFavorite.BusinessUnitID).
		SetOrganizationID(userFavorite.OrganizationID).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return newUF, nil
}

// UserFavoriteDelete deletes a user favorite.
func (r *UserFavoriteOps) UserFavoriteDelete(userID uuid.UUID, pageLink string) error {
	_, err := r.client.UserFavorite.Delete().
		Where(
			userfavorite.And(
				userfavorite.UserIDEQ(userID),
				userfavorite.PageLinkEQ(pageLink),
			),
		).
		Exec(r.ctx)
	if err != nil {
		return err
	}

	return err
}
