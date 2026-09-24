package realtimeservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ErrPublishBackpressure reports an event the bus had no room for. The write
// that caused it has already happened; only the notice to open screens is lost,
// and those screens refetch on their next reconnect or focus.
var ErrPublishBackpressure = errors.New("realtime event dropped: publish queue is full")

type Params struct {
	fx.In

	Logger    *zap.Logger
	Config    *config.Config
	Publisher services.RealtimePublisher
}

type Service struct {
	l              *zap.Logger
	publisher      services.RealtimePublisher
	maxEntityBytes int
}

func New(p Params) services.RealtimeService {
	return &Service{
		l:              p.Logger.Named("service.realtime"),
		publisher:      p.Publisher,
		maxEntityBytes: p.Config.GetRealtimeConfig().GetMaxEntityBytes(),
	}
}

// PublishResourceInvalidation tells every open screen in the tenant that a
// record changed. It never waits on the network: the event is queued and
// written in the background, so a bulk operation touching thousands of records
// pays nothing for the screens watching it.
func (s *Service) PublishResourceInvalidation(
	_ context.Context,
	req *services.PublishResourceInvalidationRequest,
) error {
	if req == nil {
		return errortypes.NewBusinessError("publish invalidation request is required")
	}

	if req.OrganizationID.IsNil() || req.BusinessUnitID.IsNil() {
		return errortypes.NewBusinessError("invalid realtime tenant context")
	}

	if req.Resource == "" || req.Action == "" {
		return errortypes.NewBusinessError("invalid realtime invalidation payload")
	}

	event := buildInvalidationEvent(req)

	payload, err := s.encode(&event)
	if err != nil {
		return err
	}

	portal, err := redactedPayload(&event, req.AudienceUserID)
	if err != nil {
		return err
	}

	if !s.publisher.Enqueue(&services.RealtimeEnvelope{
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		AudienceUserID: req.AudienceUserID,
		Event:          services.RealtimeEventInvalidation,
		Payload:        payload,
		PortalPayload:  portal,
	}) {
		return ErrPublishBackpressure
	}

	return nil
}

// encode writes the event, leaving the entity off when it is larger than a
// stream frame should carry. A reader without the entity refetches the record,
// which is what it would do for any event it cannot patch from.
func (s *Service) encode(event *services.ResourceInvalidationEvent) ([]byte, error) {
	payload, err := sonic.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("encode realtime invalidation: %w", err)
	}
	if event.Entity == nil || len(payload) <= s.maxEntityBytes {
		return payload, nil
	}

	s.l.Debug("realtime entity exceeds frame budget; sending without it",
		zap.String("resource", event.Resource),
		zap.Int("bytes", len(payload)),
	)

	slim := *event
	slim.Entity = nil
	payload, err = sonic.Marshal(&slim)
	if err != nil {
		return nil, fmt.Errorf("encode realtime invalidation: %w", err)
	}
	return payload, nil
}

// redactedPayload is what a portal user sees of an event not addressed to
// them: that a record of a kind changed, never its contents. An event
// addressed to one person is delivered whole to that person and needs no
// redacted form.
func redactedPayload(
	event *services.ResourceInvalidationEvent,
	audience pulid.ID,
) ([]byte, error) {
	if audience.IsNotNil() {
		return nil, nil
	}

	redacted := *event
	redacted.Entity = nil
	redacted.Fields = nil
	payload, err := sonic.Marshal(&redacted)
	if err != nil {
		return nil, fmt.Errorf("encode redacted realtime invalidation: %w", err)
	}
	return payload, nil
}

func buildInvalidationEvent(
	req *services.PublishResourceInvalidationRequest,
) services.ResourceInvalidationEvent {
	eventType := strings.TrimSpace(req.EventType)
	if eventType == "" {
		eventType = req.Resource + "." + req.Action
	}

	event := services.ResourceInvalidationEvent{
		EventID:        pulid.MustNew("evt_").String(),
		OrganizationID: req.OrganizationID.String(),
		BusinessUnitID: req.BusinessUnitID.String(),
		Type:           eventType,
		Resource:       req.Resource,
		Action:         req.Action,
		Fields:         req.Fields,
		EntityVersion:  req.EntityVersion,
		Entity:         req.Entity,
		OccurredAt:     time.Now().UTC(),
	}

	if req.RecordID.IsNotNil() {
		event.EntityID = req.RecordID.String()
		event.RecordID = req.RecordID.String()
	}
	if req.ActorUserID.IsNotNil() {
		event.ActorUserID = req.ActorUserID.String()
	}
	if req.ActorType != "" {
		event.ActorType = string(req.ActorType)
	}
	if req.ActorID.IsNotNil() {
		event.ActorID = req.ActorID.String()
	}
	if req.ActorAPIKeyID.IsNotNil() {
		event.ActorAPIKeyID = req.ActorAPIKeyID.String()
	}

	return event
}
