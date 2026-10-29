package factoringaction

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

type TenantRecord interface {
	GetID() pulid.ID
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
}

type Recorder struct {
	Audit    auditports.AuditService
	Realtime realtimeports.RealtimeService
	Logger   *zap.Logger
}

type AuditEntry struct {
	Resource permission.Resource
	Entity   TenantRecord
	Previous TenantRecord
	Op       permission.Operation
	Comment  string
}

func (r *Recorder) Log(actor *permissionports.RequestActor, entry *AuditEntry) {
	if r.Audit == nil || entry.Entity == nil || actor == nil {
		return
	}
	params := &auditports.LogActionParams{
		Resource:       entry.Resource,
		ResourceID:     entry.Entity.GetID().String(),
		Operation:      entry.Op,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		OrganizationID: entry.Entity.GetOrganizationID(),
		BusinessUnitID: entry.Entity.GetBusinessUnitID(),
		CurrentState:   jsonutils.MustToJSON(entry.Entity),
	}
	opts := []auditports.LogOption{auditservice.WithComment(entry.Comment)}
	if entry.Previous != nil {
		params.PreviousState = jsonutils.MustToJSON(entry.Previous)
		opts = append(opts, auditservice.WithDiff(entry.Previous, entry.Entity))
	}
	if err := r.Audit.LogAction(params, opts...); err != nil {
		r.Logger.Warn("failed to log factoring audit",
			zap.String("resource", entry.Resource.String()),
			zap.Error(err))
	}
}

func (r *Recorder) Publish(
	ctx context.Context,
	resource permission.Resource,
	entity TenantRecord,
	actor *permissionports.RequestActor,
	action string,
) {
	if r.Realtime == nil || entity == nil {
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
	if err := realtimeinvalidation.Publish(ctx, r.Realtime, params); err != nil {
		r.Logger.Warn("failed to publish factoring invalidation",
			zap.String("resource", resource.String()),
			zap.Error(err))
	}
}

func RequireActor(actor *permissionports.RequestActor, action string) error {
	if actor == nil || actor.PrincipalID.IsNil() && actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError(action + " requires an authenticated principal")
	}
	return nil
}

func RequireRequest(present bool) error {
	if present {
		return nil
	}
	return errortypes.NewValidationError("request", errortypes.ErrRequired, "Request is required")
}

func ValidationOf(multiErr *errortypes.MultiError) error {
	if multiErr != nil && multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
