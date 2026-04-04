package openaidocumentservice

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type rewriteTransport struct {
	base   http.RoundTripper
	target *url.URL
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = t.target.Scheme
	cloned.URL.Host = t.target.Host

	return t.base.RoundTrip(cloned)
}

func newTestService(t *testing.T, handler http.HandlerFunc) *Service {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg := &config.Config{
		DocumentIntelligence: config.DocumentIntelligenceConfig{
			EnableAI:     true,
			AIMaxRetries: 0,
		},
	}
	metricRegistry, err := metrics.NewRegistry(&config.Config{}, zap.NewNop())
	require.NoError(t, err)

	integrationRepo := mocks.NewMockIntegrationRepository(t)
	integrationRepo.EXPECT().GetByType(mock.Anything, mock.Anything, integration.TypeOpenAI).Return(&integration.Integration{
		ID:             pulid.MustNew("intg_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Type:           integration.TypeOpenAI,
		Enabled:        true,
		Configuration: map[string]any{
			"apiKey": "test-api-key",
		},
	}, nil)

	encryption := encryptionservice.New(encryptionservice.Params{
		Config: &config.Config{},
	})
	integrationSvc := integrationservice.New(integrationservice.Params{
		Logger:       zap.NewNop(),
		Repo:         integrationRepo,
		Encryption:   encryption,
		AuditService: &mocks.NoopAuditService{},
	})

	aiLogRepo := mocks.NewMockAILogRepository(t)
	aiLogRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(entry *ailog.Log) bool {
		return entry != nil && entry.Model != "" && entry.Operation != ""
	})).Return(&ailog.Log{}, nil).Maybe()

	service := New(Params{
		Logger:      zap.NewNop(),
		Config:      cfg,
		Metrics:     metricRegistry,
		Integration: integrationSvc,
		AILogRepo:   aiLogRepo,
	}).(*Service)

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	service.httpClient = &http.Client{
		Transport: rewriteTransport{
			base:   server.Client().Transport,
			target: serverURL,
		},
	}

	return service
}

func TestRouteDocumentBuildsStructuredRequest(t *testing.T) {
	t.Parallel()

	service := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/responses", r.URL.Path)
		require.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		var payload responsesRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "gpt-5-nano-2025-08-07", payload.Model)
		assert.Equal(t, "json_schema", payload.Text.Format.Type)
		assert.Equal(t, string(ailog.OperationDocumentIntelligenceRoute), payload.Text.Format.Name)
		assert.True(t, payload.Text.Format.Strict)
		assert.Len(t, payload.Input, 2)
		assert.Contains(t, payload.Input[1].Content[0].Text, "Rate Confirmation")
		assert.Contains(t, payload.Input[1].Content[0].Text, "Carrier rate confirmation")

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"output_text": `{"shouldExtract":true,"documentKind":"RateConfirmation","confidence":0.93,"signals":["carrier rate confirmation"],"reviewStatus":"Ready","classifierSource":"ai","providerFingerprint":"carrier-template","reason":"Matched rate confirmation headings"}`,
			"usage": map[string]any{
				"input_tokens":  42,
				"output_tokens": 17,
				"total_tokens":  59,
				"output_tokens_details": map[string]any{
					"reasoning_tokens": 3,
				},
			},
			"service_tier": "default",
		}))
	})

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	documentID := pulid.MustNew("doc_")

	result, err := service.RouteDocument(t.Context(), &serviceports.AIRouteRequest{
		TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
		DocumentID: documentID,
		FileName:   "rate-confirmation.pdf",
		Text:       "Rate Confirmation\nCarrier rate confirmation for load 123.",
		Pages: []serviceports.AIDocumentPage{
			{PageNumber: 1, Text: "Carrier rate confirmation"},
		},
	})
	require.NoError(t, err)
	assert.True(t, result.ShouldExtract)
	assert.Equal(t, "RateConfirmation", result.DocumentKind)
	assert.Equal(t, "Ready", result.ReviewStatus)
}

