package glaccountservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	DB           *postgres.Connection
	Repo         repositories.GLAccountRepository
	Validator    *Validator
	AuditService services.AuditService
	Transformer  services.DataTransformer
}

type Service struct {
	l            *zap.Logger
	db           *postgres.Connection
	repo         repositories.GLAccountRepository
	validator    *Validator
	auditService services.AuditService
	transformer  services.DataTransformer
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.glaccount"),
		db:           p.DB,
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		transformer:  p.Transformer,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListGLAccountsRequest,
) (*pagination.ListResult[*glaccount.GLAccount], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetGLAccountByIDRequest,
) (*glaccount.GLAccount, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.GLAccountSelectOptionsRequest,
) (*pagination.ListResult[*glaccount.GLAccount], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateGLAccountStatusRequest,
) ([]*glaccount.GLAccount, error) {
	log := s.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	originalEntities, err := s.repo.GetByIDs(ctx, repositories.GetGLAccountsByIDsRequest{
		TenantInfo:   req.TenantInfo,
		GLAccountIDs: req.GLAccountIDs,
	})
	if err != nil {
		log.Error("failed to get original gl accounts", zap.Error(err))
		return nil, err
	}

	entities, err := s.repo.BulkUpdateStatus(ctx, req)
	if err != nil {
		log.Error("failed to bulk update gl account status", zap.Error(err))
		return nil, err
	}

	entries := auditservice.BuildBulkLogEntries(
		&auditservice.BulkLogEntriesParams[*glaccount.GLAccount]{
			Resource:  permission.ResourceGeneralLedgerAccount,
			Operation: permission.OpUpdate,
			UserID:    req.TenantInfo.UserID,
			Updated:   entities,
			Originals: originalEntities,
		},
		auditservice.WithComment("GL account status updated"),
	)

	if err = s.auditService.LogActions(entries); err != nil {
		log.Error("failed to log audit actions", zap.Error(err))
	}

	return entities, nil
}

func (s *Service) Create(
	ctx context.Context,
	entity *glaccount.GLAccount,
	userID pulid.ID,
) (*glaccount.GLAccount, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
	)

	if err := s.transformer.TransformGLAccount(ctx, entity); err != nil {
		log.Error("failed to transform gl account", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create gl account", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceGeneralLedgerAccount,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("GL account created"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *glaccount.GLAccount,
	userID pulid.ID,
) (*glaccount.GLAccount, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
	)

	if err := s.transformer.TransformGLAccount(ctx, entity); err != nil {
		log.Error("failed to transform gl account", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetGLAccountByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original gl account", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update gl account", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceGeneralLedgerAccount,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("GL account updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.DeleteGLAccountRequest,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("id", req.ID.String()),
		zap.String("userID", userID.String()),
	)

	existing, err := s.repo.GetByID(ctx, repositories.GetGLAccountByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		log.Error("failed to get gl account for delete", zap.Error(err))
		return err
	}

	if multiErr := s.validateDelete(ctx, existing); multiErr != nil {
		return multiErr
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		log.Error("failed to delete gl account", zap.Error(err))
		return err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceGeneralLedgerAccount,
		ResourceID:     existing.GetID().String(),
		Operation:      permission.OpDelete,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(existing),
		OrganizationID: existing.OrganizationID,
		BusinessUnitID: existing.BusinessUnitID,
	},
		auditservice.WithComment("GL account deleted"),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return nil
}

func (s *Service) validateDelete(
	ctx context.Context,
	entity *glaccount.GLAccount,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if entity.IsSystem {
		multiErr.Add("isSystem", errortypes.ErrInvalid, "System accounts cannot be deleted")
	}

	if entity.CurrentBalance != 0 {
		multiErr.Add(
			"currentBalance",
			errortypes.ErrInvalid,
			"Cannot delete account with non-zero balance",
		)
	}

	childCount, err := s.db.DB().NewSelect().
		TableExpr("gl_accounts").
		Where("parent_id = ?", entity.ID).
		Where("organization_id = ?", entity.OrganizationID).
		Where("business_unit_id = ?", entity.BusinessUnitID).
		Where("status = ?", domaintypes.StatusActive).
		Count(ctx)
	if err != nil {
		return nil
	}

	if childCount > 0 {
		multiErr.Add("children", errortypes.ErrInvalid, "Cannot delete account with child accounts")
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
