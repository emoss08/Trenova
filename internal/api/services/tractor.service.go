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
	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// TractorService handles business logic for Tractor
type TractorService struct {
	common.AuditableService
	logger *config.ServerLogger
}

// NewTractorService creates a new instance of TractorService
func NewTractorService(s *server.Server) *TractorService {
	return &TractorService{
		AuditableService: common.AuditableService{
			DB:           s.DB,
			AuditService: s.AuditService,
		},
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying Tractor
type TractorQueryFilter struct {
	Query               string
	OrganizationID      uuid.UUID
	BusinessUnitID      uuid.UUID
	FleetCodeID         uuid.UUID
	Status              string
	ExpandWorkerDetails bool
	ExpandEquipDetails  bool
	Limit               int
	Offset              int
}

// filterQuery applies filters to the query
func (s *TractorService) filterQuery(q *bun.SelectQuery, f *TractorQueryFilter) *bun.SelectQuery {
	q = q.Where("tr.organization_id = ?", f.OrganizationID).
		Where("tr.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("tr.code = ? OR tr.code ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	if f.ExpandWorkerDetails {
		q = q.Relation("PrimaryWorker").
			Relation("PrimaryWorker.WorkerProfile").
			Relation("SecondaryWorker").
			Relation("SecondaryWorker.WorkerProfile")
	}

	if f.ExpandEquipDetails {
		q = q.Relation("EquipmentType").
			Relation("EquipmentManufacturer")
	}

	if f.Status != "" {
		q = q.Where("tr.status = ?", f.Status)
	}

	if f.FleetCodeID != uuid.Nil {
		q = q.Where("tr.fleet_code_id = ?", f.FleetCodeID)
	}

	q = q.OrderExpr("CASE WHEN tr.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("tr.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all Tractor based on the provided filter
func (s *TractorService) GetAll(ctx context.Context, filter *TractorQueryFilter) ([]*models.Tractor, int, error) {
	var entities []*models.Tractor

	q := s.DB.NewSelect().Model(&entities)
	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Tractor")
		return nil, 0, fmt.Errorf("failed to fetch Tractor: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single Tractor by ID
func (s *TractorService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.Tractor, error) {
	entity := new(models.Tractor)
	err := s.GetByID(ctx, id, orgID, buID, entity)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Tractor")
		return nil, fmt.Errorf("failed to fetch Tractor: %w", err)
	}

	return entity, nil
}

// Create creates a new Tractor
func (s *TractorService) Create(ctx context.Context, entity *models.Tractor, userID uuid.UUID) (*models.Tractor, error) {
	_, err := s.CreateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create Tractor")
		return nil, fmt.Errorf("failed to create Tractor: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing Tractor
func (s *TractorService) UpdateOne(ctx context.Context, entity *models.Tractor, userID uuid.UUID) (*models.Tractor, error) {
	err := s.UpdateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update Tractor")
		return nil, fmt.Errorf("failed to update Tractor: %w", err)
	}

	return entity, nil
}

// AssignmentQueryFilter defines the filter parameters for querying Active Assignments
type AssignmentQueryFilter struct {
	OrganizationID        uuid.UUID
	BusinessUnitID        uuid.UUID
	TractorID             uuid.UUID
	Status                property.AssignmentStatus
	ExpandShipmentDetails bool
}

// filterAssignmentQuery applies filters to the get active assignments query.
func (s *TractorService) filterAssignmentQuery(q *bun.SelectQuery, f *AssignmentQueryFilter) *bun.SelectQuery {
	q = q.Where("ta.organization_id = ?", f.OrganizationID).
		Where("ta.business_unit_id = ?", f.BusinessUnitID)

	if f.ExpandShipmentDetails {
		q = q.Relation("Shipment").
			Relation("ShipmentMove")
	}

	if f.Status != "" {
		q = q.Where("ta.status = ?", f.Status)
	}

	if f.TractorID != uuid.Nil {
		q = q.Where("ta.tractor_id = ?", f.TractorID)
	}

	return q
}

func (s *TractorService) GetActiveAssignments(ctx context.Context, filter *AssignmentQueryFilter) ([]models.TractorAssignment, error) {
	var assignments []models.TractorAssignment

	q := s.DB.NewSelect().
		Model(&assignments)

	q = s.filterAssignmentQuery(q, filter)

	if err := q.Scan(ctx); err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch active assignments")
		return nil, fmt.Errorf("failed to fetch active assignments: %w", err)
	}

	return assignments, nil
}
