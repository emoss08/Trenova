package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

// CustomerService handles business logic for Customer
type CustomerService struct {
	db      *bun.DB
	logger  *zerolog.Logger
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
func (s *CustomerService) filterQuery(q *bun.SelectQuery, f *CustomerQueryFilter) *bun.SelectQuery {
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
func (s *CustomerService) GetAll(ctx context.Context, filter *CustomerQueryFilter) ([]*models.Customer, int, error) {
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
func (s *CustomerService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.Customer, error) {
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
func (s *CustomerService) Create(ctx context.Context, entity *models.Customer) (*models.Customer, error) {
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
func (s *CustomerService) UpdateOne(ctx context.Context, entity *models.Customer) (*models.Customer, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update Customer")
		return nil, fmt.Errorf("failed to update Customer: %w", err)
	}

	return entity, nil
}
