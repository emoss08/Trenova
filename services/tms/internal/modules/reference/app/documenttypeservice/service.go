package documenttypeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         documenttypeports.DocumentTypeRepository
	Validator    *Validator
	AuditService auditports.AuditService
	Transformer  accessorialchargeports.DataTransformer
}

type Service struct {
	l            *zap.Logger
	repo         documenttypeports.DocumentTypeRepository
	validator    *Validator
	auditService auditports.AuditService
	transformer  accessorialchargeports.DataTransformer
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.documenttype"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		transformer:  p.Transformer,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *documenttypeports.ListDocumentTypesRequest,
) (*pagination.ListResult[*documenttype.DocumentType], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *documenttypeports.ListDocumentTypeConnectionRequest,
) (*pagination.CursorListResult[*documenttype.DocumentType], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req documenttypeports.GetDocumentTypeByIDRequest,
) (*documenttype.DocumentType, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*documenttype.DocumentType], error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *documenttype.DocumentType,
	userID pulid.ID,
) (*documenttype.DocumentType, error) {
	if err := s.transformer.TransformDocumentType(ctx, entity); err != nil {
		s.l.Error("failed to transform document type", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		s.l.Error("failed to create document type", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceDocumentType,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Document type created"),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *documenttype.DocumentType,
	userID pulid.ID,
) (*documenttype.DocumentType, error) {
	if err := s.transformer.TransformDocumentType(ctx, entity); err != nil {
		s.l.Error("failed to transform document type", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, documenttypeports.GetDocumentTypeByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		s.l.Error("failed to get original document type", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		s.l.Error("failed to update document type", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceDocumentType,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Document type updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}
