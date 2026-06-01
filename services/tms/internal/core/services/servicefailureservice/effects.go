package servicefailureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/zap"
)

type serviceFailureActionParams struct {
	entity   *servicefailure.ServiceFailure
	actor    *services.RequestActor
	op       permission.Operation
	previous any
	current  any
	comment  string
}

type serviceFailureInvalidationParams struct {
	entity  *servicefailure.ServiceFailure
	actor   *services.RequestActor
	action  string
	payload any
}

type commentParams struct {
	entity   *servicefailure.ServiceFailure
	comment  string
	metadata map[string]any
}

func (s *service) afterServiceFailureCreate(
	ctx context.Context,
	entity *servicefailure.ServiceFailure,
	actor *services.RequestActor,
	comment string,
) {
	s.logServiceFailureAction(serviceFailureActionParams{
		entity:  entity,
		actor:   actor,
		op:      permission.OpCreate,
		current: entity,
		comment: comment,
	})
	s.publishInvalidation(ctx, serviceFailureInvalidationParams{
		entity:  entity,
		actor:   actor,
		action:  "created",
		payload: entity,
	})
}

func (s *service) logServiceFailureAction(params serviceFailureActionParams) {
	auditActor := params.actor.AuditActorOrSystem()
	logParams := &services.LogActionParams{
		Resource:       permission.ResourceServiceFailure,
		ResourceID:     params.entity.ID.String(),
		Operation:      params.op,
		UserID:         auditActor.UserID,
		APIKeyID:       auditActor.APIKeyID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		OrganizationID: params.entity.OrganizationID,
		BusinessUnitID: params.entity.BusinessUnitID,
	}
	if params.current != nil {
		logParams.CurrentState = jsonutils.MustToJSON(params.current)
	}
	if params.previous != nil {
		logParams.PreviousState = jsonutils.MustToJSON(params.previous)
	}

	opts := []services.LogOption{
		auditservice.WithComment(params.comment),
		auditservice.WithMetadata(serviceFailureMetadata(params.entity, params.actor)),
	}
	if params.previous != nil && params.current != nil {
		opts = append(opts, auditservice.WithDiff(params.previous, params.current))
	}
	if err := s.auditService.LogAction(logParams, opts...); err != nil {
		s.l.Warn("failed to log service failure audit", zap.Error(err))
	}
}

func serviceFailureLifecycleMetadata(
	previous *servicefailure.ServiceFailure,
	current *servicefailure.ServiceFailure,
	actor *services.RequestActor,
) map[string]any {
	metadata := serviceFailureMetadata(current, actor)
	if previous != nil {
		metadata["previousStatus"] = string(previous.Status)
		metadata["previousReasonCodeId"] = optionalIDString(previous.ReasonCodeID)
		if previous.ReasonCode != nil {
			metadata["previousReasonCode"] = previous.ReasonCode.Code
			metadata["previousReasonLabel"] = previous.ReasonCode.Label
		}
	}
	return metadata
}

func serviceFailureMetadata(
	entity *servicefailure.ServiceFailure,
	actor *services.RequestActor,
) map[string]any {
	metadata := map[string]any{}
	if entity == nil {
		return metadata
	}
	metadata["serviceFailureId"] = entity.ID.String()
	metadata["serviceFailureNumber"] = entity.Number
	metadata["shipmentId"] = entity.ShipmentID.String()
	metadata["stopId"] = entity.StopID.String()
	metadata["stopType"] = string(entity.StopType)
	metadata["source"] = string(entity.Source)
	metadata["status"] = string(entity.Status)
	metadata["lateMinutes"] = entity.LateMinutes
	metadata["reasonCodeId"] = optionalIDString(entity.ReasonCodeID)
	metadata["x12StatusCode"] = entity.X12StatusCodeOverride
	metadata["x12ReasonCode"] = entity.X12ReasonCodeOverride
	if entity.Stop != nil {
		metadata["stopSequence"] = entity.Stop.Sequence
	}
	if entity.ReasonCode != nil {
		metadata["reasonCode"] = entity.ReasonCode.Code
		metadata["reasonLabel"] = entity.ReasonCode.Label
		metadata["reasonCategory"] = string(entity.ReasonCode.Category)
		metadata["reasonAppliesTo"] = string(entity.ReasonCode.AppliesTo)
		if entity.X12ReasonCodeOverride == "" {
			metadata["x12ReasonCode"] = entity.ReasonCode.DefaultReasonCode
		}
		if entity.X12StatusCodeOverride == "" {
			metadata["x12StatusCode"] = entity.ReasonCode.DefaultStatusCode
		}
	}
	auditActor := actor.AuditActorOrSystem()
	metadata["principalType"] = string(auditActor.PrincipalType)
	metadata["principalId"] = auditActor.PrincipalID.String()
	if auditActor.UserID.IsNotNil() {
		metadata["userId"] = auditActor.UserID.String()
	}
	if auditActor.APIKeyID.IsNotNil() {
		metadata["apiKeyId"] = auditActor.APIKeyID.String()
	}
	return metadata
}

func (s *service) publishInvalidation(
	ctx context.Context,
	params serviceFailureInvalidationParams,
) {
	auditActor := params.actor.AuditActorOrSystem()
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: params.entity.OrganizationID,
		BusinessUnitID: params.entity.BusinessUnitID,
		ActorUserID:    auditActor.UserID,
		ActorType:      auditActor.PrincipalType,
		ActorID:        auditActor.PrincipalID,
		ActorAPIKeyID:  auditActor.APIKeyID,
		Resource:       permission.ResourceServiceFailure.String(),
		Action:         params.action,
		RecordID:       params.entity.ShipmentID,
		Entity:         params.payload,
	})
	if err != nil {
		s.l.Warn("failed to publish service failure invalidation", zap.Error(err))
	}
}

func (s *service) comment(ctx context.Context, params commentParams) {
	if s.commentService == nil {
		return
	}
	if params.metadata == nil {
		params.metadata = map[string]any{}
	}
	params.metadata["serviceFailureNumber"] = params.entity.Number
	params.metadata["serviceFailureStatus"] = string(params.entity.Status)
	if _, err := s.commentService.CreateSystem(ctx, &services.CreateSystemShipmentCommentRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: params.entity.OrganizationID,
			BuID:  params.entity.BusinessUnitID,
		},
		ShipmentID: params.entity.ShipmentID,
		Comment:    params.comment,
		Type:       shipment.CommentTypeException,
		Visibility: shipment.CommentVisibilityOperations,
		Priority:   shipment.CommentPriorityHigh,
		Metadata:   params.metadata,
	}); err != nil {
		s.l.Warn("failed to create service failure shipment comment", zap.Error(err))
	}
}
