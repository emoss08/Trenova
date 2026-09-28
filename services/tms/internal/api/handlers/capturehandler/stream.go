package capturehandler

import (
	"bytes"
	"context"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	// EventCaptureRequest tells the device to fetch its open requests.
	EventCaptureRequest = "capture.request"
	// EventCaptureRevoked tells the device it has been revoked. The stream
	// closes right after it.
	EventCaptureRevoked = "capture.revoked"
	lastEventIDHeader   = "Last-Event-ID"
	defaultLifetime     = 30 * time.Minute
)

// captureResources are the only invalidations a device's stream considers.
// Everything else the person's tenant publishes is dropped before it is even
// decoded, which keeps a busy tenant from costing a device anything.
var captureResources = [][]byte{
	[]byte(`"resource":"` + permission.ResourceCaptureBatch.String() + `"`),
	[]byte(`"resource":"` + permission.ResourceCaptureDevice.String() + `"`),
}

type streamedInvalidation struct {
	Resource string                       `json:"resource"`
	Action   string                       `json:"action"`
	Entity   *captureservice.DeviceSignal `json:"entity"`
}

// @Summary Open the device's live stream
// @Description A server-sent event stream carrying ready, heartbeat, reset and close as the web
// @Description app's stream does, plus capture.request when the device has something to do and
// @Description capture.revoked when it has been revoked. The device fetches its requests on
// @Description capture.request, on reset, and whenever it reconnects.
// @ID openCaptureDeviceStream
// @Tags Capture
// @Produce text/event-stream
// @Success 200 {string} string "event stream"
// @Failure 401 {object} helpers.ProblemDetail
// @Failure 429 {object} helpers.ProblemDetail
// @Security BearerAuth
// @Router /capture/device/stream/ [get]
func (h *Handler) stream(c *gin.Context) {
	if h.gateway == nil {
		h.eh.HandleError(c, errortypes.NewBusinessError("Live updates are not available"))

		return
	}

	principal := devicePrincipal(c)
	tenantInfo := principal.TenantInfo()
	ctx := c.Request.Context()

	rt, err := h.gateway.Open(ctx, &services.OpenRealtimeStreamRequest{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		UserID:         tenantInfo.UserID,
		LastEventID:    c.GetHeader(lastEventIDHeader),
	})
	if err != nil {
		if errortypes.IsRateLimitError(err) {
			retryAfter(c)
		}
		h.eh.HandleError(c, err)

		return
	}
	defer rt.Close()

	stream, err := helpers.OpenEventStream(c, helpers.EventStreamOptions{Heartbeat: -1})
	if err != nil {
		h.eh.HandleError(c, errortypes.NewBusinessError("Streaming is not supported"))

		return
	}
	defer stream.Close()

	for _, frame := range rt.Prelude() {
		stream.EmitRaw(frame.ID, frame.Event, frame.Data)
	}

	h.pump(ctx, c.ClientIP(), principal, rt, stream)
}

// pump forwards the frames a device needs until the stream ends. It is
// recycled on a jittered clock like the web app's, so a revoked or expired
// credential is noticed at the next reconnect even if no event says so.
func (h *Handler) pump(
	ctx context.Context,
	clientIP string,
	principal *captureservice.DevicePrincipal,
	rt services.RealtimeStream,
	stream *helpers.EventStream,
) {
	rotate := time.NewTimer(timeutils.Jittered(h.lifetime()))
	defer rotate.Stop()

	deviceID := principal.Device.ID
	frames := rt.Frames()
	for {
		select {
		case <-ctx.Done():
			return
		case <-rotate.C:
			h.emitClose(stream, services.RealtimeCloseRotate)

			return
		case frame, ok := <-frames:
			if !ok {
				h.emitClose(stream, rt.Reason())

				return
			}

			switch frame.Event {
			case services.RealtimeEventHeartbeat:
				h.service.Heartbeat(ctx, principal, clientIP)
				stream.EmitRaw(frame.ID, frame.Event, frame.Data)
			case services.RealtimeEventReset,
				services.RealtimeEventClose,
				services.RealtimeEventReady:
				stream.EmitRaw(frame.ID, frame.Event, frame.Data)
				if frame.Event == services.RealtimeEventClose {
					return
				}
			case services.RealtimeEventInvalidation:
				if h.forward(stream, frame, deviceID) {
					return
				}
			}
		}
	}
}

// forward relays an invalidation addressed to this device, reporting whether
// the stream must now end.
func (h *Handler) forward(
	stream *helpers.EventStream,
	frame services.RealtimeFrame,
	deviceID pulid.ID,
) bool {
	relevant := false
	for _, marker := range captureResources {
		if bytes.Contains(frame.Data, marker) {
			relevant = true

			break
		}
	}
	if !relevant {
		return false
	}

	event := new(streamedInvalidation)
	if err := sonic.Unmarshal(frame.Data, event); err != nil {
		h.l.Debug("could not read a capture invalidation", zap.Error(err))

		return false
	}
	if event.Entity == nil || event.Entity.DeviceID != deviceID {
		return false
	}

	switch event.Resource {
	case permission.ResourceCaptureDevice.String():
		if event.Action == "revoked" {
			if err := stream.Emit(EventCaptureRevoked, event.Entity); err != nil {
				h.l.Debug("could not tell a device it was revoked", zap.Error(err))
			}

			return true
		}
	case permission.ResourceCaptureBatch.String():
		if err := stream.EmitWithID(frame.ID, EventCaptureRequest, event.Entity); err != nil {
			h.l.Debug("could not relay a capture request", zap.Error(err))
		}
	}

	return false
}

func (h *Handler) emitClose(stream *helpers.EventStream, reason string) {
	if err := stream.Emit(
		services.RealtimeEventClose,
		services.RealtimeCloseEvent{Reason: reason},
	); err != nil {
		h.l.Debug("could not send capture stream close notice", zap.Error(err))
	}
}

func (h *Handler) lifetime() time.Duration {
	if h.maxLifetime <= 0 {
		return defaultLifetime
	}

	return time.Duration(h.maxLifetime) * time.Second
}
