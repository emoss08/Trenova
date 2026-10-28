package dashcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         tenantports.DashControlRepository
	AuditService auditports.AuditService
	Realtime     realtimeports.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         tenantports.DashControlRepository
	auditService auditports.AuditService
	realtime     realtimeports.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.dash-control"),
		repo:         p.Repo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.DashControl, error) {
	return s.repo.GetOrCreate(ctx, tenantInfo)
}

func (s *Service) Update(
	ctx context.Context,
	entity *tenant.DashControl,
	userID pulid.ID,
) (*tenant.DashControl, error) {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	original, err := s.repo.GetOrCreate(ctx, pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}
	entity.ID = original.ID

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	params := &auditports.LogActionParams{
		Resource:       permission.ResourceDashControl,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}
	if logErr := s.auditService.LogAction(
		params,
		auditservice.WithComment("Dash control updated"),
		auditservice.WithDiff(original, updated),
	); logErr != nil {
		s.l.Error("failed to log dash control audit action", zap.Error(logErr))
	}

	s.publishInvalidation(ctx, updated, userID)

	return updated, nil
}

func (s *Service) publishInvalidation(
	ctx context.Context,
	control *tenant.DashControl,
	userID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: control.OrganizationID,
		BusinessUnitID: control.BusinessUnitID,
		ActorUserID:    userID,
		ActorType:      permissionports.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       "dash_control",
		Action:         string(permission.OpUpdate),
		RecordID:       control.ID,
	})
	if err != nil {
		s.l.Warn("failed to publish dash control invalidation", zap.Error(err))
	}
}
