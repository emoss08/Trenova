package realtimeinvalidation

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type RecordChange struct {
	Resource       string
	Action         string
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	RecordID       pulid.ID
	Actor          kernelports.AuditActor
	Entity         any
}

type Record interface {
	GetID() pulid.ID
	GetOrganizationID() pulid.ID
	GetBusinessUnitID() pulid.ID
}

func ChangeOf(
	resource string,
	record Record,
	action string,
	actor kernelports.AuditActor,
	entity any,
) *RecordChange {
	return &RecordChange{
		Resource:       resource,
		Action:         action,
		OrganizationID: record.GetOrganizationID(),
		BusinessUnitID: record.GetBusinessUnitID(),
		RecordID:       record.GetID(),
		Actor:          actor,
		Entity:         entity,
	}
}

func (c *RecordChange) params() *PublishParams {
	return &PublishParams{
		OrganizationID: c.OrganizationID,
		BusinessUnitID: c.BusinessUnitID,
		ActorUserID:    c.Actor.UserID,
		ActorType:      c.Actor.PrincipalType,
		ActorID:        c.Actor.PrincipalID,
		ActorAPIKeyID:  c.Actor.APIKeyID,
		Resource:       c.Resource,
		Action:         c.Action,
		RecordID:       c.RecordID,
		Entity:         c.Entity,
	}
}

func PublishAfterCommit(
	ctx context.Context,
	realtime realtimeports.RealtimeService,
	logger *zap.Logger,
	change *RecordChange,
) {
	if realtime == nil || change == nil || change.RecordID.IsNil() {
		return
	}
	kernelports.AfterCommit(ctx, func(committedCtx context.Context) {
		if err := Publish(committedCtx, realtime, change.params()); err != nil && logger != nil {
			logger.Warn(
				"failed to publish realtime invalidation",
				zap.String("resource", change.Resource),
				zap.String("action", change.Action),
				zap.String("recordId", change.RecordID.String()),
				zap.Error(err),
			)
		}
	})
}
