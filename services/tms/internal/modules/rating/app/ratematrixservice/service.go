package ratematrixservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixcell"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type usageLister interface {
	ListMatrixUsages(
		ctx context.Context,
		req *formulatemplateports.ListMatrixUsagesRequest,
	) ([]*formulatemplateports.MatrixUsage, error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	DB            kernelports.Transactor
	Repo          ratematrixports.RateMatrixRepository
	VersionRepo   ratematrixports.RateMatrixVersionRepository
	ReviewRepo    ratematrixports.RateMatrixReviewRepository
	Usages        formulatemplateports.FormulaTemplateVersionRepository
	Validator     *Validator
	AuditService  auditports.AuditService
	Notifications *notificationservice.Service   `optional:"true"`
	Roles         permissionports.RoleRepository `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	db            kernelports.Transactor
	repo          ratematrixports.RateMatrixRepository
	versionRepo   ratematrixports.RateMatrixVersionRepository
	reviewRepo    ratematrixports.RateMatrixReviewRepository
	usages        usageLister
	validator     *Validator
	auditService  auditports.AuditService
	notifications *notificationservice.Service
	roles         permissionports.RoleRepository
}

//nolint:gocritic // fx param structs are passed by value
func New(p Params) *Service {
	return &Service{
		l:             p.Logger.Named("service.rate-matrix"),
		db:            p.DB,
		repo:          p.Repo,
		versionRepo:   p.VersionRepo,
		reviewRepo:    p.ReviewRepo,
		usages:        p.Usages,
		validator:     p.Validator,
		auditService:  p.AuditService,
		notifications: p.Notifications,
		roles:         p.Roles,
	}
}

func matrixTenant(matrix *ratematrix.RateMatrix) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: matrix.OrganizationID, BuID: matrix.BusinessUnitID}
}

func (s *Service) List(
	ctx context.Context,
	req *ratematrixports.ListRateMatrixRequest,
) (*pagination.ListResult[*ratematrix.RateMatrix], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *ratematrixports.ListRateMatrixConnectionRequest,
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
	req *ratematrixports.GetRateMatrixByIDRequest,
) (*ratematrix.RateMatrix, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetByIDs(
	ctx context.Context,
	req *ratematrixports.GetRateMatricesByIDsRequest,
) ([]*ratematrix.RateMatrix, error) {
	return s.repo.GetByIDs(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *ratematrix.RateMatrix,
	userID pulid.ID,
) (*ratematrix.RateMatrix, error) {
	entity.ID = pulid.Nil
	entity.Status = formulatemplate.StatusDraft
	entity.PublishedVersionNumber = 0
	entity.PublishedAt = nil
	entity.HasDraftChanges = true
	entity.DraftCellCount = 0
	entity.DraftEditorIDs = []pulid.ID{}
	entity.ClearApproval()
	entity.RecordEditor(userID)
	entity.StampDimensions()

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.audit(ctx, &auditRecord{
		entity:    created,
		operation: permission.OpCreate,
		userID:    userID,
		comment:   "Rate matrix draft created",
	})
	return created, nil
}

func withDraft(original, payload *ratematrix.RateMatrix) *ratematrix.RateMatrix {
	next := *original
	next.Code = payload.Code
	next.Name = payload.Name
	next.Description = payload.Description
	next.ValueType = payload.ValueType
	next.Version = payload.Version
	next.Dimensions = payload.Dimensions
	return &next
}

func (s *Service) Update(
	ctx context.Context,
	payload *ratematrix.RateMatrix,
	userID pulid.ID,
) (*ratematrix.RateMatrix, error) {
	var original, updated *ratematrix.RateMatrix
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		loaded, err := s.repo.GetByID(txCtx, &ratematrixports.GetRateMatrixByIDRequest{
			RateMatrixID:      payload.ID,
			TenantInfo:        matrixTenant(payload),
			IncludeDimensions: true,
			ForUpdate:         true,
		})
		if err != nil {
			return err
		}
		original = loaded

		entity := withDraft(original, payload)
		if err = refuseRecode(entity, original); err != nil {
			return err
		}
		entity.StampDimensions()
		if multiErr := s.validator.ValidateUpdate(txCtx, entity); multiErr != nil {
			return multiErr
		}

		if entity.HasMaterialChange(original) {
			markDraftChanged(entity, userID)
		}

		saved, err := s.repo.Update(txCtx, entity)
		if err != nil {
			return err
		}
		updated = saved
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.audit(ctx, &auditRecord{
		entity:    updated,
		original:  original,
		operation: permission.OpUpdate,
		userID:    userID,
		comment:   updateComment(original, updated),
	})
	return updated, nil
}

func updateComment(original, updated *ratematrix.RateMatrix) string {
	if original.Status != updated.Status && updated.Status == formulatemplate.StatusDraft {
		return "Rate matrix draft changed; the change reverted its approval"
	}
	return "Rate matrix draft updated"
}

func markDraftChanged(entity *ratematrix.RateMatrix, userID pulid.ID) {
	entity.HasDraftChanges = true
	entity.RecordEditor(userID)
	if entity.Status == formulatemplate.StatusActive ||
		entity.Status == formulatemplate.StatusInReview {
		entity.Status = formulatemplate.StatusDraft
		entity.ClearApproval()
	}
}

func refuseRecode(entity, original *ratematrix.RateMatrix) error {
	if entity.Code == original.Code || !original.IsPublished() {
		return nil
	}
	return errortypes.NewValidationError(
		"code",
		errortypes.ErrInvalid,
		"{0} has a published version, and tariffs read it as tables.{0}; a matrix cannot change its code once it has been published",
		original.Code,
	)
}

func (s *Service) Delete(
	ctx context.Context,
	req *ratematrixports.GetRateMatrixByIDRequest,
	userID pulid.ID,
) error {
	existing, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return err
	}
	if existing.IsPublished() {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"{0} has published versions that tariffs may pin; archive it instead of deleting it",
			existing.Code,
		)
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		return err
	}

	s.audit(ctx, &auditRecord{
		entity:    existing,
		operation: permission.OpDelete,
		userID:    userID,
		comment:   "Rate matrix draft deleted",
	})
	return nil
}

func (s *Service) ListCells(
	ctx context.Context,
	req *ratematrixports.ListRateMatrixCellsRequest,
) (*pagination.ListResult[*ratematrixcell.RateMatrixCell], error) {
	return s.repo.ListCells(ctx, req)
}

func (s *Service) ReplaceCells(
	ctx context.Context,
	req *ratematrixports.ReplaceRateMatrixCellsRequest,
	userID pulid.ID,
) error {
	var original, updated *ratematrix.RateMatrix
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		matrix, err := s.repo.GetByID(txCtx, &ratematrixports.GetRateMatrixByIDRequest{
			RateMatrixID:      req.RateMatrixID,
			TenantInfo:        req.TenantInfo,
			IncludeDimensions: true,
			ForUpdate:         true,
		})
		if err != nil {
			return err
		}
		snapshot := *matrix
		original = &snapshot

		if multiErr := s.validator.ValidateCells(matrix, req); multiErr != nil {
			return multiErr
		}
		if err = s.repo.ReplaceCells(txCtx, req); err != nil {
			return err
		}

		matrix.DraftCellCount = int64(countCells(req.Cells))
		markDraftChanged(matrix, userID)
		saved, err := s.repo.UpdateHeader(txCtx, matrix)
		if err != nil {
			return err
		}
		updated = saved
		return nil
	})
	if err != nil {
		return err
	}

	s.audit(ctx, &auditRecord{
		entity:    updated,
		original:  original,
		operation: permission.OpUpdate,
		userID:    userID,
		comment:   "Rate matrix draft cells replaced",
	})
	return nil
}

func countCells(cells []*ratematrixcell.RateMatrixCell) int {
	count := 0
	for _, cell := range cells {
		if cell != nil {
			count++
		}
	}
	return count
}

type auditRecord struct {
	entity    *ratematrix.RateMatrix
	original  *ratematrix.RateMatrix
	operation permission.Operation
	userID    pulid.ID
	comment   string
	critical  bool
}

func (s *Service) audit(ctx context.Context, record *auditRecord) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceRateMatrix,
		ResourceID:     record.entity.GetID().String(),
		Operation:      record.operation,
		UserID:         record.userID,
		OrganizationID: record.entity.OrganizationID,
		BusinessUnitID: record.entity.BusinessUnitID,
	}
	if record.operation == permission.OpDelete {
		params.PreviousState = jsonutils.MustToJSON(record.entity)
	} else {
		params.CurrentState = jsonutils.MustToJSON(record.entity)
	}

	opts := []auditports.LogOption{
		auditservice.WithComment(record.comment),
		auditservice.WithRequest(ctx),
	}
	if record.original != nil {
		params.PreviousState = jsonutils.MustToJSON(record.original)
		opts = append(opts, auditservice.WithDiff(record.original, record.entity))
	}
	if record.critical {
		opts = append(opts, auditservice.WithCritical())
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log audit action",
			zap.String("rateMatrixID", record.entity.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) invalidateServed(ctx context.Context, tenant pagination.TenantInfo) {
	kernelports.AfterCommit(ctx, func(context.Context) {
		ratetablecache.Invalidate(tenant.OrgID, tenant.BuID)
	})
}
