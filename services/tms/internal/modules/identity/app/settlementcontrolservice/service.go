package settlementcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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
	Repo         tenantports.SettlementControlRepository
	AuditService auditports.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         tenantports.SettlementControlRepository
	auditService auditports.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.settlement-control"),
		repo:         p.Repo,
		auditService: p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.SettlementControl, error) {
	return s.repo.GetOrCreate(ctx, tenantInfo)
}

func (s *Service) Update(
	ctx context.Context,
	entity *tenant.SettlementControl,
	userID pulid.ID,
) (*tenant.SettlementControl, error) {
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
		Resource:       permission.ResourceSettlementControl,
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
		auditservice.WithComment("Settlement control updated"),
		auditservice.WithDiff(original, updated),
	); logErr != nil {
		s.l.Error("failed to log settlement control audit action", zap.Error(logErr))
	}

	return updated, nil
}
