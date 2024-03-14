package services

import (
	"context"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/user"
	"golang.org/x/crypto/bcrypt"
)

// LoginOps is the service for login
type LoginOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewLoginOps returns a new instance of LoginOps
func NewLoginOps(ctx context.Context) *LoginOps {
	return &LoginOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// AuthenticateUser returns back the user if the username and password are correct
func (r *LoginOps) AuthenticateUser(username, password string) (*ent.User, error) {
	u, err := r.client.User.
		Query().
		Where(user.UsernameEQ(username)).
		Only(r.ctx)
	if err != nil {
		return nil, err
	}

	// Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	return u, nil
}
