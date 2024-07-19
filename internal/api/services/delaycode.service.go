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
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

// DelayCodeService handles business logic for DelayCode
type DelayCodeService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewDelayCodeService creates a new instance of DelayCodeService
func NewDelayCodeService(s *server.Server) *DelayCodeService {
	return &DelayCodeService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying DelayCode
type DelayCodeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s DelayCodeService) filterQuery(q *bun.SelectQuery, f *DelayCodeQueryFilter) *bun.SelectQuery {
	q = q.Where("dc.organization_id = ?", f.OrganizationID).
		Where("dc.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("dc.code = ? OR dc.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN dc.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("dc.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all DelayCode based on the provided filter
func (s DelayCodeService) GetAll(ctx context.Context, filter *DelayCodeQueryFilter) ([]*models.DelayCode, int, error) {
	var entities []*models.DelayCode

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch DelayCode")
		return nil, 0, fmt.Errorf("failed to fetch DelayCode: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single DelayCode by ID
func (s DelayCodeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.DelayCode, error) {
	entity := new(models.DelayCode)
	err := s.db.NewSelect().
		Model(entity).
		Where("dc.organization_id = ?", orgID).
		Where("dc.business_unit_id = ?", buID).
		Where("dc.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch DelayCode")
		return nil, fmt.Errorf("failed to fetch DelayCode: %w", err)
	}

	return entity, nil
}

// Create creates a new DelayCode
func (s DelayCodeService) Create(ctx context.Context, entity *models.DelayCode) (*models.DelayCode, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create DelayCode")
		return nil, fmt.Errorf("failed to create DelayCode: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing DelayCode
func (s DelayCodeService) UpdateOne(ctx context.Context, entity *models.DelayCode) (*models.DelayCode, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update DelayCode")
		return nil, fmt.Errorf("failed to update DelayCode: %w", err)
	}

	return entity, nil
}
