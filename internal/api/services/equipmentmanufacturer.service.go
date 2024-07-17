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

// EquipmentManufacturerService handles business logic for EquipmentManufacturer
type EquipmentManufacturerService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewEquipmentManufacturerService creates a new instance of EquipmentManufacturerService
func NewEquipmentManufacturerService(s *server.Server) *EquipmentManufacturerService {
	return &EquipmentManufacturerService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying EquipmentManufacturer
type EquipmentManufacturerQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s EquipmentManufacturerService) filterQuery(q *bun.SelectQuery, f *EquipmentManufacturerQueryFilter) *bun.SelectQuery {
	q = q.Where("em.organization_id = ?", f.OrganizationID).
		Where("em.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("em.name = ? OR em.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN em.name = ? THEN 0 ELSE 1 END", f.Query).
		Order("em.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all EquipmentManufacturer based on the provided filter
func (s EquipmentManufacturerService) GetAll(ctx context.Context, filter *EquipmentManufacturerQueryFilter) ([]*models.EquipmentManufacturer, int, error) {
	var entities []*models.EquipmentManufacturer

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch EquipmentManufacturer")
		return nil, 0, fmt.Errorf("failed to fetch EquipmentManufacturer: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single EquipmentManufacturer by ID
func (s EquipmentManufacturerService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.EquipmentManufacturer, error) {
	entity := new(models.EquipmentManufacturer)
	err := s.db.NewSelect().
		Model(entity).
		Where("em.organization_id = ?", orgID).
		Where("em.business_unit_id = ?", buID).
		Where("em.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch EquipmentManufacturer")
		return nil, fmt.Errorf("failed to fetch EquipmentManufacturer: %w", err)
	}

	return entity, nil
}

// Create creates a new EquipmentManufacturer
func (s EquipmentManufacturerService) Create(ctx context.Context, entity *models.EquipmentManufacturer) (*models.EquipmentManufacturer, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create EquipmentManufacturer")
		return nil, fmt.Errorf("failed to create EquipmentManufacturer: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing EquipmentManufacturer
func (s EquipmentManufacturerService) UpdateOne(ctx context.Context, entity *models.EquipmentManufacturer) (*models.EquipmentManufacturer, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update EquipmentManufacturer")
		return nil, fmt.Errorf("failed to update EquipmentManufacturer: %w", err)
	}

	return entity, nil
}
