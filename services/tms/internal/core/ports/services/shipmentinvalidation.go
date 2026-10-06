package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ShipmentInvalidation struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ActorUserID    pulid.ID
	ActorType      PrincipalType
	ActorID        pulid.ID
	ActorAPIKeyID  pulid.ID
	Action         string
	RecordID       pulid.ID
	Entity         any
}

func ShipmentInvalidationByUser(
	tenantInfo pagination.TenantInfo,
	recordID pulid.ID,
	action string,
) *ShipmentInvalidation {
	return &ShipmentInvalidation{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    tenantInfo.UserID,
		ActorType:      PrincipalTypeUser,
		ActorID:        tenantInfo.UserID,
		Action:         action,
		RecordID:       recordID,
	}
}

func ShipmentInvalidationByActor(
	tenantInfo pagination.TenantInfo,
	actor AuditActor,
	recordID pulid.ID,
	action string,
) *ShipmentInvalidation {
	return &ShipmentInvalidation{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Action:         action,
		RecordID:       recordID,
	}
}

type ShipmentInvalidator interface {
	InvalidateShipments(ctx context.Context, req *ShipmentInvalidation)
}

type ShipmentRecord interface {
	GetID() pulid.ID
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
}

func ShipmentInvalidationForRecord(
	record ShipmentRecord,
	actor AuditActor,
	action string,
	entity any,
) *ShipmentInvalidation {
	req := ShipmentInvalidationByActor(
		pagination.TenantInfo{
			OrgID: record.GetOrganizationID(),
			BuID:  record.GetBusinessUnitID(),
		},
		actor,
		record.GetID(),
		action,
	)
	req.Entity = entity
	return req
}

func InvalidateShipments(
	ctx context.Context,
	invalidator ShipmentInvalidator,
	req *ShipmentInvalidation,
) {
	if invalidator == nil {
		return
	}
	invalidator.InvalidateShipments(ctx, req)
}
