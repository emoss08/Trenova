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

// DocumentClassificationService handles business logic for DocumentClassification
type DocumentClassificationService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewDocumentClassificationService creates a new instance of DocumentClassificationService
func NewDocumentClassificationService(s *server.Server) *DocumentClassificationService {
	return &DocumentClassificationService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying DocumentClassification
type DocumentClassificationQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s DocumentClassificationService) filterQuery(q *bun.SelectQuery, f *DocumentClassificationQueryFilter) *bun.SelectQuery {
	q = q.Where("dc.organization_id = ?", f.OrganizationID).
		Where("dc.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("dc.code = ? OR dc.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN dc.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("dc.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all DocumentClassification based on the provided filter
func (s DocumentClassificationService) GetAll(ctx context.Context, filter *DocumentClassificationQueryFilter) ([]*models.DocumentClassification, int, error) {
	var entities []*models.DocumentClassification

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch DocumentClassification")
		return nil, 0, fmt.Errorf("failed to fetch DocumentClassification: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single DocumentClassification by ID
func (s DocumentClassificationService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.DocumentClassification, error) {
	entity := new(models.DocumentClassification)
	err := s.db.NewSelect().
		Model(entity).
		Where("dc.organization_id = ?", orgID).
		Where("dc.business_unit_id = ?", buID).
		Where("dc.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch DocumentClassification")
		return nil, fmt.Errorf("failed to fetch DocumentClassification: %w", err)
	}

	return entity, nil
}

// Create creates a new DocumentClassification
func (s DocumentClassificationService) Create(ctx context.Context, entity *models.DocumentClassification) (*models.DocumentClassification, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create DocumentClassification")
		return nil, fmt.Errorf("failed to create DocumentClassification: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing DocumentClassification
func (s DocumentClassificationService) UpdateOne(ctx context.Context, entity *models.DocumentClassification) (*models.DocumentClassification, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update DocumentClassification")
		return nil, fmt.Errorf("failed to update DocumentClassification: %w", err)
	}

	return entity, nil
}
