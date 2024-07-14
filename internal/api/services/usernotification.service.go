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
