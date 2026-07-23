package webpushservice

import (
	"context"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const pushTTLSeconds = 43200

type Params struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
	Repo   repositories.PushSubscriptionRepository
}

type Service struct {
	l    *zap.Logger
	cfg  config.PushConfig
	repo repositories.PushSubscriptionRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.webpush"),
		cfg:  p.Config.Push,
		repo: p.Repo,
	}
}

func (s *Service) Enabled() bool { return s.cfg.Enabled() }

func (s *Service) PublicKey() string { return s.cfg.VAPIDPublicKey }

func (s *Service) Subscribe(
	ctx context.Context,
	req *repositories.SavePushSubscriptionRequest,
) (*notification.PushSubscription, error) {
	return s.repo.Save(ctx, req)
}

func (s *Service) Unsubscribe(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	endpoint string,
) error {
	return s.repo.DeleteByEndpoint(ctx, tenantInfo.UserID, endpoint)
}

type PushPayload struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	Link    string `json:"link,omitempty"`
	EventID string `json:"eventId,omitempty"`
}

// SendToUser delivers a push payload to every registered subscription for the
// user. Delivery is best-effort: individual failures are logged and dead
// subscriptions (404/410) are pruned.
func (s *Service) SendToUser(ctx context.Context, userID pulid.ID, payload *PushPayload) {
	if !s.Enabled() || userID.IsNil() || payload == nil {
		return
	}

	subscriptions, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		s.l.Warn("failed to list push subscriptions", zap.Error(err))
		return
	}
	if len(subscriptions) == 0 {
		return
	}

	body, err := sonic.Marshal(payload)
	if err != nil {
		s.l.Warn("failed to marshal push payload", zap.Error(err))
		return
	}

	options := &webpush.Options{
		Subscriber:      s.cfg.Subject,
		VAPIDPublicKey:  s.cfg.VAPIDPublicKey,
		VAPIDPrivateKey: s.cfg.VAPIDPrivateKey,
		TTL:             pushTTLSeconds,
		Urgency:         webpush.UrgencyNormal,
	}

	for _, subscription := range subscriptions {
		s.sendOne(ctx, subscription, body, options)
	}
}

func (s *Service) sendOne(
	ctx context.Context,
	subscription *notification.PushSubscription,
	body []byte,
	options *webpush.Options,
) {
	target := &webpush.Subscription{
		Endpoint: subscription.Endpoint,
		Keys: webpush.Keys{
			P256dh: subscription.P256dh,
			Auth:   subscription.Auth,
		},
	}

	resp, err := webpush.SendNotificationWithContext(ctx, body, target, options)
	if err != nil {
		s.l.Warn("failed to send web push",
			zap.String("subscriptionId", subscription.ID.String()),
			zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		if delErr := s.repo.DeleteByID(ctx, subscription.ID); delErr != nil {
			s.l.Warn("failed to prune dead push subscription", zap.Error(delErr))
		}
		return
	}
	if resp.StatusCode >= 400 {
		s.l.Warn("web push endpoint rejected notification",
			zap.Int("status", resp.StatusCode),
			zap.String("subscriptionId", subscription.ID.String()))
	}
}
