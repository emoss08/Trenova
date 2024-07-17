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

// CommodityService handles business logic for Commodity
type CommodityService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewCommodityService creates a new instance of CommodityService
func NewCommodityService(s *server.Server) *CommodityService {
	return &CommodityService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying Commodity
type CommodityQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s CommodityService) filterQuery(q *bun.SelectQuery, f *CommodityQueryFilter) *bun.SelectQuery {
	q = q.Where("com.organization_id = ?", f.OrganizationID).
		Where("com.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("com.name = ? OR com.name ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN com.name = ? THEN 0 ELSE 1 END", f.Query).
		Order("com.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all Commodity based on the provided filter
func (s CommodityService) GetAll(ctx context.Context, filter *CommodityQueryFilter) ([]*models.Commodity, int, error) {
	var entities []*models.Commodity

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Commodity")
		return nil, 0, fmt.Errorf("failed to fetch Commodity: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single Commodity by ID
func (s CommodityService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.Commodity, error) {
	entity := new(models.Commodity)
	err := s.db.NewSelect().
		Model(entity).
		Where("com.organization_id = ?", orgID).
		Where("com.business_unit_id = ?", buID).
		Where("com.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Commodity")
		return nil, fmt.Errorf("failed to fetch Commodity: %w", err)
	}

	return entity, nil
}

// Create creates a new Commodity
func (s CommodityService) Create(ctx context.Context, entity *models.Commodity) (*models.Commodity, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create Commodity")
		return nil, fmt.Errorf("failed to create Commodity: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing Commodity
func (s CommodityService) UpdateOne(ctx context.Context, entity *models.Commodity) (*models.Commodity, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update Commodity")
		return nil, fmt.Errorf("failed to update Commodity: %w", err)
	}

	return entity, nil
}
