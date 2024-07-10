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

// TractorService handles business logic for Tractor
type TractorService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewTractorService creates a new instance of TractorService
func NewTractorService(s *server.Server) *TractorService {
	return &TractorService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying Tractor
type TractorQueryFilter struct {
	Query               string
	OrganizationID      uuid.UUID
	BusinessUnitID      uuid.UUID
	FleetCodeID         uuid.UUID
	Status              string
	ExpandWorkerDetails bool
	ExpandEquipDetails  bool
	Limit               int
	Offset              int
}

// filterQuery applies filters to the query
func (s *TractorService) filterQuery(q *bun.SelectQuery, f *TractorQueryFilter) *bun.SelectQuery {
	q = q.Where("tr.organization_id = ?", f.OrganizationID).
		Where("tr.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("tr.code = ? OR tr.code ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	if f.ExpandWorkerDetails {
		q = q.Relation("PrimaryWorker").
			Relation("PrimaryWorker.WorkerProfile").
			Relation("SecondaryWorker").
			Relation("SecondaryWorker.WorkerProfile")
	}

	if f.ExpandEquipDetails {
		q = q.Relation("EquipmentType").
			Relation("EquipmentManufacturer")
	}

	if f.Status != "" {
		q = q.Where("tr.status = ?", f.Status)
	}

	if f.FleetCodeID != uuid.Nil {
		q = q.Where("tr.fleet_code_id = ?", f.FleetCodeID)
	}

	q = q.OrderExpr("CASE WHEN tr.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("tr.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all Tractor based on the provided filter
func (s *TractorService) GetAll(ctx context.Context, filter *TractorQueryFilter) ([]*models.Tractor, int, error) {
	var entities []*models.Tractor

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Tractor")
		return nil, 0, fmt.Errorf("failed to fetch Tractor: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single Tractor by ID
func (s *TractorService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.Tractor, error) {
	entity := new(models.Tractor)
	err := s.db.NewSelect().
		Model(entity).
		Where("tr.organization_id = ?", orgID).
		Where("tr.business_unit_id = ?", buID).
		Where("tr.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Tractor")
		return nil, fmt.Errorf("failed to fetch Tractor: %w", err)
	}

	return entity, nil
}

// Create creates a new Tractor
func (s *TractorService) Create(ctx context.Context, entity *models.Tractor) (*models.Tractor, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create Tractor")
		return nil, fmt.Errorf("failed to create Tractor: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing Tractor
func (s *TractorService) UpdateOne(ctx context.Context, entity *models.Tractor) (*models.Tractor, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update Tractor")
		return nil, fmt.Errorf("failed to update Tractor: %w", err)
	}

	return entity, nil
}
