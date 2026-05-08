package shipmenteventservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipmentevent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const realtimeResource = "shipmentEvents"

var ErrRecordParamsRequired = errors.New("shipment event record params are required")

type Params struct {
	fx.In

	Logger   *zap.Logger
	Repo     repositories.ShipmentEventRepository
	Realtime services.RealtimeService
}

type service struct {
	l        *zap.Logger
	repo     repositories.ShipmentEventRepository
	realtime services.RealtimeService
}

func New(p Params) services.ShipmentEventService {
	return &service{
		l:        p.Logger.Named("service.shipment-event"),
		repo:     p.Repo,
		realtime: p.Realtime,
	}
}

func (s *service) Record(
	ctx context.Context,
	params *services.RecordShipmentEventParams,
) error {
	if params == nil {
		return ErrRecordParamsRequired
	}

	entity, err := buildEntity(params)
	if err != nil {
		return err
	}

	if err = s.repo.Insert(ctx, entity); err != nil {
		return err
	}

	if pubErr := realtimeinvalidation.Publish(
		ctx,
		s.realtime,
		&realtimeinvalidation.PublishParams{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			ActorUserID:    params.Actor.UserID,
			ActorType:      params.Actor.PrincipalType,
			ActorID:        params.Actor.PrincipalID,
			ActorAPIKeyID:  params.Actor.APIKeyID,
			Resource:       realtimeResource,
			Action:         "created",
			RecordID:       entity.ID,
			Entity:         entity,
		},
	); pubErr != nil {
		s.l.Warn("failed to publish shipment event invalidation", zap.Error(pubErr))
	}

	return nil
}

func (s *service) List(
	ctx context.Context,
	req *repositories.ListShipmentEventsRequest,
) ([]*shipmentevent.Event, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"List request is required",
		)
	}
	if multiErr := req.Validate(); multiErr != nil {
		return nil, multiErr
	}

	return s.repo.List(ctx, req)
}

func buildEntity(params *services.RecordShipmentEventParams) (*shipmentevent.Event, error) {
	if params.OrganizationID.IsNil() {
		return nil, errortypes.NewValidationError(
			"organizationId",
			errortypes.ErrRequired,
			"Organization ID is required",
		)
	}
	if params.BusinessUnitID.IsNil() {
		return nil, errortypes.NewValidationError(
			"businessUnitId",
			errortypes.ErrRequired,
			"Business unit ID is required",
		)
	}
	if params.ShipmentID.IsNil() {
		return nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrRequired,
			"Shipment ID is required",
		)
	}
	if params.Type == "" {
		return nil, errortypes.NewValidationError(
			"type",
			errortypes.ErrRequired,
			"Event type is required",
		)
	}
	if params.Summary == "" {
		return nil, errortypes.NewValidationError(
			"summary",
			errortypes.ErrRequired,
			"Event summary is required",
		)
	}

	severity := params.Severity
	if severity == "" {
		severity = shipmentevent.SeverityMuted
	}

	occurredAt := params.OccurredAt
	if occurredAt == 0 {
		occurredAt = timeutils.NowUnix()
	}

	actorType, actorID := resolveActor(params.Actor)

	return &shipmentevent.Event{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		ShipmentID:     params.ShipmentID,
		MoveID:         params.MoveID,
		StopID:         params.StopID,
		AssignmentID:   params.AssignmentID,
		CommentID:      params.CommentID,
		HoldID:         params.HoldID,
		Type:           params.Type,
		Severity:       severity,
		ActorType:      actorType,
		ActorID:        actorID,
		ActorLabel:     params.ActorLabel,
		Summary:        params.Summary,
		Metadata:       params.Metadata,
		OccurredAt:     occurredAt,
		CorrelationID:  params.CorrelationID,
	}, nil
}

func resolveActor(actor services.AuditActor) (shipmentevent.ActorType, pulid.ID) {
	switch actor.PrincipalType {
	case services.PrincipalTypeUser:
		return shipmentevent.ActorUser, actor.UserID
	case services.PrincipalTypeAPIKey:
		return shipmentevent.ActorAPIKey, actor.APIKeyID
	default:
		return shipmentevent.ActorSystem, pulid.Nil
	}
}
