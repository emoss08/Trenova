// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

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
