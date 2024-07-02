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

// GeneralLedgerAccountService handles business logic for GeneralLedgerAccount
type GeneralLedgerAccountService struct {
	db     *bun.DB
	logger *zerolog.Logger
}

// NewGeneralLedgerAccountService creates a new instance of GeneralLedgerAccountService
func NewGeneralLedgerAccountService(s *server.Server) *GeneralLedgerAccountService {
	return &GeneralLedgerAccountService{
		db:     s.DB,
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying GeneralLedgerAccount
type GeneralLedgerAccountQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s *GeneralLedgerAccountService) filterQuery(q *bun.SelectQuery, f *GeneralLedgerAccountQueryFilter) *bun.SelectQuery {
	q = q.Where("gla.organization_id = ?", f.OrganizationID).
		Where("gla.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("gla.account_number = ? OR gla.account_number ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN gla.account_number = ? THEN 0 ELSE 1 END", f.Query).
		Order("gla.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all GeneralLedgerAccount based on the provided filter
func (s *GeneralLedgerAccountService) GetAll(ctx context.Context, filter *GeneralLedgerAccountQueryFilter) ([]*models.GeneralLedgerAccount, int, error) {
	var entities []*models.GeneralLedgerAccount

	q := s.db.NewSelect().
		Model(&entities)

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch GeneralLedgerAccount")
		return nil, 0, fmt.Errorf("failed to fetch GeneralLedgerAccount: %w", err)
	}

	return entities, count, nil
}

// Get retrieves a single GeneralLedgerAccount by ID
func (s *GeneralLedgerAccountService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.GeneralLedgerAccount, error) {
	entity := new(models.GeneralLedgerAccount)
	err := s.db.NewSelect().
		Model(entity).
		Where("gla.organization_id = ?", orgID).
		Where("gla.business_unit_id = ?", buID).
		Where("gla.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch GeneralLedgerAccount")
		return nil, fmt.Errorf("failed to fetch GeneralLedgerAccount: %w", err)
	}

	return entity, nil
}

// Create creates a new GeneralLedgerAccount
func (s *GeneralLedgerAccountService) Create(ctx context.Context, entity *models.GeneralLedgerAccount) (*models.GeneralLedgerAccount, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create GeneralLedgerAccount")
		return nil, fmt.Errorf("failed to create GeneralLedgerAccount: %w", err)
	}

	return entity, nil
}

// UpdateOne updates an existing GeneralLedgerAccount
func (s *GeneralLedgerAccountService) UpdateOne(ctx context.Context, entity *models.GeneralLedgerAccount) (*models.GeneralLedgerAccount, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().
			Model(entity).
			WherePK().
			Returning("*").
			Exec(ctx)
		return err
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update GeneralLedgerAccount")
		return nil, fmt.Errorf("failed to update GeneralLedgerAccount: %w", err)
	}

	return entity, nil
}
