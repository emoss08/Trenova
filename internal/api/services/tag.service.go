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
	"github.com/emoss08/trenova/internal/api/common"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TagService struct {
	common.AuditableService
	logger *config.ServerLogger
}

func NewTagService(s *server.Server) *TagService {
	return &TagService{
		AuditableService: common.AuditableService{
			DB:           s.DB,
			AuditService: s.AuditService,
		},
		logger: s.Logger,
	}
}

func (s *TagService) Get(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*models.Tag, int, error) {
	var tags []*models.Tag
	cnt, err := s.DB.NewSelect().
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

func (s *TagService) Create(ctx context.Context, entity *models.Tag) (*models.Tag, error) {
	entity = &models.Tag{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		Status:         property.StatusActive,
		Name:           entity.Name,
		Color:          entity.Color,
	}

	err := s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
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

func (s *TagService) UpdateOne(ctx context.Context, entity *models.Tag) (*models.Tag, error) {
	err := s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}
