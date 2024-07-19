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
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type UserNotificationService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

func NewUserNotificationService(s *server.Server) *UserNotificationService {
	return &UserNotificationService{
		db:     s.DB,
		logger: s.Logger,
	}
}

func (s UserNotificationService) GetUserNotifications(ctx context.Context, limit int, userID, buID, orgID uuid.UUID) ([]*models.UserNotification, int, error) {
	var un []*models.UserNotification

	count, err := s.db.NewSelect().
		Model(&un).
		Where("user_id = ?", userID).
		Where("business_unit_id = ?", buID).
		Where("organization_id = ?", orgID).
		Where("is_read = ?", false).
		Order("created_at DESC").
		Limit(limit).
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return un, count, nil
}

func (s UserNotificationService) MarkNotificationsAsRead(ctx context.Context, orgID, buID, userID uuid.UUID) error {
	un := new(models.UserNotification)

	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().
			Model(un).
			Set("is_read = ?", true).
			Where("user_id = ?", userID).
			Where("business_unit_id = ?", buID).
			Where("organization_id = ?", orgID).
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
}

func (s UserNotificationService) CreateUserNotification(ctx context.Context, orgID, buID, userID uuid.UUID, title, description, actionURL string) error {
	return s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		un := &models.UserNotification{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			UserID:         userID,
			Title:          title,
			Description:    description,
			ActionURL:      actionURL,
		}

		if _, err := tx.NewInsert().Model(un).Exec(ctx); err != nil {
			s.logger.Error().Err(err).Msg("failed to create user notification")
			return err
		}

		return nil
	})
}
