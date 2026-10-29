package carriersettlementworkspaceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/carriersettlementcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/zap"
)

func (s *Service) GetControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*carriersettlementcontrol.CarrierSettlementControl, error) {
	return s.settlementControl.GetOrCreate(ctx, tenantInfo)
}

func (s *Service) UpdateControl(
	ctx context.Context,
	entity *carriersettlementcontrol.CarrierSettlementControl,
	actor *permissionports.RequestActor,
) (*carriersettlementcontrol.CarrierSettlementControl, error) {
	if err := settlementshared.RequireActor(actor, "Carrier settlement control update"); err != nil {
		return nil, err
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	}
	previous, err := s.settlementControl.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	entity.ID = previous.ID

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.settlementControl.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.publisher.Publish(
		ctx,
		permission.ResourceCarrierSettlementControl.String(),
		permission.OpUpdate,
		updated.ID,
		updated.OrganizationID,
		updated.BusinessUnitID,
		actor.UserID,
	)
	if err = s.auditService.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceCarrierSettlementControl,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		UserID:         actor.UserID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(previous),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("Carrier settlement control updated"),
		auditservice.WithDiff(previous, updated)); err != nil {
		s.l.Error("failed to log carrier settlement control audit action", zap.Error(err))
	}
	return updated, nil
}
