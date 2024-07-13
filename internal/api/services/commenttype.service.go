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

// CommentTypeService handles business logic for CommentType
type CommentTypeService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewCommentTypeService creates a new instance of CommentTypeService
func NewCommentTypeService(s *server.Server) *CommentTypeService {
	return &CommentTypeService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying CommentType
type CommentTypeQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s CommentTypeService) filterQuery(q *bun.SelectQuery, f *CommentTypeQueryFilter) *bun.SelectQuery {
	q = q.Where("ct.organization_id = ?", f.OrganizationID).
		Where("ct.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("ct.name = ? OR ct.description ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN ct.name = ? THEN 0 ELSE 1 END", f.Query).
		Order("ct.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all CommentType based on the provided filter
func (s CommentTypeService) GetAll(ctx context.Context, filter *CommentTypeQueryFilter) ([]*models.CommentType, int, error) {
	var entities []*models.CommentType

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch CommentType")
		return nil, 0, fmt.Errorf("failed to fetch CommentType: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single CommentType by ID
func (s CommentTypeService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.CommentType, error) {
	entity := new(models.CommentType)
	err := s.db.NewSelect().
		Model(entity).
		Where("ct.organization_id = ?", orgID).
		Where("ct.business_unit_id = ?", buID).
		Where("ct.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch CommentType")
		return nil, fmt.Errorf("failed to fetch CommentType: %w", err)
	}

	return entity, nil
}

// Create creates a new CommentType
func (s CommentTypeService) Create(ctx context.Context, entity *models.CommentType) (*models.CommentType, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create CommentType")
		return nil, fmt.Errorf("failed to create CommentType: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing CommentType
func (s CommentTypeService) UpdateOne(ctx context.Context, entity *models.CommentType) (*models.CommentType, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := entity.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update CommentType")
		return nil, fmt.Errorf("failed to update CommentType: %w", err)
	}

	return entity, nil
}
