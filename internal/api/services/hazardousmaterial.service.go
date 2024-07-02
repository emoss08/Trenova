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

// HazardousMaterialService handles business logic for HazardousMaterial
type HazardousMaterialService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewHazardousMaterialService creates a new instance of HazardousMaterialService
func NewHazardousMaterialService(s *server.Server) *HazardousMaterialService {
	return &HazardousMaterialService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying HazardousMaterial
type HazardousMaterialQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s *HazardousMaterialService) filterQuery(q *bun.SelectQuery, f *HazardousMaterialQueryFilter) *bun.SelectQuery {
	q = q.Where("hm.organization_id = ?", f.OrganizationID).
		Where("hm.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("hm.name = ? OR hm.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN hm.name = ? THEN 0 ELSE 1 END", f.Query).
		Order("hm.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all HazardousMaterial based on the provided filter
func (s *HazardousMaterialService) GetAll(ctx context.Context, filter *HazardousMaterialQueryFilter) ([]*models.HazardousMaterial, int, error) {
	var entities []*models.HazardousMaterial

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch HazardousMaterial")
		return nil, 0, fmt.Errorf("failed to fetch HazardousMaterial: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single HazardousMaterial by ID
func (s *HazardousMaterialService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.HazardousMaterial, error) {
	entity := new(models.HazardousMaterial)
	err := s.db.NewSelect().
		Model(entity).
		Where("hm.organization_id = ?", orgID).
		Where("hm.business_unit_id = ?", buID).
		Where("hm.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch HazardousMaterial")
		return nil, fmt.Errorf("failed to fetch HazardousMaterial: %w", err)
	}

	return entity, nil
}

// Create creates a new HazardousMaterial
func (s *HazardousMaterialService) Create(ctx context.Context, entity *models.HazardousMaterial) (*models.HazardousMaterial, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create HazardousMaterial")
		return nil, fmt.Errorf("failed to create HazardousMaterial: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing HazardousMaterial
func (s *HazardousMaterialService) UpdateOne(ctx context.Context, entity *models.HazardousMaterial) (*models.HazardousMaterial, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update HazardousMaterial")
		return nil, fmt.Errorf("failed to update HazardousMaterial: %w", err)
	}

	return entity, nil
}
