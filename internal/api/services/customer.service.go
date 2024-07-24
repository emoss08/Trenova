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
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// CustomerService handles business logic for Customer
type CustomerService struct {
	db      *bun.DB
	logger  *config.ServerLogger
	codeGen *gen.CodeGenerator
}

// NewCustomerService creates a new instance of CustomerService
func NewCustomerService(s *server.Server) *CustomerService {
	return &CustomerService{
		db:      s.DB,
		logger:  s.Logger,
		codeGen: s.CodeGenerator,
	}
}

// QueryFilter defines the filter parameters for querying Customer
type CustomerQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s CustomerService) filterQuery(q *bun.SelectQuery, f *CustomerQueryFilter) *bun.SelectQuery {
	q = q.Where("cu.organization_id = ?", f.OrganizationID).
		Where("cu.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("cu.code = ? OR cu.name ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN cu.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("cu.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all Customer based on the provided filter
func (s CustomerService) GetAll(ctx context.Context, filter *CustomerQueryFilter) ([]*models.Customer, int, error) {
	var entities []*models.Customer

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Customer")
		return nil, 0, fmt.Errorf("failed to fetch Customer: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single Customer by ID
func (s CustomerService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.Customer, error) {
	entity := new(models.Customer)
	err := s.db.NewSelect().
		Model(entity).
		Where("cu.organization_id = ?", orgID).
		Where("cu.business_unit_id = ?", buID).
		Where("cu.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Customer")
		return nil, fmt.Errorf("failed to fetch Customer: %w", err)
	}

	return entity, nil
}

// Create creates a new Customer
func (s CustomerService) Create(ctx context.Context, entity *models.Customer) (*models.Customer, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Query the master key generation entity.
		mkg, mErr := models.QueryCustomerMasterKeyGenerationByOrgID(ctx, s.db, entity.OrganizationID)
		if mErr != nil {
			return mErr
		}

		return entity.InsertCustomer(ctx, tx, s.codeGen, mkg.Pattern)
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create Customer")
		return nil, err
	}

	return entity, nil
}

// UpdateOne updates an existing Customer
func (s CustomerService) UpdateOne(ctx context.Context, entity *models.Customer) (*models.Customer, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update Customer")
		return nil, fmt.Errorf("failed to update Customer: %w", err)
	}

	return entity, nil
}
