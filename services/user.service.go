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
	ctx    context.Context
	client *ent.Client
}

// NewUserOps creates a new user service.
func NewUserOps(ctx context.Context) *UserOps {
	return &UserOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetAuthenticatedUser returns the user if the user ID is correct.
func (r *UserOps) GetAuthenticatedUser(userID uuid.UUID) (*ent.User, error) {
	u, err := r.client.User.
		Query().
		Where(user.IDEQ(userID)).
		Only(r.ctx)
	if err != nil {
		return nil, err
	}

	return u, nil
}
