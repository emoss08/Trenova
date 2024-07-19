// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package services

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

// LocationService handles business logic for Location
type LocationService struct {
	db      *bun.DB
	logger  *zerolog.Logger
	codeGen *gen.CodeGenerator
}

// NewLocationService creates a new instance of LocationService
func NewLocationService(s *server.Server) *LocationService {
	return &LocationService{
		db:      s.DB,
		logger:  s.Logger,
		codeGen: s.CodeGenerator,
	}
}

// QueryFilter defines the filter parameters for querying Location
type LocationQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

// filterQuery applies filters to the query
func (s LocationService) filterQuery(q *bun.SelectQuery, f *LocationQueryFilter) *bun.SelectQuery {
	q = q.Where("lc.organization_id = ?", f.OrganizationID).
		Where("lc.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("lc.code = ? OR lc.name ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN lc.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("lc.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

// GetAll retrieves all Location based on the provided filter
func (s LocationService) GetAll(ctx context.Context, filter *LocationQueryFilter) ([]*models.Location, int, error) {
	var entities []*models.Location

	q := s.db.NewSelect().
		Model(&entities).
		Relation("LocationCategory").
		Relation("Comments").
		Relation("Contacts").
		Relation("State")

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Location")
		return nil, 0, err
	}

	return entities, count, nil
}

// Get retrieves a single Location by ID
func (s LocationService) Get(ctx context.Context, id, orgID, buID uuid.UUID) (*models.Location, error) {
	entity := new(models.Location)
	err := s.db.NewSelect().
		Model(entity).
		Where("lc.organization_id = ?", orgID).
		Where("lc.business_unit_id = ?", buID).
		Where("lc.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to fetch Location")
		return nil, err
	}

	return entity, nil
}

// Create creates a new Location
func (s LocationService) Create(ctx context.Context, entity *models.Location) (*models.Location, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Query the master key generation entity.
		mkg, mErr := models.QueryLocationMasterKeyGenerationByOrgID(ctx, s.db, entity.OrganizationID)
		if mErr != nil {
			return mErr
		}

		return entity.InsertLocation(ctx, tx, s.codeGen, mkg.Pattern)
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create Location")
		return nil, err
	}

	return entity, nil
}

// UpdateOne updates an existing Location
func (s LocationService) UpdateOne(ctx context.Context, entity *models.Location) (*models.Location, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return entity.UpdateLocation(ctx, tx)
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to update Location")
		return nil, err
	}

	return entity, nil
}
