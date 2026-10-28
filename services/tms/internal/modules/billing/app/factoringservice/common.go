package factoringservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type tenantRecord interface {
	GetID() pulid.ID
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
}

type recorder struct {
	audit    auditports.AuditService
	realtime realtimeports.RealtimeService
	l        *zap.Logger
}

type auditEntry struct {
	resource permission.Resource
	entity   tenantRecord
	previous tenantRecord
	op       permission.Operation
	comment  string
}

func (r *recorder) log(actor *permissionports.RequestActor, entry *auditEntry) {
	if r.audit == nil || entry.entity == nil || actor == nil {
		return
	}
	params := &auditports.LogActionParams{
		Resource:       entry.resource,
		ResourceID:     entry.entity.GetID().String(),
		Operation:      entry.op,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		OrganizationID: entry.entity.GetOrganizationID(),
		BusinessUnitID: entry.entity.GetBusinessUnitID(),
		CurrentState:   jsonutils.MustToJSON(entry.entity),
	}
	opts := []auditports.LogOption{auditservice.WithComment(entry.comment)}
	if entry.previous != nil {
		params.PreviousState = jsonutils.MustToJSON(entry.previous)
		opts = append(opts, auditservice.WithDiff(entry.previous, entry.entity))
	}
	if err := r.audit.LogAction(params, opts...); err != nil {
		r.l.Warn("failed to log factoring audit",
			zap.String("resource", entry.resource.String()),
			zap.Error(err))
	}
}

func (r *recorder) publish(
	ctx context.Context,
	resource permission.Resource,
	entity tenantRecord,
	actor *permissionports.RequestActor,
	action string,
) {
	if r.realtime == nil || entity == nil {
		return
	}
	params := &realtimeinvalidation.PublishParams{
		OrganizationID: entity.GetOrganizationID(),
		BusinessUnitID: entity.GetBusinessUnitID(),
		Resource:       resource.String(),
		Action:         action,
		RecordID:       entity.GetID(),
		Entity:         entity,
	}
	if actor != nil {
		params.ActorUserID = actor.UserID
		params.ActorType = actor.PrincipalType
		params.ActorID = actor.PrincipalID
		params.ActorAPIKeyID = actor.APIKeyID
	}
	if err := realtimeinvalidation.Publish(ctx, r.realtime, params); err != nil {
		r.l.Warn("failed to publish factoring invalidation",
			zap.String("resource", resource.String()),
			zap.Error(err))
	}
}

func requireActor(actor *permissionports.RequestActor, action string) error {
	if actor == nil || actor.PrincipalID.IsNil() && actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError(action + " requires an authenticated principal")
	}
	return nil
}

func requireRequest(present bool) error {
	if present {
		return nil
	}
	return errortypes.NewValidationError("request", errortypes.ErrRequired, "Request is required")
}

func validationOf(multiErr *errortypes.MultiError) error {
	if multiErr != nil && multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
