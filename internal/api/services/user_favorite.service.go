package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/user"
	"github.com/emoss08/trenova/internal/ent/userfavorite"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type UserFavoriteService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

func NewUserFavoriteService(s *api.Server) *UserFavoriteService {
	return &UserFavoriteService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetUserFavorites returns all the favorites for a user along with the count of favorites.
func (r *UserFavoriteService) GetUserFavorites(ctx context.Context, userID uuid.UUID) ([]*ent.UserFavorite, int, error) {
	// count of how many favorites the user has
	count, countErr := r.Client.UserFavorite.
		Query().
		Where(userfavorite.HasUserWith(user.IDEQ(userID))).
		Count(ctx)
	if countErr != nil {
		return nil, 0, countErr
	}

	uf, err := r.Client.UserFavorite.
		Query().
		Where(userfavorite.HasUserWith(user.IDEQ(userID))).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return uf, count, nil
}

// AddUserFavorite creates a new user favorite.
func (r *UserFavoriteService) AddUserFavorite(ctx context.Context, userFavorite *ent.UserFavorite) (*ent.UserFavorite, error) {
	newUF, err := r.Client.UserFavorite.Create().
		SetPageLink(userFavorite.PageLink).
		SetUserID(userFavorite.UserID).
		SetBusinessUnitID(userFavorite.BusinessUnitID).
		SetOrganizationID(userFavorite.OrganizationID).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return newUF, nil
}

// UserFavoriteDelete deletes a user favorite.
func (r *UserFavoriteService) RemoveUserFavorite(ctx context.Context, userID uuid.UUID, pageLink string) error {
	_, err := r.Client.UserFavorite.Delete().
		Where(
			userfavorite.And(
				userfavorite.UserIDEQ(userID),
				userfavorite.PageLinkEQ(pageLink),
			),
		).
		Exec(ctx)
	if err != nil {
		return err
	}

	return err
}
