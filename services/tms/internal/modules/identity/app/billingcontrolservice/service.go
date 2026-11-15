package billingcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
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

	Logger           *zap.Logger
	Repo             tenantports.BillingControlRepository
	Validator        *Validator
	AuditService     auditports.AuditService
	PricingTemplates permissionports.PricingTemplatePlan
}

type Service struct {
	l                *zap.Logger
	repo             tenantports.BillingControlRepository
	validator        *Validator
	auditService     auditports.AuditService
	pricingTemplates permissionports.PricingTemplatePlan
}

func New(p Params) *Service {
	return &Service{
		l:                p.Logger.Named("service.billingcontrol"),
		repo:             p.Repo,
		validator:        p.Validator,
		auditService:     p.AuditService,
		pricingTemplates: p.PricingTemplates,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req tenantports.GetBillingControlRequest,
) (*billingcontrol.BillingControl, error) {
	return s.repo.GetByOrgID(ctx, req.TenantInfo.OrgID)
}

func (s *Service) Update(
	ctx context.Context,
	entity *billingcontrol.BillingControl,
	userID pulid.ID,
) (*billingcontrol.BillingControl, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		log.Error("failed to get original billing control", zap.Error(err))
		return nil, err
	}
	if err = s.refuseOffPlanFallback(ctx, entity, original); err != nil {
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update billing control", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(ctx,
		&auditports.LogActionParams{
			Resource:       permission.ResourceBillingControl,
			ResourceID:     updatedEntity.GetID().String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		auditservice.WithComment("Billing control updated"),
		auditservice.WithDiff(original, updatedEntity),
		auditservice.WithCritical(),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) refuseOffPlanFallback(
	ctx context.Context,
	entity, original *billingcontrol.BillingControl,
) error {
	if entity.FallbackFormulaTemplateID == nil {
		return nil
	}
	if original != nil && original.FallbackFormulaTemplateID != nil &&
		*original.FallbackFormulaTemplateID == *entity.FallbackFormulaTemplateID {
		return nil
	}

	return s.pricingTemplates.RefuseOffPlan(
		ctx,
		pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		*entity.FallbackFormulaTemplateID,
	)
}
