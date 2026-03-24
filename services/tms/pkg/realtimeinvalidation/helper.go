package realtimeinvalidation

import (
	"context"
	"errors"

	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrPublishParamsRequired = errors.New("publish params are required")

type PublishParams struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ActorUserID    pulid.ID
	ActorType      servicesport.PrincipalType
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
	realtime servicesport.RealtimeService,
	params *PublishParams,
) error {
	if params == nil {
		return ErrPublishParamsRequired
	}

	return realtime.PublishResourceInvalidation(
		ctx,
		&servicesport.PublishResourceInvalidationRequest{
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
