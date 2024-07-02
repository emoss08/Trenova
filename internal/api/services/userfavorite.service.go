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

func (s *UserFavoriteService) GetUserFavorites(ctx context.Context, userID uuid.UUID) ([]*models.UserFavorite, int, error) {
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

func (s *UserFavoriteService) AddUserFavorite(ctx context.Context, uf *models.UserFavorite) error {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Model(uf).Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return err
}

func (s *UserFavoriteService) DeleteUserFavorite(ctx context.Context, uf *models.UserFavorite) error {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model(uf).
			Where("user_id = ?", uf.UserID).
			Where("page_link = ?", uf.PageLink).
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
