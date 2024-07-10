package services

import (
	"context"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type UserTaskService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

func NewUserTaskService(s *server.Server) *UserTaskService {
	return &UserTaskService{
		db:     s.DB,
		logger: s.Logger,
	}
}

func (s UserTaskService) GetTasksByUserID(ctx context.Context, userID, buID, orgID uuid.UUID) ([]*models.UserTask, int, error) {
	var entities []*models.UserTask
	cnt, err := s.db.NewSelect().
		Model(&entities).
		Where("ut.user_id = ?", userID).
		Where("ut.business_unit_id = ?", buID).
		Where("ut.organization_id = ?", orgID).
		Order("ut.created_at DESC").
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, cnt, nil
}
