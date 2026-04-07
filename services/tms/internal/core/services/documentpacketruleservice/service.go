package documentpacketruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
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
	Repo         repositories.DocumentPacketRuleRepository
	Validator    *Validator
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.DocumentPacketRuleRepository
	validator    *Validator
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.document-packet-rule"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListDocumentPacketRulesRequest,
) (*pagination.ListResult[*documentpacketrule.DocumentPacketRule], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListByResourceType(
	ctx context.Context,
	resourceType string,
	tenantInfo pagination.TenantInfo,
) ([]*documentpacketrule.DocumentPacketRule, error) {
	return s.repo.ListByResourceType(ctx, &repositories.ListDocumentPacketRulesByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceType: resourceType,
	})
}

func (s *Service) Create(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
	userID pulid.ID,
) (*documentpacketrule.DocumentPacketRule, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create document packet rule", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocumentType,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.GetOrganizationID(),
		BusinessUnitID: createdEntity.GetBusinessUnitID(),
	}, auditservice.WithComment("Document packet rule created")); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *documentpacketrule.DocumentPacketRule,
	userID pulid.ID,
) (*documentpacketrule.DocumentPacketRule, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("userID", userID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetDocumentPacketRuleByIDRequest{
		ID: entity.GetID(),
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.GetOrganizationID(),
			BuID:  entity.GetBusinessUnitID(),
		},
	})
	if err != nil {
		log.Error("failed to get original document packet rule", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update document packet rule", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocumentType,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(original),
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		OrganizationID: updatedEntity.GetOrganizationID(),
		BusinessUnitID: updatedEntity.GetBusinessUnitID(),
	}, auditservice.WithComment("Document packet rule updated"),
		auditservice.WithDiff(original, updatedEntity)); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "Delete"),
		zap.String("userID", userID.String()),
	)

	original, err := s.repo.GetByID(ctx, repositories.GetDocumentPacketRuleByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		log.Error("failed to get original document packet rule", zap.Error(err))
		return err
	}

	if err = s.repo.Delete(ctx, repositories.GetDocumentPacketRuleByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		log.Error("failed to delete document packet rule", zap.Error(err))
		return err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocumentType,
		ResourceID:     original.GetID().String(),
		Operation:      permission.OpDelete,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: original.GetOrganizationID(),
		BusinessUnitID: original.GetBusinessUnitID(),
	}, auditservice.WithComment("Document packet rule deleted"),
		auditservice.WithDiff(original, nil)); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return nil
}
