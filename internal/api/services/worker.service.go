package services

import (
	"context"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/gen"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type WorkerService struct {
	db      *bun.DB
	logger  *zerolog.Logger
	codeGen *gen.CodeGenerator
}

func NewWorkerService(s *server.Server) *WorkerService {
	return &WorkerService{
		db:      s.DB,
		logger:  s.Logger,
		codeGen: s.CodeGenerator,
	}
}

func (s *WorkerService) GetAll(ctx context.Context, limit, offset int, query string, orgID, buID uuid.UUID) ([]*models.Worker, int, error) {
	var entities []*models.Worker
	count, err := s.db.NewSelect().
		Model(&entities).
		Relation("WorkerProfile").
		Where("wk.organization_id = ?", orgID).
		Where("wk.business_unit_id = ?", buID).
		Where("wk.code ILIKE ?", "%"+query+"%").
		Order("wk.created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, count, nil
}

func (s *WorkerService) Get(ctx context.Context, id uuid.UUID, orgID, buID uuid.UUID) (*models.Worker, error) {
	entity := new(models.Worker)
	err := s.db.NewSelect().
		Model(entity).
		Where("wk.organization_id = ?", orgID).
		Where("wk.business_unit_id = ?", buID).
		Where("wk.id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *WorkerService) Create(ctx context.Context, entity *models.Worker) (*models.Worker, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Query the master key generation entity.
		mkg, mErr := models.QueryWorkerMasterKeyGenerationByOrgID(ctx, s.db, entity.OrganizationID)
		if mErr != nil {
			return mErr
		}

		return entity.InsertWorker(ctx, tx, s.codeGen, mkg.Pattern)
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func (s *WorkerService) UpdateOne(ctx context.Context, entity *models.Worker) (*models.Worker, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return entity.UpdateWorker(ctx, tx)
	})
	if err != nil {
		return nil, err
	}

	return entity, nil
}