func TestExtractRateConfirmationBuildsExtractionSchemaRequest(t *testing.T) {
	t.Parallel()

	service := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		var payload responsesRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "gpt-5-mini-2025-08-07", payload.Model)
		assert.Equal(t, string(ailog.OperationDocumentIntelligenceExtract), payload.Text.Format.Name)
		assert.Equal(t, "object", payload.Text.Format.Schema["type"])
		assert.Contains(t, payload.Input[1].Content[0].Text, "[Page 1]")
		assert.Contains(t, payload.Input[1].Content[0].Text, "Rate Confirmation")
		assert.Contains(t, payload.Input[1].Content[0].Text, "Canonical field keys:")
		properties := payload.Text.Format.Schema["properties"].(map[string]any)
		fields := properties["fields"].(map[string]any)
		assert.Equal(t, "array", fields["type"])
		assert.EqualValues(t, 18, fields["maxItems"])
		fieldItems := fields["items"].(map[string]any)
		fieldKey := fieldItems["properties"].(map[string]any)["key"].(map[string]any)
		assert.Contains(t, fieldKey["enum"], "rate")
		stopItems := properties["stops"].(map[string]any)
		assert.EqualValues(t, 8, stopItems["maxItems"])

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"output_text": `{"documentKind":"RateConfirmation","overallConfidence":0.88,"reviewStatus":"NeedsReview","missingFields":["customer"],"signals":["pickup","delivery"],"fields":[{"key":"bol","label":"BOL","value":"BOL-123","confidence":0.91,"evidenceExcerpt":"BOL-123","pageNumber":1,"reviewRequired":false,"conflict":false,"source":"page","alternativeValues":[]}],"stops":[],"conflicts":[]}`,
			"usage": map[string]any{
				"input_tokens":  50,
				"output_tokens": 22,
				"total_tokens":  72,
				"output_tokens_details": map[string]any{
					"reasoning_tokens": 4,
				},
			},
			"service_tier": "default",
		}))
	})

	result, err := service.ExtractRateConfirmation(t.Context(), &serviceports.AIExtractRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		DocumentID: pulid.MustNew("doc_"),
		FileName:   "rate-confirmation.pdf",
		Text:       "Rate Confirmation text",
		Pages: []serviceports.AIDocumentPage{
			{PageNumber: 1, Text: "Rate Confirmation page 1"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "RateConfirmation", result.DocumentKind)
	assert.Equal(t, 1, len(result.Fields))
	assert.Equal(t, "NeedsReview", result.ReviewStatus)
}

func TestSubmitRateConfirmationBackgroundExtraction(t *testing.T) {
	t.Parallel()

	service := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)

		var payload responsesRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.True(t, payload.Background)
		assert.True(t, payload.Store)
		assert.Equal(t, "gpt-5-mini-2025-08-07", payload.Model)
		assert.Equal(t, 5000, payload.MaxOutputTokens)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp_123",
			"status": "queued",
			"model":  "gpt-5-mini-2025-08-07",
		}))
	})

	result, err := service.SubmitRateConfirmationBackgroundExtraction(t.Context(), &serviceports.AIExtractRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		DocumentID: pulid.MustNew("doc_"),
		FileName:   "rate-confirmation.pdf",
		Text:       "Rate Confirmation text",
		Pages: []serviceports.AIDocumentPage{
			{PageNumber: 1, Text: "Rate Confirmation page 1"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "resp_123", result.ResponseID)
	assert.Equal(t, "queued", result.Status)
}

func TestPollRateConfirmationBackgroundExtractionCompleted(t *testing.T) {
	t.Parallel()

	service := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v1/responses/resp_123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":          "resp_123",
			"status":      "completed",
			"model":       "gpt-5-mini-2025-08-07",
			"output_text": `{"documentKind":"RateConfirmation","overallConfidence":0.88,"reviewStatus":"NeedsReview","missingFields":["customer"],"signals":["pickup","delivery"],"fields":[{"key":"bol","label":"BOL","value":"BOL-123","confidence":0.91,"evidenceExcerpt":"BOL-123","pageNumber":1,"reviewRequired":false,"conflict":false,"source":"page","alternativeValues":[]}],"stops":[],"conflicts":[]}`,
		}))
	})

	result, err := service.PollRateConfirmationBackgroundExtraction(t.Context(), &serviceports.AIBackgroundExtractPollRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		DocumentID: pulid.MustNew("doc_"),
		ResponseID: "resp_123",
	})
	require.NoError(t, err)
	assert.Equal(t, serviceports.AIBackgroundExtractionStatusCompleted, result.Status)
	require.NotNil(t, result.ExtractResult)
	assert.Equal(t, "BOL-123", result.ExtractResult.Fields["bol"].Value)
}

func TestPollRateConfirmationBackgroundExtractionIncompleteIncludesReason(t *testing.T) {
	t.Parallel()

	service := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v1/responses/resp_123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":     "resp_123",
			"status": "incomplete",
			"model":  "gpt-5-mini-2025-08-07",
			"incomplete_details": map[string]any{
				"reason": "max_output_tokens",
			},
		}))
	})

	result, err := service.PollRateConfirmationBackgroundExtraction(t.Context(), &serviceports.AIBackgroundExtractPollRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		DocumentID: pulid.MustNew("doc_"),
		ResponseID: "resp_123",
	})
	require.NoError(t, err)
	assert.Equal(t, serviceports.AIBackgroundExtractionStatusFailed, result.Status)
	assert.Equal(t, "max_output_tokens", result.FailureCode)
	assert.Equal(t, "AI background extraction ended incomplete: max_output_tokens", result.FailureMessage)
}
