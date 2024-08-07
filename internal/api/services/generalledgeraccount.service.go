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

// GeneralLedgerAccountService handles business logic for GeneralLedgerAccount
type GeneralLedgerAccountService struct {
	common.AuditableService
	logger *config.ServerLogger
}

// NewGeneralLedgerAccountService creates a new instance of GeneralLedgerAccountService
func NewGeneralLedgerAccountService(s *server.Server) *GeneralLedgerAccountService {
	return &GeneralLedgerAccountService{
		AuditableService: common.AuditableService{
			DB:           s.DB,
			AuditService: s.AuditService,
		},
		logger: s.Logger,
	}
}

// GeneralLedgerAccountQueryFilter defines the filter parameters for querying GeneralLedgerAccount
type GeneralLedgerAccountQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s GeneralLedgerAccountService) filterQuery(q *bun.SelectQuery, f *GeneralLedgerAccountQueryFilter) *bun.SelectQuery {
	q = q.Where("gla.organization_id = ?", f.OrganizationID).
		Where("gla.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("gla.account_number = ? OR gla.account_number ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN gla.account_number = ? THEN 0 ELSE 1 END", f.Query).
		Order("gla.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all GeneralLedgerAccount based on the provided filter
func (s GeneralLedgerAccountService) GetAll(ctx context.Context, filter *GeneralLedgerAccountQueryFilter) ([]*models.GeneralLedgerAccount, int, error) {
	var entities []*models.GeneralLedgerAccount

	q := s.DB.NewSelect().Model(&entities)
	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch GeneralLedgerAccount")
		return nil, 0, fmt.Errorf("failed to fetch GeneralLedgerAccount: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single GeneralLedgerAccount by ID
func (s GeneralLedgerAccountService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.GeneralLedgerAccount, error) {
	entity := new(models.GeneralLedgerAccount)
	err := s.GetByID(ctx, id, orgID, buID, entity)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch GeneralLedgerAccount")
		return nil, fmt.Errorf("failed to fetch GeneralLedgerAccount: %w", err)
	}

	return entity, nil
}

// Create creates a new GeneralLedgerAccount
func (s GeneralLedgerAccountService) Create(ctx context.Context, entity *models.GeneralLedgerAccount, userID uuid.UUID) (*models.GeneralLedgerAccount, error) {
	_, err := s.CreateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create GeneralLedgerAccount")
		return nil, fmt.Errorf("failed to create GeneralLedgerAccount: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing GeneralLedgerAccount
func (s GeneralLedgerAccountService) UpdateOne(ctx context.Context, entity *models.GeneralLedgerAccount, userID uuid.UUID) (*models.GeneralLedgerAccount, error) {
	err := s.UpdateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update GeneralLedgerAccount")
		return nil, fmt.Errorf("failed to update GeneralLedgerAccount: %w", err)
	}

	return entity, nil
}
