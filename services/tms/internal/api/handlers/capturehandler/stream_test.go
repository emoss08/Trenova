package capturehandler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturedevices"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func invalidationFrame(
	t *testing.T,
	resource, action string,
	signal capturedevices.DeviceSignal,
) realtimeports.RealtimeFrame {
	t.Helper()

	data, err := sonic.Marshal(realtimeports.ResourceInvalidationEvent{
		Resource: resource,
		Action:   action,
		Entity:   signal,
	})
	require.NoError(t, err)

	return realtimeports.RealtimeFrame{
		ID:    "1-0",
		Event: realtimeports.RealtimeEventInvalidation,
		Data:  data,
	}
}

func openTestStream(t *testing.T) (*helpers.EventStream, *httptest.ResponseRecorder) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil).WithContext(t.Context())

	stream, err := helpers.OpenEventStream(c, helpers.EventStreamOptions{Heartbeat: -1})
	require.NoError(t, err)

	return stream, recorder
}

func TestStreamRelaysOnlyThisDevicesRequests(t *testing.T) {
	t.Parallel()

	h := &Handler{l: zap.NewNop()}
	stream, recorder := openTestStream(t)
	defer stream.Close()

	mine, theirs := pulid.MustNew("cdev_"), pulid.MustNew("cdev_")
	requestID := pulid.MustNew("creq_")

	assert.False(t, h.forward(stream, invalidationFrame(
		t,
		permission.ResourceCaptureBatch.String(),
		"request.created",
		capturedevices.DeviceSignal{DeviceID: theirs, RequestID: requestID},
	), mine))
	assert.False(t, h.forward(stream, invalidationFrame(t, permission.ResourceShipment.String(),
		"updated", capturedevices.DeviceSignal{DeviceID: mine}), mine))
	assert.NotContains(t, recorder.Body.String(), EventCaptureRequest,
		"another device's request and unrelated resources are dropped")

	assert.False(t, h.forward(stream, invalidationFrame(
		t,
		permission.ResourceCaptureBatch.String(),
		"request.created",
		capturedevices.DeviceSignal{DeviceID: mine, RequestID: requestID},
	), mine))
	assert.Contains(t, recorder.Body.String(), "event: "+EventCaptureRequest)
	assert.Contains(t, recorder.Body.String(), requestID.String())
}

func TestStreamEndsWhenTheDeviceIsRevoked(t *testing.T) {
	t.Parallel()

	h := &Handler{l: zap.NewNop()}
	stream, recorder := openTestStream(t)
	defer stream.Close()

	device := pulid.MustNew("cdev_")
	done := h.forward(stream, invalidationFrame(t, permission.ResourceCaptureDevice.String(),
		"revoked", capturedevices.DeviceSignal{DeviceID: device}), device)

	assert.True(t, done)
	assert.Contains(t, recorder.Body.String(), "event: "+EventCaptureRevoked)
}

func TestFailMapsDeviceErrors(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	h := &Handler{l: zap.NewNop()}

	cases := []struct {
		err    error
		status int
		body   string
	}{
		{
			&capturedevices.PairingError{Code: capturedevices.PairingSlowDown},
			http.StatusBadRequest,
			`"error":"slow_down"`,
		},
		{
			&capturedevices.OutdatedAgentError{Current: "1.0.0", Minimum: "1.2.0"},
			http.StatusUpgradeRequired, `"minimumVersion":"1.2.0"`,
		},
		{
			&capturedevices.OutdatedAgentError{
				Current:    "1.0.0",
				Minimum:    "1.2.0",
				AutoUpdate: true,
			},
			http.StatusUpgradeRequired, `"autoUpdate":true`,
		},
	}

	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		h.fail(c, tc.err)

		assert.Equal(t, tc.status, recorder.Code)
		assert.True(t, strings.Contains(recorder.Body.String(), tc.body), recorder.Body.String())
	}
}

func TestRequireDeviceRefusesAnythingButABearerToken(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	h := &Handler{l: zap.NewNop(), eh: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    zap.NewNop(),
		AppConfig: &appconfig.Config{},
	})}

	for _, header := range []string{"", "Basic abc", "Bearer", "bearer "} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		if header != "" {
			c.Request.Header.Set("Authorization", header)
		}

		h.RequireDevice()(c)

		assert.True(t, c.IsAborted(), header)
		assert.Equal(t, http.StatusUnauthorized, recorder.Code, header)
	}
}
