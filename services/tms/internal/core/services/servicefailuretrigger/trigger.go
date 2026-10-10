package servicefailuretrigger

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func ActorFromTenant(tenantInfo pagination.TenantInfo) *permissionports.RequestActor {
	if tenantInfo.UserID.IsNil() {
		return nil
	}
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    tenantInfo.UserID,
		UserID:         tenantInfo.UserID,
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
	}
}

func EvaluateShipment(
	ctx context.Context,
	evaluator servicefailureports.ServiceFailureEvaluator,
	shipmentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	actor *permissionports.RequestActor,
) error {
	if evaluator == nil || shipmentID.IsNil() {
		return nil
	}
	_, err := evaluator.EvaluateShipment(ctx, &servicefailureports.EvaluateShipmentServiceFailuresRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: tenantInfo.OrgID,
			BuID:  tenantInfo.BuID,
		},
		ShipmentID: shipmentID,
	}, actor)
	return err
}
