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

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// QualifierCodeService handles business logic for QualifierCode
type QualifierCodeService struct {
	db     *bun.DB
	logger *config.ServerLogger
}

// NewQualifierCodeService creates a new instance of QualifierCodeService
func NewQualifierCodeService(s *server.Server) *QualifierCodeService {
	return &QualifierCodeService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying QualifierCode
type QualifierCodeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s QualifierCodeService) filterQuery(q *bun.SelectQuery, f *QualifierCodeQueryFilter) *bun.SelectQuery {
	q = q.Where("qc.organization_id = ?", f.OrganizationID).
		Where("qc.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("qc.code = ? OR qc.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN qc.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("qc.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all QualifierCode based on the provided filter
func (s QualifierCodeService) GetAll(ctx context.Context, filter *QualifierCodeQueryFilter) ([]*models.QualifierCode, int, error) {
	var entities []*models.QualifierCode

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch QualifierCode")
		return nil, 0, fmt.Errorf("failed to fetch QualifierCode: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single QualifierCode by ID
func (s QualifierCodeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.QualifierCode, error) {
	entity := new(models.QualifierCode)
	err := s.db.NewSelect().
		Model(entity).
		Where("qc.organization_id = ?", orgID).
		Where("qc.business_unit_id = ?", buID).
		Where("qc.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch QualifierCode")
		return nil, fmt.Errorf("failed to fetch QualifierCode: %w", err)
	}

	return entity, nil
}

// Create creates a new QualifierCode
func (s QualifierCodeService) Create(ctx context.Context, entity *models.QualifierCode) (*models.QualifierCode, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create QualifierCode")
		return nil, fmt.Errorf("failed to create QualifierCode: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing QualifierCode
func (s QualifierCodeService) UpdateOne(ctx context.Context, entity *models.QualifierCode) (*models.QualifierCode, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update QualifierCode")
		return nil, fmt.Errorf("failed to update QualifierCode: %w", err)
	}

	return entity, nil
}
