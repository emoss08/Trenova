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

// ChargeTypeService handles business logic for ChargeType
type ChargeTypeService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewChargeTypeService creates a new instance of ChargeTypeService
func NewChargeTypeService(s *server.Server) *ChargeTypeService {
	return &ChargeTypeService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying ChargeType
type ChargeTypeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s *ChargeTypeService) filterQuery(q *bun.SelectQuery, f *ChargeTypeQueryFilter) *bun.SelectQuery {
	q = q.Where("ct.organization_id = ?", f.OrganizationID).
		Where("ct.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("ct.name = ? OR ct.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN ct.name = ? THEN 0 ELSE 1 END", f.Query).
		Order("ct.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all ChargeType based on the provided filter
func (s *ChargeTypeService) GetAll(ctx context.Context, filter *ChargeTypeQueryFilter) ([]*models.ChargeType, int, error) {
	var entities []*models.ChargeType

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch ChargeType")
		return nil, 0, fmt.Errorf("failed to fetch ChargeType: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single ChargeType by ID
func (s *ChargeTypeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.ChargeType, error) {
	entity := new(models.ChargeType)
	err := s.db.NewSelect().
		Model(entity).
		Where("ct.organization_id = ?", orgID).
		Where("ct.business_unit_id = ?", buID).
		Where("ct.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch ChargeType")
		return nil, fmt.Errorf("failed to fetch ChargeType: %w", err)
	}

	return entity, nil
}

// Create creates a new ChargeType
func (s *ChargeTypeService) Create(ctx context.Context, entity *models.ChargeType) (*models.ChargeType, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create ChargeType")
		return nil, fmt.Errorf("failed to create ChargeType: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing ChargeType
func (s *ChargeTypeService) UpdateOne(ctx context.Context, entity *models.ChargeType) (*models.ChargeType, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update ChargeType")
		return nil, fmt.Errorf("failed to update ChargeType: %w", err)
	}

	return entity, nil
}
