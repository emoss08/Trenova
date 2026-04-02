package shipmentservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/equipmentcontinuityhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *service) advanceContinuityForCompletedMoves(
	ctx context.Context,
	original *shipment.Shipment,
	updated *shipment.Shipment,
) error {
	if s.continuityRepo == nil || updated == nil {
		return nil
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: updated.OrganizationID,
		BuID:  updated.BusinessUnitID,
	}

	originalMoveCap := 0
	if original != nil {
		originalMoveCap = len(original.Moves)
	}
	originalMoves := make(map[pulid.ID]*shipment.ShipmentMove, originalMoveCap)
	if original != nil {
		for _, move := range original.Moves {
			if move == nil || move.ID.IsNil() {
				continue
			}
			originalMoves[move.ID] = move
		}
	}

	for _, move := range updated.Moves {
		if move == nil || move.ID.IsNil() || move.Assignment == nil || !move.IsCompleted() {
			continue
		}

		if previous := originalMoves[move.ID]; previous != nil && previous.IsCompleted() {
			continue
		}

		if err := equipmentcontinuityhelper.AdvanceForCompletedMove(
			ctx,
			s.continuityRepo,
			tenantInfo,
			move,
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *service) logShipmentAction(
	entity shipmentTenantResource,
	actor services.AuditActor,
	operation permission.Operation,
	previous any,
	current any,
	opts ...services.LogOption,
) error {
	params := &services.LogActionParams{
		Resource:       permission.ResourceShipment,
		ResourceID:     entity.GetID().String(),
		Operation:      operation,
		UserID:         actor.UserID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		APIKeyID:       actor.APIKeyID,
		OrganizationID: entity.GetOrganizationID(),
		BusinessUnitID: entity.GetBusinessUnitID(),
	}
	if current != nil {
		params.CurrentState = jsonutils.MustToJSON(current)
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	return s.auditService.LogAction(params, opts...)
}

func (s *service) publishShipmentInvalidation(
	ctx context.Context,
	entity shipmentTenantResource,
	actor services.AuditActor,
	action string,
	payload any,
) error {
	return realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.GetOrganizationID(),
		BusinessUnitID: entity.GetBusinessUnitID(),
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       "shipments",
		Action:         action,
		RecordID:       entity.GetID(),
		Entity:         payload,
	})
}

func (s *service) publishBulkShipmentInvalidation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	actor services.AuditActor,
	action string,
) error {
	return realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       "shipments",
		Action:         action,
	})
}
