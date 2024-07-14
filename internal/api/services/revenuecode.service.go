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

// RevenueCodeService handles business logic for RevenueCode
type RevenueCodeService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewRevenueCodeService creates a new instance of RevenueCodeService
func NewRevenueCodeService(s *server.Server) *RevenueCodeService {
	return &RevenueCodeService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying RevenueCode
type RevenueCodeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s RevenueCodeService) filterQuery(q *bun.SelectQuery, f *RevenueCodeQueryFilter) *bun.SelectQuery {
	q = q.Where("rc.organization_id = ?", f.OrganizationID).
		Where("rc.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("rc.code = ? OR rc.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN rc.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("rc.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all RevenueCode based on the provided filter
func (s RevenueCodeService) GetAll(ctx context.Context, filter *RevenueCodeQueryFilter) ([]*models.RevenueCode, int, error) {
	var entities []*models.RevenueCode

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch RevenueCode")
		return nil, 0, fmt.Errorf("failed to fetch RevenueCode: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single RevenueCode by ID
func (s RevenueCodeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.RevenueCode, error) {
	entity := new(models.RevenueCode)
	err := s.db.NewSelect().
		Model(entity).
		Where("rc.organization_id = ?", orgID).
		Where("rc.business_unit_id = ?", buID).
		Where("rc.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch RevenueCode")
		return nil, fmt.Errorf("failed to fetch RevenueCode: %w", err)
	}

	return entity, nil
}

// Create creates a new RevenueCode
func (s RevenueCodeService) Create(ctx context.Context, entity *models.RevenueCode) (*models.RevenueCode, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create RevenueCode")
		return nil, fmt.Errorf("failed to create RevenueCode: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing RevenueCode
func (s RevenueCodeService) UpdateOne(ctx context.Context, entity *models.RevenueCode) (*models.RevenueCode, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update RevenueCode")
		return nil, fmt.Errorf("failed to update RevenueCode: %w", err)
	}

	return entity, nil
}
