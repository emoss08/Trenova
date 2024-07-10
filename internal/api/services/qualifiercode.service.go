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

// QualifierCodeService handles business logic for QualifierCode
type QualifierCodeService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewQualifierCodeService creates a new instance of QualifierCodeService
func NewQualifierCodeService(s *server.Server) *QualifierCodeService {
	return &QualifierCodeService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying QualifierCode
type QualifierCodeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s QualifierCodeService) filterQuery(q *bun.SelectQuery, f *QualifierCodeQueryFilter) *bun.SelectQuery {
	q = q.Where("qc.organization_id = ?", f.OrganizationID).
		Where("qc.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("qc.code = ? OR qc.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN qc.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("qc.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all QualifierCode based on the provided filter
func (s QualifierCodeService) GetAll(ctx context.Context, filter *QualifierCodeQueryFilter) ([]*models.QualifierCode, int, error) {
	var entities []*models.QualifierCode

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch QualifierCode")
		return nil, 0, fmt.Errorf("failed to fetch QualifierCode: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single QualifierCode by ID
func (s QualifierCodeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.QualifierCode, error) {
	entity := new(models.QualifierCode)
	err := s.db.NewSelect().
		Model(entity).
		Where("qc.organization_id = ?", orgID).
		Where("qc.business_unit_id = ?", buID).
		Where("qc.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch QualifierCode")
		return nil, fmt.Errorf("failed to fetch QualifierCode: %w", err)
	}

	return entity, nil
}

// Create creates a new QualifierCode
func (s QualifierCodeService) Create(ctx context.Context, entity *models.QualifierCode) (*models.QualifierCode, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create QualifierCode")
		return nil, fmt.Errorf("failed to create QualifierCode: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing QualifierCode
func (s QualifierCodeService) UpdateOne(ctx context.Context, entity *models.QualifierCode) (*models.QualifierCode, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update QualifierCode")
		return nil, fmt.Errorf("failed to update QualifierCode: %w", err)
	}

	return entity, nil
}
