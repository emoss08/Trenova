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

// ServiceTypeService handles business logic for ServiceType
type ServiceTypeService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewServiceTypeService creates a new instance of ServiceTypeService
func NewServiceTypeService(s *server.Server) *ServiceTypeService {
	return &ServiceTypeService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying ServiceType
type ServiceTypeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s *ServiceTypeService) filterQuery(q *bun.SelectQuery, f *ServiceTypeQueryFilter) *bun.SelectQuery {
	q = q.Where("st.organization_id = ?", f.OrganizationID).
		Where("st.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("st.code = ? OR st.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN st.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("st.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all ServiceType based on the provided filter
func (s *ServiceTypeService) GetAll(ctx context.Context, filter *ServiceTypeQueryFilter) ([]*models.ServiceType, int, error) {
	var entities []*models.ServiceType

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch ServiceType")
		return nil, 0, fmt.Errorf("failed to fetch ServiceType: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single ServiceType by ID
func (s *ServiceTypeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.ServiceType, error) {
	entity := new(models.ServiceType)
	err := s.db.NewSelect().
		Model(entity).
		Where("st.organization_id = ?", orgID).
		Where("st.business_unit_id = ?", buID).
		Where("st.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch ServiceType")
		return nil, fmt.Errorf("failed to fetch ServiceType: %w", err)
	}

	return entity, nil
}

// Create creates a new ServiceType
func (s *ServiceTypeService) Create(ctx context.Context, entity *models.ServiceType) (*models.ServiceType, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create ServiceType")
		return nil, fmt.Errorf("failed to create ServiceType: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing ServiceType
func (s *ServiceTypeService) UpdateOne(ctx context.Context, entity *models.ServiceType) (*models.ServiceType, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update ServiceType")
		return nil, fmt.Errorf("failed to update ServiceType: %w", err)
	}

	return entity, nil
}
