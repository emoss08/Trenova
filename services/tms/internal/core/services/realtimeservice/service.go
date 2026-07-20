package realtimeservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	realtime "github.com/Foony-Limited/realtime-go"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultTokenTTL = time.Hour
const resourceInvalidationEventName = "resource.invalidation"

type Params struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
	Client *realtime.Rest
}

type Service struct {
	l      *zap.Logger
	apiKey string
	client *realtime.Rest
}

func New(p Params) services.RealtimeService {
	return &Service{
		l:      p.Logger.Named("service.realtime"),
		apiKey: p.Config.GetFoonyConfig().APIKey,
		client: p.Client,
	}
}

func (s *Service) CreateToken(
	req *services.CreateRealtimeTokenRequest,
) (*services.RealtimeToken, error) {
	if req == nil {
		return nil, errortypes.NewBusinessError("realtime token request is required")
	}

	if req.UserID.IsNil() || req.OrganizationID.IsNil() || req.BusinessUnitID.IsNil() {
		return nil, errortypes.NewBusinessError("invalid realtime auth context")
	}

	if s.client == nil {
		return nil, errortypes.NewBusinessError("realtime service is not configured")
	}

	clientID := req.UserID.String()
	token, err := realtime.CreateJWT(s.apiKey, realtime.CreateJWTParams{
		ClientID:   clientID,
		TTL:        defaultTokenTTL,
		Capability: tenantCapability(req.OrganizationID.String(), req.BusinessUnitID.String()),
	})
	if err != nil {
		s.l.Error("failed to mint Foony token", zap.Error(err))
		return nil, fmt.Errorf("mint realtime token: %w", err)
	}

	return &services.RealtimeToken{
		Token:     token,
		ClientID:  clientID,
		ExpiresAt: time.Now().Add(defaultTokenTTL).UnixMilli(),
	}, nil
}

func tenantCapability(orgID, buID string) realtime.Capability {
	return realtime.Capability{
		fmt.Sprintf("tenant:%s:%s:*", orgID, buID): {"subscribe", "presence", "history"},
	}
}

func (s *Service) PublishResourceInvalidation(
	ctx context.Context,
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

	if s.client == nil {
		return errortypes.NewBusinessError("realtime service is not configured")
	}

	eventType := strings.TrimSpace(req.EventType)
	if eventType == "" {
		eventType = fmt.Sprintf("%s.%s", req.Resource, req.Action)
	}

	channelName := tenantDataEventsChannelName(
		req.OrganizationID.String(),
		req.BusinessUnitID.String(),
	)
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

	if !req.RecordID.IsNil() {
		event.EntityID = req.RecordID.String()
		event.RecordID = req.RecordID.String()
	}

	if !req.ActorUserID.IsNil() {
		event.ActorUserID = req.ActorUserID.String()
	}

	if req.ActorType != "" {
		event.ActorType = string(req.ActorType)
	}

	if !req.ActorID.IsNil() {
		event.ActorID = req.ActorID.String()
	}

	if !req.ActorAPIKeyID.IsNil() {
		event.ActorAPIKeyID = req.ActorAPIKeyID.String()
	}

	if _, err := s.client.Channels.Get(channelName).Publish(
		ctx,
		resourceInvalidationEventName,
		event,
	); err != nil {
		s.l.Warn(
			"failed to publish realtime invalidation event",
			zap.Error(err),
			zap.String("channel", channelName),
			zap.Any("req", req),
		)
		return fmt.Errorf("publish realtime invalidation event: %w", err)
	}

	return nil
}

func tenantDataEventsChannelName(orgID, buID string) string {
	return fmt.Sprintf("tenant:%s:%s:data-events", orgID, buID)
}
