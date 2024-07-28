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
	"github.com/emoss08/trenova/internal/api/common"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// TrailerService handles business logic for Trailer
type TrailerService struct {
	common.AuditableService
	logger *config.ServerLogger
}

// NewTrailerService creates a new instance of TrailerService
func NewTrailerService(s *server.Server) *TrailerService {
	return &TrailerService{
		AuditableService: common.AuditableService{
			DB:           s.DB,
			AuditService: s.AuditService,
		},
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

	q := s.DB.NewSelect().Model(&entities)
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
	err := s.GetByID(ctx, id, orgID, buID, entity)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Trailer")
		return nil, fmt.Errorf("failed to fetch Trailer: %w", err)
	}

	return entity, nil
}

// Create creates a new Trailer
func (s TrailerService) Create(ctx context.Context, entity *models.Trailer, userID uuid.UUID) (*models.Trailer, error) {
	_, err := s.CreateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create Trailer")
		return nil, fmt.Errorf("failed to create Trailer: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing Trailer
func (s TrailerService) UpdateOne(ctx context.Context, entity *models.Trailer, userID uuid.UUID) (*models.Trailer, error) {
	err := s.UpdateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update Trailer")
		return nil, fmt.Errorf("failed to update Trailer: %w", err)
	}

	return entity, nil
}
