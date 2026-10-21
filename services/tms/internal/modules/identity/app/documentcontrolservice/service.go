package documentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
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
	Repo         tenantports.DocumentControlRepository
	Validator    *Validator
	AuditService auditports.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         tenantports.DocumentControlRepository
	validator    *Validator
	auditService auditports.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.documentcontrol"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req tenantports.GetDocumentControlRequest,
) (*tenant.DocumentControl, error) {
	return s.repo.GetOrCreate(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID)
}

func (s *Service) Update(
	ctx context.Context,
	entity *tenant.DocumentControl,
	userID pulid.ID,
) (*tenant.DocumentControl, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
	)

	original, err := s.repo.GetOrCreate(ctx, entity.OrganizationID, entity.BusinessUnitID)
	if err != nil {
		log.Error("failed to get original document control", zap.Error(err))
		return nil, err
	}

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update document control", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(
		&auditports.LogActionParams{
			Resource:       permission.ResourceDocumentControl,
			ResourceID:     updatedEntity.GetID().String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		auditservice.WithComment("Document control updated"),
		auditservice.WithDiff(original, updatedEntity),
		auditservice.WithCritical(),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}
