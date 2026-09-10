package controlplaneprovisioninghandler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testMaxProvisioningBodyBytes = 4096

type fakeProvisioningService struct {
	called bool
	req    *services.TenantProvisioningRequest
}

func (s *fakeProvisioningService) ProvisionTenant(
	_ context.Context,
	req *services.TenantProvisioningRequest,
) (*services.TenantProvisioningResult, error) {
	s.called = true
	s.req = req
	return &services.TenantProvisioningResult{
		Accepted:              true,
		BusinessUnitID:        req.Customer.ID,
		OrganizationID:        req.Workspace.ID,
		BusinessUnitsUpserted: 1,
		OrganizationsUpserted: 1,
		ReceivedAt:            123,
	}, nil
}

func TestHandler_ProvisionTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("accepts signed provisioning requests", func(t *testing.T) {
		service := &fakeProvisioningService{}
		handler := newTestHandler(service)
		router := gin.New()
		handler.RegisterPublicRoutes(router.Group("/api/v1"))
		body := mustProvisioningBody(t)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/control-plane/tenants/provision",
			bytes.NewReader(body),
		)
		signTestRequest(req, "cp_secret", body, time.Unix(100, 0))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusAccepted, rec.Code)
		require.True(t, service.called)
		require.Equal(t, "inst_test", service.req.InstanceID)
	})

	t.Run("rejects unsigned provisioning requests", func(t *testing.T) {
		service := &fakeProvisioningService{}
		handler := newTestHandler(service)
		router := gin.New()
		handler.RegisterPublicRoutes(router.Group("/api/v1"))

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/control-plane/tenants/provision",
			bytes.NewReader(mustProvisioningBody(t)),
		)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusForbidden, rec.Code)
		require.False(t, service.called)
	})
}

func TestHandler_ProvisionTenantBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("rejects a declared oversized body without reading it", func(t *testing.T) {
		service := &fakeProvisioningService{}
		handler := newTestHandler(service)
		router := gin.New()
		handler.RegisterPublicRoutes(router.Group("/api/v1"))

		payload := bytes.Repeat([]byte("a"), testMaxProvisioningBodyBytes*16)
		source := &countingReader{r: bytes.NewReader(payload)}
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/control-plane/tenants/provision",
			source,
		)
		req.ContentLength = int64(len(payload))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		require.False(t, service.called)
		require.Zero(t, source.read)
	})

	t.Run("rejects an oversized chunked body with no content length", func(t *testing.T) {
		service := &fakeProvisioningService{}
		handler := newTestHandler(service)
		router := gin.New()
		handler.RegisterPublicRoutes(router.Group("/api/v1"))

		source := &countingReader{
			r: bytes.NewReader(bytes.Repeat([]byte("a"), testMaxProvisioningBodyBytes*16)),
		}
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/control-plane/tenants/provision",
			source,
		)
		req.ContentLength = -1
		req.TransferEncoding = []string{"chunked"}
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		require.False(t, service.called)
		require.LessOrEqual(t, source.read, int64(testMaxProvisioningBodyBytes)+1)
	})

	t.Run("rejects a signed oversized body", func(t *testing.T) {
		service := &fakeProvisioningService{}
		handler := newTestHandler(service)
		router := gin.New()
		handler.RegisterPublicRoutes(router.Group("/api/v1"))
		body := bytes.Repeat([]byte("a"), testMaxProvisioningBodyBytes+1)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/control-plane/tenants/provision",
			bytes.NewReader(body),
		)
		signTestRequest(req, "cp_secret", body, time.Unix(100, 0))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
		require.False(t, service.called)
	})

	t.Run("accepts a signed body at the configured limit", func(t *testing.T) {
		service := &fakeProvisioningService{}
		handler := newTestHandler(service)
		router := gin.New()
		handler.RegisterPublicRoutes(router.Group("/api/v1"))
		body := paddedProvisioningBody(t, testMaxProvisioningBodyBytes)
		require.Len(t, body, testMaxProvisioningBodyBytes)

		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/control-plane/tenants/provision",
			bytes.NewReader(body),
		)
		signTestRequest(req, "cp_secret", body, time.Unix(100, 0))
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusAccepted, rec.Code)
		require.True(t, service.called)
	})
}

type countingReader struct {
	r    io.Reader
	read int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += int64(n)
	return n, err
}

func newTestHandler(service services.TenantProvisioningService) *Handler {
	cfg := &config.Config{
		App: config.AppConfig{
			Env: "test",
		},
		Platform: config.PlatformConfig{
			InstanceID: "inst_test",
			ControlPlane: config.PlatformControlPlaneConfig{
				APIKey:                   "cp_secret",
				MaxProvisioningBodyBytes: testMaxProvisioningBodyBytes,
			},
		},
	}

	handler := New(Params{
		Config:  cfg,
		Service: service,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
		}),
	})
	handler.now = func() time.Time { return time.Unix(100, 0) }
	return handler
}

func mustProvisioningBody(t *testing.T) []byte {
	t.Helper()

	return mustProvisioningBodyWithPadding(t, 0)
}

func paddedProvisioningBody(t *testing.T, size int) []byte {
	t.Helper()

	base := mustProvisioningBodyWithPadding(t, 0)
	require.Less(t, len(base), size)

	return mustProvisioningBodyWithPadding(t, size-len(base))
}

func mustProvisioningBodyWithPadding(t *testing.T, padding int) []byte {
	t.Helper()

	customerID := pulid.MustNew("bu_")
	body, err := sonic.Marshal(services.TenantProvisioningRequest{
		InstanceID: "inst_test",
		Customer: services.TenantProvisioningCustomer{
			ID:   customerID,
			Name: "Acme Logistics",
			Code: "ACME",
		},
		Workspace: services.TenantProvisioningWorkspace{
			ID:             pulid.MustNew("org_"),
			BusinessUnitID: customerID,
			Name:           "Acme Northeast" + strings.Repeat("a", padding),
			State:          "NY",
			AddressLine1:   "100 Main Street",
			City:           "Albany",
			PostalCode:     "12207",
			Timezone:       "America/New_York",
			BucketName:     "acme-northeast",
			TaxID:          "12-3456789",
			ScacCode:       "ACME",
			DOTNumber:      "123456",
			LoginSlug:      "acme",
		},
		SentAt: 100,
	})
	require.NoError(t, err)
	return body
}

func signTestRequest(req *http.Request, secret string, body []byte, now time.Time) {
	timestamp := strconv.FormatInt(now.Unix(), 10)
	bodyHash := bodySHA256(body)

	req.Header.Set(headerInstanceID, "inst_test")
	req.Header.Set(headerTimestamp, timestamp)
	req.Header.Set(headerBodySHA256, bodyHash)
	req.Header.Set(
		headerSignature,
		computeSignature(secret, req.Method, req.URL.Path, bodyHash, timestamp),
	)
	req.Header.Set("Content-Type", "application/json")
}
