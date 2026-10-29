package carriersettlementrealtime

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

func (s *Publisher) Publish(
	ctx context.Context,
	resource string,
	action permission.Operation,
	recordID, orgID, buID, userID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ActorUserID:    userID,
		ActorType:      permissionports.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       resource,
		Action:         string(action),
		RecordID:       recordID,
	})
	if err != nil {
		s.l.Warn("failed to publish realtime invalidation",
			zap.String("resource", resource),
			zap.Error(err))
	}
}

func (s *Publisher) PublishCostEvent(
	ctx context.Context,
	entity *carriersettlement.CostEvent,
	action permission.Operation,
	userID pulid.ID,
) {
	if entity == nil {
		return
	}
	s.Publish(
		ctx,
		"carrier_cost_event",
		action,
		entity.ID,
		entity.OrganizationID,
		entity.BusinessUnitID,
		userID,
	)
}
