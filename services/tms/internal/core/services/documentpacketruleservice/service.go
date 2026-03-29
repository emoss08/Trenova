package documentpacketruleservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documentpacketrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
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
	Repo         repositories.DocumentPacketRuleRepository
	AuditService services.AuditService
}

type Service struct {
	logger       *zap.Logger
	repo         repositories.DocumentPacketRuleRepository
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		logger:       p.Logger.Named("service.document-packet-rule"),
		repo:         p.Repo,
		auditService: p.AuditService,
	}
}

func (s *Service) ListByResourceType(
	ctx context.Context,
	resourceType string,
	tenantInfo pagination.TenantInfo,
) ([]*documentpacketrule.Rule, error) {
	return s.repo.ListByResourceType(ctx, &repositories.ListDocumentPacketRulesByResourceRequest{
		TenantInfo:   tenantInfo,
		ResourceType: resourceType,
	})
}

func (s *Service) Create(
	ctx context.Context,
	entity *documentpacketrule.Rule,
	userID pulid.ID,
) (*documentpacketrule.Rule, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	_ = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocumentType,
		ResourceID:     created.ID.String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(created),
		OrganizationID: created.OrganizationID,
		BusinessUnitID: created.BusinessUnitID,
	}, auditservice.WithComment("Document packet rule created"))

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *documentpacketrule.Rule,
	userID pulid.ID,
) (*documentpacketrule.Rule, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, repositories.GetDocumentPacketRuleByIDRequest{
		ID: entity.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	_ = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocumentType,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(original),
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("Document packet rule updated"))

	return updated, nil
}

func (s *Service) Delete(
	ctx context.Context,
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) error {
	original, err := s.repo.GetByID(ctx, repositories.GetDocumentPacketRuleByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	if err = s.repo.Delete(ctx, repositories.GetDocumentPacketRuleByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		return err
	}

	_ = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceDocumentType,
		ResourceID:     original.ID.String(),
		Operation:      permission.OpDelete,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: original.OrganizationID,
		BusinessUnitID: original.BusinessUnitID,
	}, auditservice.WithComment("Document packet rule deleted"))

	return nil
}
