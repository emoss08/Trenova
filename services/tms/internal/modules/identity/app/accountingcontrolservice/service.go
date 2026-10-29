package accountingcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         tenantports.AccountingControlRepository
	Validator    *Validator
	AuditService auditports.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         tenantports.AccountingControlRepository
	validator    *Validator
	auditService auditports.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.accountingcontrol"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req tenantports.GetAccountingControlRequest,
) (*accountingcontrol.AccountingControl, error) {
	return s.repo.GetByOrgID(ctx, req.TenantInfo.OrgID)
}

func (s *Service) Update(
	ctx context.Context,
	entity *accountingcontrol.AccountingControl,
	userID pulid.ID,
) (*accountingcontrol.AccountingControl, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	original, err := s.repo.GetByOrgID(ctx, entity.OrganizationID)
	if err != nil {
		log.Error("failed to get original accounting control", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update accounting control", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceAccountingControl,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Accounting control updated"),
		auditservice.WithDiff(original, updatedEntity),
		auditservice.WithCritical(),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}
