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

// AccessorialChargeService handles business logic for AccessorialCharge
type AccessorialChargeService struct {
	common.AuditableService
	logger *config.ServerLogger
}

func NewAccessorialChargeService(s *server.Server) *AccessorialChargeService {
	return &AccessorialChargeService{
		AuditableService: common.AuditableService{
			DB:           s.DB,
			AuditService: s.AuditService,
		},
		logger: s.Logger,
	}
}

// AccessorialChargeQueryFilter defines the filter parameters for querying AccessorialCharge
type AccessorialChargeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	UserID         uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s AccessorialChargeService) filterQuery(q *bun.SelectQuery, f *AccessorialChargeQueryFilter) *bun.SelectQuery {
	q = q.Where("ac.organization_id = ?", f.OrganizationID).
		Where("ac.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("ac.code = ? OR ac.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN ac.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("ac.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all AccessorialCharge based on the provided filter
func (s AccessorialChargeService) GetAll(ctx context.Context, filter *AccessorialChargeQueryFilter) ([]*models.AccessorialCharge, int, error) {
	var entities []*models.AccessorialCharge

	q := s.DB.NewSelect().Model(&entities)
	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch AccessorialCharge")
		return nil, 0, fmt.Errorf("failed to fetch AccessorialCharge: %w", err)
	}

	return entities, count, nil
}

func (s AccessorialChargeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.AccessorialCharge, error) {
	entity := new(models.AccessorialCharge)
	err := s.GetByID(ctx, id, orgID, buID, entity)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch AccessorialCharge")
		return nil, fmt.Errorf("failed to fetch AccessorialCharge: %w", err)
	}

	return entity, nil
}

// Create creates a new AccessorialCharge
func (s AccessorialChargeService) Create(ctx context.Context, entity *models.AccessorialCharge, userID uuid.UUID) (*models.AccessorialCharge, error) {
	_, err := s.CreateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create AccessorialCharge")
		return nil, fmt.Errorf("failed to create AccessorialCharge: %w", err)
	}

	return entity, nil
}

func (s AccessorialChargeService) UpdateOne(ctx context.Context, entity *models.AccessorialCharge, userID uuid.UUID) (*models.AccessorialCharge, error) {
	err := s.UpdateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update AccessorialCharge")
		return nil, fmt.Errorf("failed to update AccessorialCharge: %w", err)
	}

	return entity, nil
}
