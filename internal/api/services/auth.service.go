package services

import (
	"context"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type AuthenticationService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

func NewAuthenticationService(s *server.Server) *AuthenticationService {
	return &AuthenticationService{
		db:     s.DB,
		logger: s.Logger,
	}
}

func (s AuthenticationService) CheckEmail(ctx context.Context, emailAddress string) (*types.CheckEmailResponse, error) {
	user := new(models.User)

	if err := s.db.NewSelect().Model(user).Where("email = ?", emailAddress).Scan(ctx); err != nil {
		return &types.CheckEmailResponse{
			Exists:  false,
			Message: "Email address does not exist. Please Try again.",
		}, err
	}

	return &types.CheckEmailResponse{
		Exists:        user.ID != uuid.Nil,
		AccountStatus: user.Status,
		Message:       "Email address exists",
	}, nil
}

func (s AuthenticationService) AuthenticateUser(ctx context.Context, emailAddress, password string) (*models.User, string, error) {
	user := new(models.User)

	if err := s.db.NewSelect().Model(user).Where("email = ?", emailAddress).Scan(ctx); err != nil {
		s.logger.Error().Err(err).Msg("error getting user")
		return nil, "", err
	}

	if err := user.VerifyPassword(password); err != nil {
		return nil, "", err
	}

	return user, "", nil
}
