package invoiceaction

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerealtime"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func ActorUserID(actor *permissionports.RequestActor, tenantInfo pagination.TenantInfo) pulid.ID {
	if actor != nil && actor.UserID.IsNotNil() {
		return actor.UserID
	}
	return tenantInfo.UserID
}

type PostedBillingQueueResult struct {
	Previous *billingqueue.BillingQueueItem
	Updated  *billingqueue.BillingQueueItem
}

func (s *Recorder) LogAction(
	entity *invoice.Invoice,
	actor kernelports.AuditActor,
	op permission.Operation,
	previous any,
	current any,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceInvoice,
		ResourceID:     entity.ID.String(),
		Operation:      op,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}
	if current != nil {
		params.CurrentState = jsonutils.MustToJSON(current)
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	opts := []auditports.LogOption{
		auditservice.WithComment(comment),
		auditservice.WithMetadata(map[string]any{
			"shipmentId":         entity.ShipmentID.String(),
			"billingQueueItemId": entity.BillingQueueItemID.String(),
			"invoiceNumber":      entity.Number,
			"status":             entity.Status,
		}),
	}
	if previous != nil && current != nil {
		opts = append(opts, auditservice.WithDiff(previous, current))
	}

	if err := s.auditService.LogAction(params, opts...); err != nil {
		s.l.Error("failed to log invoice action", zap.Error(err))
	}
}

func (s *Recorder) PublishInvalidation(
	ctx context.Context,
	entity *invoice.Invoice,
	actor kernelports.AuditActor,
	action string,
	payload any,
) {
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceInvoice.String(),
		Action:         action,
		RecordID:       entity.ID,
		Entity:         payload,
	}); err != nil {
		s.l.Warn("failed to publish invoice invalidation", zap.Error(err))
	}
}

func (s *Recorder) PublishInvoiceChange(
	ctx context.Context,
	entity *invoice.Invoice,
	actor kernelports.AuditActor,
	action string,
) {
	if entity == nil {
		return
	}
	realtimeinvalidation.PublishAfterCommit(ctx, s.realtime, s.l, realtimeinvalidation.ChangeOf(
		permission.ResourceInvoice.String(),
		entity,
		action,
		actor,
		invoicerealtime.NewRealtimeSnapshot(entity),
	))
}

func (s *Recorder) PublishBillingQueueInvalidation(
	ctx context.Context,
	result *PostedBillingQueueResult,
	actor kernelports.AuditActor,
) {
	if result == nil || result.Previous == nil || result.Updated == nil {
		return
	}

	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: result.Updated.OrganizationID,
		BusinessUnitID: result.Updated.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       permission.ResourceBillingQueue.String(),
		Action:         "updated",
		RecordID:       result.Updated.ID,
		Entity:         result.Updated,
	}); err != nil {
		s.l.Warn("failed to publish billing queue invalidation", zap.Error(err))
	}
}
