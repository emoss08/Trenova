//go:build integration

package realtimehandler

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/realtimeservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/realtimebroker"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

type sseEvent struct {
	id    string
	event string
	data  string
}

// sseReader reads frames off a live response the way a browser would.
type sseReader struct {
	events chan sseEvent
}

func readSSE(t *testing.T, body *bufio.Reader) *sseReader {
	t.Helper()
	r := &sseReader{events: make(chan sseEvent, 64)}
	go func() {
		defer close(r.events)
		var current sseEvent
		for {
			line, err := body.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSuffix(line, "\n")
			switch {
			case line == "":
				if current.event != "" {
					r.events <- current
				}
				current = sseEvent{}
			case strings.HasPrefix(line, "id: "):
				current.id = strings.TrimPrefix(line, "id: ")
			case strings.HasPrefix(line, "event: "):
				current.event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				current.data = strings.TrimPrefix(line, "data: ")
			}
		}
	}()
	return r
}

func (r *sseReader) next(t *testing.T, event string) sseEvent {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case e, ok := <-r.events:
			require.True(t, ok, "stream ended while waiting for %s", event)
			if e.event == event {
				return e
			}
		case <-deadline:
			t.Fatalf("timed out waiting for %s", event)
		}
	}
}

func TestStreamEndToEnd(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	cfg := &config.Config{
		App:      config.AppConfig{Debug: true},
		Realtime: config.RealtimeConfig{ShardCount: 4, HeartbeatInterval: time.Hour},
	}
	logger := zap.NewNop()

	lc := fxtest.NewLifecycle(t)
	publisher := realtimebroker.NewPublisher(realtimebroker.PublisherParams{
		Client: client, Config: cfg, Logger: logger, LC: lc,
	})
	broker := realtimebroker.NewBroker(realtimebroker.BrokerParams{
		Publisher: publisher, Client: client, Config: cfg, Logger: logger, LC: lc,
	})
	lc.RequireStart()
	t.Cleanup(lc.RequireStop)

	users := mocks.NewMockUserRepository(t)
	users.On("GetByID", mock.Anything, mock.Anything).
		Return(&tenant.User{ID: testUserID, Name: "Dana Dispatcher"}, nil)

	gateway := realtimeservice.NewGateway(realtimeservice.GatewayParams{
		Logger: logger, Config: cfg, Broker: broker, UserRepo: users,
	})
	publish := realtimeservice.New(realtimeservice.Params{
		Logger: logger, Config: cfg, Publisher: publisher,
	})

	engine := mocks.NewMockPermissionEngine(t)
	engine.On("Check", mock.Anything, mock.Anything).
		Return(&servicesport.PermissionCheckResult{Allowed: true}, nil).Maybe()
	eh := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: logger, Config: cfg})
	h := New(Params{
		Gateway:      gateway,
		ErrorHandler: eh,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{PermissionEngine: engine, ErrorHandler: eh},
		),
		Config: cfg,
		Logger: logger,
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		authctx.SetAuthContext(c, testUserID, testBUID, testOrgID)
		c.Next()
	})
	h.RegisterRoutes(api)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(
		ctx, http.MethodGet, server.URL+"/api/v1/realtime/stream/?presence=users", nil,
	)
	require.NoError(t, err)
	resp, err := server.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	stream := readSSE(t, bufio.NewReader(resp.Body))

	var ready servicesport.RealtimeReadyEvent
	require.NoError(t, sonic.UnmarshalString(stream.next(t, "ready").data, &ready))
	require.NotEmpty(t, ready.ConnectionID)

	var snapshot servicesport.RealtimePresenceSnapshot
	require.NoError(t, sonic.UnmarshalString(stream.next(t, "presence.snapshot").data, &snapshot))
	require.Len(t, snapshot.Members, 1)
	assert.Equal(t, "Dana Dispatcher", snapshot.Members[0].Name)

	t.Run("an invalidation arrives with a resumable id", func(t *testing.T) {
		require.NoError(t, publish.PublishResourceInvalidation(t.Context(),
			&servicesport.PublishResourceInvalidationRequest{
				OrganizationID: testOrgID,
				BusinessUnitID: testBUID,
				Resource:       "shipments",
				Action:         "updated",
				RecordID:       pulid.MustNew("shp_"),
			}))

		event := stream.next(t, "resource.invalidation")
		assert.NotEmpty(t, event.id)
		assert.Contains(t, event.data, `"resource":"shipments"`)
	})

	t.Run("joining a thread and typing in it reaches the stream", func(t *testing.T) {
		shipmentID := pulid.MustNew("shp_").String()
		body, _ := sonic.MarshalString(map[string]any{"connectionId": ready.ConnectionID})
		joinResp, err := server.Client().Post(
			server.URL+"/api/v1/shipments/"+shipmentID+"/comments/presence/",
			"application/json",
			bytes.NewBufferString(body),
		)
		require.NoError(t, err)
		_ = joinResp.Body.Close()
		require.Equal(t, http.StatusOK, joinResp.StatusCode)

		body, _ = sonic.MarshalString(map[string]any{"connectionId": ready.ConnectionID})
		typingResp, err := server.Client().Post(
			server.URL+"/api/v1/shipments/"+shipmentID+"/comments/typing/",
			"application/json",
			bytes.NewBufferString(body),
		)
		require.NoError(t, err)
		_ = typingResp.Body.Close()
		require.Equal(t, http.StatusNoContent, typingResp.StatusCode)

		var typing servicesport.RealtimeTypingEvent
		require.NoError(t, sonic.UnmarshalString(stream.next(t, "typing").data, &typing))
		assert.Equal(t, "shipment-comments:"+shipmentID, typing.Scope)
		assert.Equal(t, testUserID.String(), typing.UserID)
		assert.Equal(t, "Dana Dispatcher", typing.Name)
	})

	// The regression this replaced: every publish was a blocking HTTPS round
	// trip to a third party, and a bulk billing transfer made several per
	// shipment. A publish now only queues.
	t.Run("publishing does not wait on the network", func(t *testing.T) {
		// Another tenant's burst, so this reader's buffer is not what is measured.
		otherOrg, otherBU := pulid.MustNew("org_"), pulid.MustNew("bu_")
		const n = 2_000
		started := time.Now()
		for range n {
			require.NoError(t, publish.PublishResourceInvalidation(t.Context(),
				&servicesport.PublishResourceInvalidationRequest{
					OrganizationID: otherOrg,
					BusinessUnitID: otherBU,
					Resource:       "billing_queue",
					Action:         "updated",
					RecordID:       pulid.MustNew("bqi_"),
				}))
		}
		elapsed := time.Since(started)
		perPublish := elapsed / n
		t.Logf("%d publishes queued in %s (%s each)", n, elapsed, perPublish)
		// A round trip to the old vendor measured 570-880ms. Even under the
		// race detector a queued publish is two orders of magnitude below that.
		assert.Less(t, perPublish, 5*time.Millisecond)
	})

	t.Run("draining ends the stream with a shutdown notice", func(t *testing.T) {
		gateway.Drain()
		var closing servicesport.RealtimeCloseEvent
		require.NoError(t, sonic.UnmarshalString(stream.next(t, "close").data, &closing))
		assert.Equal(t, servicesport.RealtimeCloseShutdown, closing.Reason)
	})
}
