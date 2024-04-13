package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/user"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

type AuthenticationService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

func NewAuthenticationService(s *api.Server) *AuthenticationService {
	return &AuthenticationService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// AuthenticateUser returns back the user if the username and password are correct.
func (r *AuthenticationService) AuthenticateUser(ctx context.Context, username, password string) (*ent.User, error) {
	u, err := r.Client.User.
		Query().
		Where(user.UsernameEQ(username)).
		Only(ctx)
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
