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

// DivisionCodeService handles business logic for DivisionCode
type DivisionCodeService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewDivisionCodeService creates a new instance of DivisionCodeService
func NewDivisionCodeService(s *server.Server) *DivisionCodeService {
	return &DivisionCodeService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying DivisionCode
type DivisionCodeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s DivisionCodeService) filterQuery(q *bun.SelectQuery, f *DivisionCodeQueryFilter) *bun.SelectQuery {
	q = q.Where("dc.organization_id = ?", f.OrganizationID).
		Where("dc.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("dc.code = ? OR dc.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN dc.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("dc.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all DivisionCode based on the provided filter
func (s DivisionCodeService) GetAll(ctx context.Context, filter *DivisionCodeQueryFilter) ([]*models.DivisionCode, int, error) {
	var entities []*models.DivisionCode

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch DivisionCode")
		return nil, 0, fmt.Errorf("failed to fetch DivisionCode: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single DivisionCode by ID
func (s DivisionCodeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.DivisionCode, error) {
	entity := new(models.DivisionCode)
	err := s.db.NewSelect().
		Model(entity).
		Where("dc.organization_id = ?", orgID).
		Where("dc.business_unit_id = ?", buID).
		Where("dc.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch DivisionCode")
		return nil, fmt.Errorf("failed to fetch DivisionCode: %w", err)
	}

	return entity, nil
}

// Create creates a new DivisionCode
func (s DivisionCodeService) Create(ctx context.Context, entity *models.DivisionCode) (*models.DivisionCode, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create DivisionCode")
		return nil, fmt.Errorf("failed to create DivisionCode: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing DivisionCode
func (s DivisionCodeService) UpdateOne(ctx context.Context, entity *models.DivisionCode) (*models.DivisionCode, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update DivisionCode")
		return nil, fmt.Errorf("failed to update DivisionCode: %w", err)
	}

	return entity, nil
}
