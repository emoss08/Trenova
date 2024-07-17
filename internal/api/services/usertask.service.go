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
