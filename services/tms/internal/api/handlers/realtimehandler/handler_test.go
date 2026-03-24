package realtimehandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubRealtimeService struct {
	resp    *servicesport.RealtimeTokenRequest
	err     error
	lastReq *servicesport.CreateRealtimeTokenRequest
}

func (s *stubRealtimeService) CreateTokenRequest(
	req *servicesport.CreateRealtimeTokenRequest,
) (*servicesport.RealtimeTokenRequest, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}

	if s.resp == nil {
		return &servicesport.RealtimeTokenRequest{}, nil
	}

	return s.resp, nil
}

func (s *stubRealtimeService) PublishResourceInvalidation(
	_ context.Context,
	_ *servicesport.PublishResourceInvalidationRequest,
) error {
	return nil
}

func newRealtimeTestHandler(
	t *testing.T,
	svc servicesport.RealtimeService,
) (*Handler, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		App: config.AppConfig{Debug: true},
	}

	eh := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: cfg,
	})

	h := New(Params{
		Service:      svc,
		ErrorHandler: eh,
	})

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		authctx.SetAuthContext(
			c,
			pulid.MustNew("usr_"),
			pulid.MustNew("bu_"),
			pulid.MustNew("org_"),
		)
		c.Next()
	})
	h.RegisterRoutes(api)

	return h, r
}

func TestGetTokenRequest_Success(t *testing.T) {
	t.Parallel()

	svc := &stubRealtimeService{
		resp: &servicesport.RealtimeTokenRequest{KeyName: "ably-key"},
	}
	_, router := newRealtimeTestHandler(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/realtime/token-request/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body servicesport.RealtimeTokenRequest
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, svc.resp.KeyName, body.KeyName)
	assert.NotNil(t, svc.lastReq)
	assert.False(t, svc.lastReq.UserID.IsNil())
	assert.False(t, svc.lastReq.OrganizationID.IsNil())
	assert.False(t, svc.lastReq.BusinessUnitID.IsNil())
}

func TestGetTokenRequest_ServiceError(t *testing.T) {
	t.Parallel()

	svc := &stubRealtimeService{
		err: errors.New("boom"),
	}

	_, router := newRealtimeTestHandler(t, svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/realtime/token-request/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
