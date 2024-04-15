package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/user"
	"github.com/google/uuid"
)

type UserService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewUserService creates a new accessorial charge service.
func NewUserService(s *api.Server) *UserService {
	return &UserService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetAuthenticatedUser returns the user if the user ID is correct.
func (r *UserService) GetAuthenticatedUser(ctx context.Context, userID uuid.UUID) (*ent.User, error) {
	u, err := r.Client.User.
		Query().
		Where(user.IDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return u, nil
}
