// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
