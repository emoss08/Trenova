package ratematrixservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.RateMatrixRepository
	DensityRepo  repositories.DensityScaleRepository
	Validator    *Validator
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.RateMatrixRepository
	densityRepo  repositories.DensityScaleRepository
	validator    *Validator
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.rate-matrix"),
		repo:         p.Repo,
		densityRepo:  p.DensityRepo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListRateMatrixRequest,
) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListRateMatrixConnectionRequest,
) (*pagination.CursorListResult[*ratematrix.RateMatrix], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req *repositories.GetRateMatrixByIDRequest,
) (*ratematrix.RateMatrix, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
	userID pulid.ID,
) (*ratematrix.RateMatrix, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userId", userID.String()),
	)

	entity.StampDimensions()

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create rate matrix", zap.Error(err))
		return nil, err
	}

	s.audit(log, created, nil, permission.OpCreate, userID, "Rate matrix created")

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
	userID pulid.ID,
) (*ratematrix.RateMatrix, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userId", userID.String()),
	)

	entity.StampDimensions()

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetRateMatrixByIDRequest{
		RateMatrixID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		IncludeDimensions: true,
	})
	if err != nil {
		log.Error("failed to load rate matrix for update", zap.Error(err))
		return nil, err
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update rate matrix", zap.Error(err))
		return nil, err
	}

	s.audit(log, updated, original, permission.OpUpdate, userID, "Rate matrix updated")

	return updated, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req *repositories.GetRateMatrixByIDRequest,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("rateMatrixId", req.RateMatrixID.String()),
	)

	existing, err := s.repo.GetByID(ctx, req)
	if err != nil {
		log.Error("failed to load rate matrix for delete", zap.Error(err))
		return err
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete rate matrix", zap.Error(err))
		return err
	}

	s.audit(log, existing, nil, permission.OpDelete, userID, "Rate matrix deleted")

	return nil
}

func (s *Service) ListCells(
	ctx context.Context,
	req *repositories.ListRateMatrixCellsRequest,
) (*pagination.ListResult[*ratematrix.RateMatrixCell], error) {
	return s.repo.ListCells(ctx, req)
}

// ReplaceCells swaps a matrix's entire grid for a new one.
//
// Replacement rather than merge is deliberate. A class tariff arrives as a
// whole sheet, and reconciling it row by row would leave behind whatever the
// new sheet dropped — cells that then keep pricing shipments at last year's
// rates with nothing on screen to suggest they exist.
func (s *Service) ReplaceCells(
	ctx context.Context,
	req *repositories.ReplaceRateMatrixCellsRequest,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "ReplaceCells"),
		zap.String("rateMatrixId", req.RateMatrixID.String()),
		zap.Int("cells", len(req.Cells)),
	)

	matrix, err := s.repo.GetByID(ctx, &repositories.GetRateMatrixByIDRequest{
		RateMatrixID:      req.RateMatrixID,
		TenantInfo:        req.TenantInfo,
		IncludeDimensions: true,
	})
	if err != nil {
		log.Error("failed to load rate matrix for cell replacement", zap.Error(err))
		return err
	}

	if multiErr := s.validator.ValidateCells(matrix, req); multiErr != nil {
		return multiErr
	}

	if err = s.repo.ReplaceCells(ctx, req); err != nil {
		log.Error("failed to replace rate matrix cells", zap.Error(err))
		return err
	}

	s.audit(log, matrix, nil, permission.OpUpdate, userID, "Rate matrix cells replaced")

	return nil
}

func (s *Service) ListDensityScales(
	ctx context.Context,
	req *repositories.ListRateMatrixRequest,
) (*pagination.ListResult[*ratematrix.DensityScale], error) {
	return s.densityRepo.List(ctx, req)
}

func (s *Service) GetDensityScale(
	ctx context.Context,
	req *repositories.GetDensityScaleRequest,
) (*ratematrix.DensityScale, error) {
	return s.densityRepo.GetByID(ctx, req)
}

func (s *Service) audit(
	log *zap.Logger,
	entity *ratematrix.RateMatrix,
	original *ratematrix.RateMatrix,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
) {
	params := &services.LogActionParams{
		Resource:       permission.ResourceRateMatrix,
		ResourceID:     entity.GetID().String(),
		Operation:      operation,
		UserID:         userID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}

	if operation == permission.OpDelete {
		params.PreviousState = jsonutils.MustToJSON(entity)
	} else {
		params.CurrentState = jsonutils.MustToJSON(entity)
	}

	opts := []services.LogOption{auditservice.WithComment(comment)}

	if original != nil {
		params.PreviousState = jsonutils.MustToJSON(original)
		opts = append(opts, auditservice.WithDiff(original, entity))
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}
