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

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UserFavoriteService struct {
	db     *bun.DB
	logger *config.ServerLogger
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
