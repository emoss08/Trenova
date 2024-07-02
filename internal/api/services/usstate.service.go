package services

import (
	"context"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type USStateService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

func NewUSStateService(s *server.Server) *USStateService {
	return &USStateService{
		db:     s.DB,
		logger: s.Logger,
	}
}

func (s *USStateService) GetUSStates(ctx context.Context) ([]*models.UsState, int, error) {
	var states []*models.UsState
	count, err := s.db.NewSelect().
		Model(&states).
		Order("us.created_at DESC").
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return states, count, nil
}
