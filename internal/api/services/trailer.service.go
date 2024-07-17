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
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

// TrailerService handles business logic for Trailer
type TrailerService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewTrailerService creates a new instance of TrailerService
func NewTrailerService(s *server.Server) *TrailerService {
	return &TrailerService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying Trailer
type TrailerQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s TrailerService) filterQuery(q *bun.SelectQuery, f *TrailerQueryFilter) *bun.SelectQuery {
	q = q.Where("tr.organization_id = ?", f.OrganizationID).
		Where("tr.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("tr.code = ? OR tr.code ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN tr.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("tr.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all Trailer based on the provided filter
func (s TrailerService) GetAll(ctx context.Context, filter *TrailerQueryFilter) ([]*models.Trailer, int, error) {
	var entities []*models.Trailer

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Trailer")
		return nil, 0, fmt.Errorf("failed to fetch Trailer: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single Trailer by ID
func (s TrailerService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.Trailer, error) {
	entity := new(models.Trailer)
	err := s.db.NewSelect().
		Model(entity).
		Where("tr.organization_id = ?", orgID).
		Where("tr.business_unit_id = ?", buID).
		Where("tr.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Trailer")
		return nil, fmt.Errorf("failed to fetch Trailer: %w", err)
	}

	return entity, nil
}

// Create creates a new Trailer
func (s TrailerService) Create(ctx context.Context, entity *models.Trailer) (*models.Trailer, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create Trailer")
		return nil, fmt.Errorf("failed to create Trailer: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing Trailer
func (s TrailerService) UpdateOne(ctx context.Context, entity *models.Trailer) (*models.Trailer, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update Trailer")
		return nil, fmt.Errorf("failed to update Trailer: %w", err)
	}

	return entity, nil
}
