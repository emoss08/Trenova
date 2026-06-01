package servicefailuretrigger

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func ActorFromTenant(tenantInfo pagination.TenantInfo) *services.RequestActor {
	if tenantInfo.UserID.IsNil() {
		return nil
	}
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    tenantInfo.UserID,
		UserID:         tenantInfo.UserID,
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
	}
}

func EvaluateShipment(
	ctx context.Context,
	evaluator services.ServiceFailureEvaluator,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	actor *services.RequestActor,
) error {
	if evaluator == nil || shipmentID.IsNil() {
		return nil
	}
	_, err := evaluator.EvaluateShipment(ctx, &services.EvaluateShipmentServiceFailuresRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: tenantInfo.OrgID,
			BuID:  tenantInfo.BuID,
		},
		ShipmentID: shipmentID,
	}, actor)
	return err
}
