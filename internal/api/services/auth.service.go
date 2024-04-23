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

type CheckEmailRequest struct {
	EmailAddress string `json:"emailAddress"`
}

type CheckEmailResponse struct {
	Exists        bool        `json:"exists"`
	AccountStatus user.Status `json:"accountStatus"`
	Message       string      `json:"message"`
}

func (r *AuthenticationService) CheckEmail(ctx context.Context, emailAddress string) (*CheckEmailResponse, error) {
	u, err := r.Client.User.
		Query().
		Where(user.EmailEQ(emailAddress)).
		Only(ctx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to query user")
		return &CheckEmailResponse{
			Exists:  false,
			Message: "Email address does not exist!",
		}, nil
	}

	return &CheckEmailResponse{
		Exists:        true,
		AccountStatus: u.Status,
		Message:       "Email address exists",
	}, nil
}

// AuthenticateUser returns back the user if the username and password are correct.
func (r *AuthenticationService) AuthenticateUser(ctx context.Context, emailAddress, password string) (*ent.User, error) {
	u, err := r.Client.User.
		Query().
		Where(user.EmailEQ(emailAddress)).
		Only(ctx)
	if err != nil {
		r.Logger.Error().Err(err).Msg("Failed to query user")
		return nil, err
	}

	// Compare the provided password with the stored hash
	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	return u, nil
}
