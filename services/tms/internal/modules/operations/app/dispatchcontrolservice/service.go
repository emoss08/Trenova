package dispatchcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         dispatchcontrolports.DispatchControlRepository
	Validator    *Validator
	AuditService auditports.AuditService
	Fields       permissionports.FieldAccess `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         dispatchcontrolports.DispatchControlRepository
	validator    *Validator
	auditService auditports.AuditService
	fields       permissionports.FieldAccess
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.dispatchcontrol"),
		repo:         p.Repo,
		validator:    p.Validator,
		auditService: p.AuditService,
		fields:       p.Fields,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req dispatchcontrolports.GetDispatchControlRequest,
) (*dispatchcontrol.DispatchControl, error) {
	return s.repo.GetByOrgID(ctx, req)
}

func (s *Service) Update(
	ctx context.Context,
	entity *dispatchcontrol.DispatchControl,
	actor *permissionports.RequestActor,
) (*dispatchcontrol.DispatchControl, error) {
	userID := actor.UserID
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	original, err := s.repo.GetByOrgID(ctx, dispatchcontrolports.GetDispatchControlRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  entity.OrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: userID,
		},
	})
	if err != nil {
		log.Error("failed to get original dispatch control", zap.Error(err))
		return nil, err
	}
	entity.KeepFields(original, permissionports.VisibleFieldGroups(ctx, s.fields, actor))

	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update dispatch control", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(
		&auditports.LogActionParams{
			Resource:       permission.ResourceDispatchControl,
			ResourceID:     updatedEntity.GetID().String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		auditservice.WithComment("Dispatch control updated"),
		auditservice.WithDiff(original, updatedEntity),
		auditservice.WithCritical(),
	); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}
