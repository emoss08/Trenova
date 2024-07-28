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

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/api/common"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type WorkerService struct {
	common.AuditableService
	logger *config.ServerLogger
}

func NewWorkerService(s *server.Server) *WorkerService {
	return &WorkerService{
		AuditableService: common.AuditableService{
			DB:            s.DB,
			AuditService:  s.AuditService,
			CodeGenerator: s.CodeGenerator,
		},
		logger: s.Logger,
	}
}

// QueryFilter defines the filter parameters for querying Worker
type WorkerQueryFilter struct {
	Query          string
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	Limit          int
	Offset         int
}

func (s WorkerService) filterQuery(q *bun.SelectQuery, f *WorkerQueryFilter) *bun.SelectQuery {
	q = q.Where("wk.organization_id = ?", f.OrganizationID).
		Where("wk.business_unit_id = ?", f.BusinessUnitID)

	if f.Query != "" {
		q = q.Where("wk.code = ? OR wk.code ILIKE ?", f.Query, "%"+strings.ToLower(f.Query)+"%")
	}

	q = q.OrderExpr("CASE WHEN wk.code = ? THEN 0 ELSE 1 END", f.Query).
		Order("wk.created_at DESC")

	return q.Limit(f.Limit).Offset(f.Offset)
}

func (s WorkerService) GetAll(ctx context.Context, filter *WorkerQueryFilter) ([]*models.Worker, int, error) {
	var entities []*models.Worker

	q := s.DB.NewSelect().
		Model(&entities).
		Relation("WorkerProfile")

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get workers")
		return nil, 0, err
	}

	return entities, count, nil
}

func (s WorkerService) Get(ctx context.Context, id uuid.UUID, orgID, buID uuid.UUID) (*models.Worker, error) {
	entity := new(models.Worker)
	err := s.GetByID(ctx, id, orgID, buID, entity)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get worker")
		return nil, err
	}

	return entity, nil
}

func (s WorkerService) Create(ctx context.Context, entity *models.Worker, userID uuid.UUID) (*models.Worker, error) {
	mkg, err := models.QueryWorkerMasterKeyGenerationByOrgID(ctx, s.DB, entity.OrganizationID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get worker master key generation")
		return nil, err
	}
	err = s.CreateWithAuditAndCodeGen(ctx, entity, userID, mkg.Pattern)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create worker")
		return nil, err
	}

	return entity, nil
}

func (s WorkerService) UpdateOne(ctx context.Context, entity *models.Worker, userID uuid.UUID) (*models.Worker, error) {
	err := s.UpdateWithAudit(ctx, entity, userID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to update worker")
		return nil, err
	}

	return entity, nil
}
