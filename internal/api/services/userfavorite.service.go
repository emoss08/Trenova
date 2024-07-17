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
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type UserFavoriteService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

func NewUserFavoriteService(s *server.Server) *UserFavoriteService {
	return &UserFavoriteService{
		db:     s.DB,
		logger: s.Logger,
	}
}

func (s UserFavoriteService) GetUserFavorites(ctx context.Context, userID uuid.UUID) ([]*models.UserFavorite, int, error) {
	var uf []*models.UserFavorite

	count, err := s.db.NewSelect().
		Model(&uf).
		Where("user_id = ?", userID).
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return uf, count, nil
}

func (s UserFavoriteService) AddUserFavorite(ctx context.Context, entity *models.UserFavorite) error {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(entity).Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return err
}

func (s UserFavoriteService) DeleteUserFavorite(ctx context.Context, entity *models.UserFavorite) error {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model(entity).
			Where("user_id = ?", entity.UserID).
			Where("page_link = ?", entity.PageLink).
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return err
}
