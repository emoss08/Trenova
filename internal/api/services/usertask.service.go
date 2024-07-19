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
