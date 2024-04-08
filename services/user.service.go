package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/user"
	"github.com/google/uuid"
)

// UserOps is the service for user.
type UserOps struct {
	client *ent.Client
}

// NewUserOps creates a new user service.
func NewUserOps() *UserOps {
	return &UserOps{
		client: database.GetClient(),
	}
}

// GetAuthenticatedUser returns the user if the user ID is correct.
func (r *UserOps) GetAuthenticatedUser(ctx context.Context, userID uuid.UUID) (*ent.User, error) {
	u, err := r.client.User.
		Query().
		Where(user.IDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	return u, nil
}
