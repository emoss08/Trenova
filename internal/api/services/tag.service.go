package services

import (
	"context"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type TagService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

func NewTagService(s *server.Server) *TagService {
	return &TagService{
		db:     s.DB,
		logger: s.Logger,
	}
}

func (s *TagService) GetTags(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*models.Tag, int, error) {
	var tags []*models.Tag
	cnt, err := s.db.NewSelect().
		Model(&tags).
		Where("t.organization_id = ?", orgID).
		Where("t.business_unit_id = ?", buID).
		Order("t.created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return tags, cnt, nil
}

func (s *TagService) CreateTag(ctx context.Context, entity *models.Tag) (*models.Tag, error) {
	entity = &models.Tag{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		Status:         property.StatusActive,
		Name:           entity.Name,
		Color:          entity.Color,
	}

	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *TagService) UpdateTag(ctx context.Context, entity *models.Tag) (*models.Tag, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}
