package accountingwebhookhandler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingService struct {
	services.AccountingConnectionService
	received *services.ReceiveAccountingWebhookRequest
	err      error
}

func (s *recordingService) ReceiveWebhook(
	_ context.Context,
	req *services.ReceiveAccountingWebhookRequest,
) error {
	s.received = req
	return s.err
}

type headerProvider struct{ services.AccountingProvider }

func (headerProvider) WebhookSignatureHeader() string { return "intuit-signature" }

type registry struct{}

func (registry) For(typ integration.Type) (services.AccountingProvider, bool) {
	if typ == integration.TypeQuickBooksOnline {
		return headerProvider{}, true
	}
	return nil, false
}

func serve(t *testing.T, svc *recordingService, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	New(Params{Service: svc, Connectors: registry{}}).RegisterPublicRoutes(engine.Group(""))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("intuit-signature", "c2lnbmF0dXJl")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestWebhookPassesSignatureAndBodyToTheService(t *testing.T) {
	t.Parallel()

	svc := &recordingService{}
	rec := serve(t, svc, WebhookPath("quickbooks"), []byte(`[]`))
	assert.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, svc.received)
	assert.Equal(t, integration.TypeQuickBooksOnline, svc.received.IntegrationType)
	assert.Equal(t, "c2lnbmF0dXJl", svc.received.Signature)
	assert.Equal(t, []byte(`[]`), svc.received.Body)
}

func TestWebhookRejectsABadSignature(t *testing.T) {
	t.Parallel()

	svc := &recordingService{err: errortypes.NewAuthenticationError("bad signature")}
	rec := serve(t, svc, WebhookPath("quickbooks"), []byte(`[]`))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestWebhookRejectsAnUnreadableBody(t *testing.T) {
	t.Parallel()

	svc := &recordingService{err: errortypes.NewValidationError("body", errortypes.ErrInvalid, "bad")}
	rec := serve(t, svc, WebhookPath("quickbooks"), []byte(`not json`))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestWebhookRejectsUnknownProvidersAndOversizedBodies(t *testing.T) {
	t.Parallel()

	svc := &recordingService{}
	assert.Equal(t, http.StatusNotFound, serve(t, svc, WebhookPath("xero"), []byte(`[]`)).Code)
	assert.Nil(t, svc.received)

	big := bytes.Repeat([]byte("a"), maxWebhookBodyBytes+1)
	assert.Equal(t, http.StatusRequestEntityTooLarge, serve(t, svc, WebhookPath("quickbooks"), big).Code)
	assert.Nil(t, svc.received)
}
