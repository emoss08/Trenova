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

// CommentTypeService handles business logic for CommentType
type CommentTypeService struct {
	common.AuditableService
	logger *config.ServerLogger
}

// NewCommentTypeService creates a new instance of CommentTypeService
func NewCommentTypeService(s *server.Server) *CommentTypeService {
	return &CommentTypeService{
		AuditableService: common.AuditableService{
			DB:           s.DB,
			AuditService: s.AuditService,
		},
		logger: s.Logger,
	}
}

// CommentTypeQueryFilter defines the filter parameters for querying CommentType
type CommentTypeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s CommentTypeService) filterQuery(q *bun.SelectQuery, f *CommentTypeQueryFilter) *bun.SelectQuery {
	q = q.Where("ct.organization_id = ?", f.OrganizationID).
		Where("ct.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("ct.name = ? OR ct.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN ct.name = ? THEN 0 ELSE 1 END", f.Query).
		Order("ct.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all CommentType based on the provided filter
func (s CommentTypeService) GetAll(ctx context.Context, filter *CommentTypeQueryFilter) ([]*models.CommentType, int, error) {
	var entities []*models.CommentType

	q := s.DB.NewSelect().Model(&entities)
	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch CommentType")
		return nil, 0, fmt.Errorf("failed to fetch CommentType: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single CommentType by ID
func (s CommentTypeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.CommentType, error) {
	entity := new(models.CommentType)
	err := s.GetByID(ctx, id, orgID, buID, entity)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch CommentType")
		return nil, fmt.Errorf("failed to fetch CommentType: %w", err)
	}

	return entity, nil
}

// Create creates a new CommentType
func (s CommentTypeService) Create(ctx context.Context, entity *models.CommentType, userID uuid.UUID) (*models.CommentType, error) {
	_, err := s.CreateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create CommentType")
		return nil, fmt.Errorf("failed to create CommentType: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing CommentType
func (s CommentTypeService) UpdateOne(ctx context.Context, entity *models.CommentType, userID uuid.UUID) (*models.CommentType, error) {
	err := s.UpdateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update CommentType")
		return nil, fmt.Errorf("failed to update CommentType: %w", err)
	}

	return entity, nil
}
