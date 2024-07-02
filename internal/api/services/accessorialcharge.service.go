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

// AccessorialChargeService handles business logic for AccessorialCharge
type AccessorialChargeService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewAccessorialChargeService creates a new instance of AccessorialChargeService
func NewAccessorialChargeService(s *server.Server) *AccessorialChargeService {
	return &AccessorialChargeService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying AccessorialCharge
type AccessorialChargeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s *AccessorialChargeService) filterQuery(q *bun.SelectQuery, f *AccessorialChargeQueryFilter) *bun.SelectQuery {
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
func (s *AccessorialChargeService) GetAll(ctx context.Context, filter *AccessorialChargeQueryFilter) ([]*models.AccessorialCharge, int, error) {
	var entities []*models.AccessorialCharge

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch AccessorialCharge")
		return nil, 0, fmt.Errorf("failed to fetch AccessorialCharge: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single AccessorialCharge by ID
func (s *AccessorialChargeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.AccessorialCharge, error) {
	entity := new(models.AccessorialCharge)
	err := s.db.NewSelect().
		Model(entity).
		Where("ac.organization_id = ?", orgID).
		Where("ac.business_unit_id = ?", buID).
		Where("ac.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch AccessorialCharge")
		return nil, fmt.Errorf("failed to fetch AccessorialCharge: %w", err)
	}

	return entity, nil
}

// Create creates a new AccessorialCharge
func (s *AccessorialChargeService) Create(ctx context.Context, entity *models.AccessorialCharge) (*models.AccessorialCharge, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create AccessorialCharge")
		return nil, fmt.Errorf("failed to create AccessorialCharge: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing AccessorialCharge
func (s *AccessorialChargeService) UpdateOne(ctx context.Context, entity *models.AccessorialCharge) (*models.AccessorialCharge, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update AccessorialCharge")
		return nil, fmt.Errorf("failed to update AccessorialCharge: %w", err)
	}

	return entity, nil
}
