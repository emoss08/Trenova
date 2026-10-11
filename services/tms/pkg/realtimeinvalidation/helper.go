package realtimeinvalidation

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrPublishParamsRequired = errors.New("publish params are required")

type PublishParams struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ActorUserID    pulid.ID
	ActorType      kernelports.PrincipalType
	ActorID        pulid.ID
	ActorAPIKeyID  pulid.ID
	Resource       string
	Action         string
	RecordID       pulid.ID
	Entity         any
	Fields         []string
}

func Publish(
	ctx context.Context,
	realtime agentports.RealtimeService,
	params *PublishParams,
) error {
	if params == nil {
		return ErrPublishParamsRequired
	}

	return realtime.PublishResourceInvalidation(
		ctx,
		&agentports.PublishResourceInvalidationRequest{
			OrganizationID: params.OrganizationID,
			BusinessUnitID: params.BusinessUnitID,
			Resource:       params.Resource,
			Action:         params.Action,
			Fields:         params.Fields,
			Entity:         params.Entity,
			RecordID:       params.RecordID,
			ActorUserID:    params.ActorUserID,
			ActorType:      params.ActorType,
			ActorID:        params.ActorID,
			ActorAPIKeyID:  params.ActorAPIKeyID,
		},
	)
}
