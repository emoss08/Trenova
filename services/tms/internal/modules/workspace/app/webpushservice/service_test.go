package webpushservice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/pushconfig"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakePushRepository struct {
	mu            sync.Mutex
	saved         []*notificationports.SavePushSubscriptionRequest
	subscriptions []*notification.PushSubscription
	deleted       []pulid.ID
}

func (r *fakePushRepository) Save(
	_ context.Context,
	req *notificationports.SavePushSubscriptionRequest,
) (*notification.PushSubscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, req)
	return &notification.PushSubscription{Endpoint: req.Endpoint}, nil
}

func (r *fakePushRepository) ListByUser(
	context.Context,
	pulid.ID,
) ([]*notification.PushSubscription, error) {
	return r.subscriptions, nil
}

func (r *fakePushRepository) DeleteByEndpoint(context.Context, pulid.ID, string) error {
	return nil
}

func (r *fakePushRepository) DeleteByID(_ context.Context, id pulid.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deleted = append(r.deleted, id)
	return nil
}

func newTestService(repo *fakePushRepository, push pushconfig.Config) *Service {
	return New(Params{
		Logger:     zap.NewNop(),
		PushConfig: &push,
		Repo:       repo,
	})
}

func subscribeRequest(endpoint string) *notificationports.SavePushSubscriptionRequest {
	return &notificationports.SavePushSubscriptionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		Endpoint: endpoint,
		P256dh:   "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM",
		Auth:     "tBHItJI5svbpez7KI4CCXg",
	}
}

func TestSubscribeAcceptsKnownPushServices(t *testing.T) {
	t.Parallel()

	repo := &fakePushRepository{}
	service := newTestService(repo, pushconfig.Config{})
	for _, endpoint := range []string{
		"https://fcm.googleapis.com/fcm/send/abc:def",
		"https://updates.push.services.mozilla.com/wpush/v2/gAAAA",
		"https://eu-central-1.push.services.mozilla.com/wpush/v2/gAAAA",
		"https://wns2-by3p.notify.windows.com/w/?token=abc",
		"https://web.push.apple.com/QGZx",
		"https://api.push.apple.com/3/device/abc",
		"https://fcm.googleapis.com:443/fcm/send/abc",
	} {
		_, err := service.Subscribe(t.Context(), subscribeRequest(endpoint))
		require.NoError(t, err, endpoint)
	}
	require.Len(t, repo.saved, 7)
}

func TestSubscribeRefusesEndpointsOutsideTheAllowList(t *testing.T) {
	t.Parallel()

	repo := &fakePushRepository{}
	service := newTestService(repo, pushconfig.Config{})
	for _, endpoint := range []string{
		"http://fcm.googleapis.com/fcm/send/abc",
		"https://169.254.169.254/latest/meta-data/",
		"https://127.0.0.1/push",
		"https://localhost/push",
		"https://internal.example/push",
		"https://fcm.googleapis.com.evil.example/fcm/send/abc",
		"https://push.apple.com/abc",
		"https://user:pass@fcm.googleapis.com/fcm/send/abc",
		"https://fcm.googleapis.com:8443/fcm/send/abc",
		"file:///etc/passwd",
		"not a url",
	} {
		_, err := service.Subscribe(t.Context(), subscribeRequest(endpoint))
		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr, endpoint)
		require.True(t, multiErr.HasErrors(), endpoint)
	}
	require.Empty(t, repo.saved)
}

func TestSubscribeHonoursAConfiguredAllowList(t *testing.T) {
	t.Parallel()

	repo := &fakePushRepository{}
	service := newTestService(repo, pushconfig.Config{
		AllowedEndpointHosts: []string{"push.internal.example"},
	})

	_, err := service.Subscribe(
		t.Context(),
		subscribeRequest("https://push.internal.example/abc"),
	)
	require.NoError(t, err)

	_, err = service.Subscribe(
		t.Context(),
		subscribeRequest("https://fcm.googleapis.com/fcm/send/abc"),
	)
	require.Error(t, err)
}

func TestSubscribeTruncatesALongUserAgent(t *testing.T) {
	t.Parallel()

	repo := &fakePushRepository{}
	service := newTestService(repo, pushconfig.Config{})
	req := subscribeRequest("https://fcm.googleapis.com/fcm/send/abc")
	req.UserAgent = string(make([]byte, 400))

	_, err := service.Subscribe(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, repo.saved[0].UserAgent, maxUserAgentLength)
}

func TestSendToUserNeverPostsToADisallowedEndpoint(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)

	stored := &notification.PushSubscription{
		ID:       pulid.MustNew("ups_"),
		Endpoint: server.URL + "/push",
		P256dh:   "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM",
		Auth:     "tBHItJI5svbpez7KI4CCXg",
	}
	repo := &fakePushRepository{subscriptions: []*notification.PushSubscription{stored}}
	service := newTestService(repo, pushconfig.Config{
		VAPIDPublicKey:  "BEl62iUYgUivxIkv69yViEuiBIa-Ib9-SkvMeAtA3LFgDzkrxZJjSgSnfckjBJuBkr3qBUYIHBQFLXYp5Nksh8U",
		VAPIDPrivateKey: "UUxI4O8-FbRouAevSmBQ6o18hgE4nSG3qwvJTfKc-ls",
		Subject:         "mailto:admin@example.com",
	})

	service.SendToUser(t.Context(), pulid.MustNew("usr_"), &PushPayload{Title: "Load"})

	require.Zero(t, hits.Load())
	require.Equal(t, []pulid.ID{stored.ID}, repo.deleted)
}

func TestSendToUserUsesTheGuardedClient(t *testing.T) {
	t.Parallel()

	service := newTestService(&fakePushRepository{}, pushconfig.Config{})
	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"https://127.0.0.1:1/push",
		http.NoBody,
	)
	require.NoError(t, err)
	_, err = service.client.Do(request)
	require.Error(t, err)
	require.Contains(t, err.Error(), "disallowed network")
}
