package controlplaneprovisioninghandler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func newTestHandler(service services.TenantProvisioningService) *Handler {
	cfg := &config.Config{
		App: config.AppConfig{
			Env: "test",
		},
		Platform: config.PlatformConfig{
			InstanceID: "inst_test",
			ControlPlane: config.PlatformControlPlaneConfig{
				APIKey: "cp_secret",
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
			Name:           "Acme Northeast",
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
