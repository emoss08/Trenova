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
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type WorkerService struct {
	db      *bun.DB
	logger  *config.ServerLogger
	codeGen *gen.CodeGenerator
}

func NewWorkerService(s *server.Server) *WorkerService {
	return &WorkerService{
		db:      s.DB,
		logger:  s.Logger,
		codeGen: s.CodeGenerator,
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

	q := s.db.NewSelect().
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
	err := s.db.NewSelect().
		Model(entity).
		Where("wk.organization_id = ?", orgID).
		Where("wk.business_unit_id = ?", buID).
		Where("wk.id = ?", id).
		Scan(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get worker")
		return nil, err
	}

	return entity, nil
}

func (s WorkerService) Create(ctx context.Context, entity *models.Worker) (*models.Worker, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		mkg, mErr := models.QueryWorkerMasterKeyGenerationByOrgID(ctx, s.db, entity.OrganizationID)
		if mErr != nil {
			s.logger.Error().Err(mErr).Msg("failed to get worker master key generation")
			return mErr
		}

		return entity.InsertWorker(ctx, tx, s.codeGen, mkg.Pattern)
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create worker")
		return nil, err
	}

	return entity, nil
}

func (s WorkerService) UpdateOne(ctx context.Context, entity *models.Worker) (*models.Worker, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return entity.UpdateWorker(ctx, tx)
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to update worker")
		return nil, err
	}

	return entity, nil
}
