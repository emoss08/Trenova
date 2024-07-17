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

func (s USStateService) GetUSStates(ctx context.Context) ([]*models.UsState, int, error) {
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
