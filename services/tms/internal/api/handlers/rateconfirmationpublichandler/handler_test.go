package rateconfirmationpublichandler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenantlimitsconfig"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/httpmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateconfirmationservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateconfirmationports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type tokenRepo struct {
	rateconfirmationports.RateConfirmationRepository

	token  *rateconfirmation.RateConfirmationToken
	entity *rateconfirmation.RateConfirmation
}

func (f *tokenRepo) GetTokenByHash(
	_ context.Context,
	tokenHash string,
) (*rateconfirmation.RateConfirmationToken, error) {
	if f.token != nil && f.token.TokenHash == tokenHash {
		return f.token, nil
	}
	return nil, nil //nolint:nilnil // mirrors the production contract
}

func (f *tokenRepo) GetByID(
	_ context.Context,
	req *rateconfirmationports.GetRateConfirmationByIDRequest,
) (*rateconfirmation.RateConfirmation, error) {
	if f.entity != nil && f.entity.ID == req.RateConfirmationID {
		return f.entity, nil
	}
	return nil, errortypes.NewNotFoundError("Rate confirmation not found")
}

func (f *tokenRepo) MarkTokenUsed(
	_ context.Context,
	_ *rateconfirmationports.MarkRateConfirmationTokenUsedRequest,
) (bool, error) {
	return true, nil
}

func newTestHandler(repo *tokenRepo) *Handler {
	return newTestHandlerWith(repo, plantest.AllFeatures())
}

func newTestHandlerWith(
	repo rateconfirmationports.RateConfirmationRepository,
	features platformcatalogports.FeatureGate,
) *Handler {
	service := rateconfirmationservice.New(rateconfirmationservice.Params{
		Logger:   zap.NewNop(),
		DB:       dbtest.NopConnection{},
		Repo:     repo,
		Features: features,
	})
	cfg := &config.Config{
		Security: securityconfig.Config{
			RateLimit: securityconfig.RateLimitConfig{
				Enabled: true,
				PublicToken: securityconfig.RateLimitScopeConfig{
					RequestsPerMinute: 10,
					BurstSize:         throttleBurst,
				},
			},
		},
	}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger:    zap.NewNop(),
		AppConfig: &cfg.App,
	})
	return New(Params{
		Service:      service,
		ErrorHandler: errorHandler,
		RateLimiter: middleware.NewRateLimiter(middleware.RateLimiterParams{
			SecurityConfig: &cfg.Security,
			Store:          ratelimit.NewMemoryStore(ratelimit.MemoryStoreOptions{}),
			Managed:        &managedconfig.Config{},
			TenantLimits:   &tenantlimitsconfig.Config{},
			ErrorHandler:   errorHandler,
			Logger:         zap.NewNop(),
			Metrics: &metrics.Registry{
				RateLimit: httpmetrics.NewRateLimit(nil, zap.NewNop(), false),
			},
		}),
	})
}

const throttleBurst = 5

func newTestRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler.RegisterPublicRoutes(router.Group("/api/v1"))
	return router
}

func signedRepo() (*tokenRepo, string) {
	raw := "raw-test-token"
	entity := &rateconfirmation.RateConfirmation{
		ID:                  pulid.MustNew("ratecon_"),
		OrganizationID:      pulid.MustNew("org_"),
		BusinessUnitID:      pulid.MustNew("bu_"),
		CarrierAssignmentID: pulid.MustNew("cass_"),
		Status:              rateconfirmation.StatusSent,
		PayloadSnapshot: map[string]any{
			"CarrierName":       "Eastline Transport",
			"ShipmentProNumber": "PRO-1001",
			"TotalCost":         "1850.00",
		},
	}
	return &tokenRepo{
		token: &rateconfirmation.RateConfirmationToken{
			ID:                 pulid.MustNew("rct_"),
			OrganizationID:     entity.OrganizationID,
			BusinessUnitID:     entity.BusinessUnitID,
			RateConfirmationID: entity.ID,
			TokenHash:          tokenutils.Hash(raw),
			ExpiresAt:          timeutils.NowUnix() + 3600,
		},
		entity: entity,
	}, raw
}

func TestPreviewServesTheFrozenSnapshot(t *testing.T) {
	repo, raw := signedRepo()
	router := newTestRouter(newTestHandler(repo))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate-confirmation-links/"+raw+"/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Eastline Transport")
	assert.Contains(t, rec.Body.String(), "PRO-1001")
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
}

func TestUnknownTokenGetsTheVagueCopy(t *testing.T) {
	repo, _ := signedRepo()
	router := newTestRouter(newTestHandler(repo))

	req := httptest.NewRequest(
		http.MethodGet, "/api/v1/rate-confirmation-links/not-a-token/", nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	require.Less(t, rec.Code, http.StatusInternalServerError)
	assert.Contains(t, rec.Body.String(), "no longer valid")
}

func TestConfirmRequiresSignerName(t *testing.T) {
	repo, raw := signedRepo()
	router := newTestRouter(newTestHandler(repo))

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/rate-confirmation-links/"+raw+"/confirm/",
		strings.NewReader(`{"signerTitle":"Dispatcher"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.GreaterOrEqual(t, rec.Code, http.StatusBadRequest)
	assert.Contains(t, rec.Body.String(), "name is required")
}

func TestPerTokenThrottleCapsBursts(t *testing.T) {
	repo, raw := signedRepo()
	router := newTestRouter(newTestHandler(repo))

	var lastCode int
	var lastRec *httptest.ResponseRecorder
	for range throttleBurst + 1 {
		req := httptest.NewRequest(
			http.MethodGet, "/api/v1/rate-confirmation-links/"+raw+"/", nil,
		)
		lastRec = httptest.NewRecorder()
		router.ServeHTTP(lastRec, req)
		lastCode = lastRec.Code
	}

	require.Equal(t, http.StatusTooManyRequests, lastCode)
	assert.NotEmpty(t, lastRec.Header().Get("Retry-After"))
	assert.Equal(t, "publicToken", lastRec.Header().Get("X-RateLimit-Scope"))

	otherReq := httptest.NewRequest(
		http.MethodGet, "/api/v1/rate-confirmation-links/other-token/", nil,
	)
	otherRec := httptest.NewRecorder()
	router.ServeHTTP(otherRec, otherReq)
	assert.NotEqual(t, http.StatusTooManyRequests, otherRec.Code,
		"the throttle is per token, not global")
}

type serverRepo struct {
	rateconfirmationports.RateConfirmationRepository

	tenants []*tokenRepo
}

func (s *serverRepo) GetTokenByHash(
	ctx context.Context,
	tokenHash string,
) (*rateconfirmation.RateConfirmationToken, error) {
	for _, repo := range s.tenants {
		if token, _ := repo.GetTokenByHash(ctx, tokenHash); token != nil {
			return token, nil
		}
	}
	return nil, nil //nolint:nilnil // mirrors the production contract
}

func (s *serverRepo) GetByID(
	ctx context.Context,
	req *rateconfirmationports.GetRateConfirmationByIDRequest,
) (*rateconfirmation.RateConfirmation, error) {
	for _, repo := range s.tenants {
		if entity, err := repo.GetByID(ctx, req); err == nil {
			return entity, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Rate confirmation not found")
}

type brokerageByOrganization map[pulid.ID]bool

func (b brokerageByOrganization) Enabled(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	tenant, ok := dbscope.TenantFrom(ctx)
	if !ok {
		return false, errors.New("a shared server refuses a plan question with no tenant")
	}
	return feature == platformcatalog.FeatureCarrierBrokerage && b[tenant.OrganizationID], nil
}

func (brokerageByOrganization) EnabledAnywhere(
	context.Context,
	platformcatalog.FeatureKey,
) (bool, error) {
	return false, errors.New("a public link never asks about every organization")
}

func preview(router *gin.Engine, raw string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rate-confirmation-links/"+raw+"/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestALinkAnswersOnlyForAnOrganizationWithBrokerageOnTheSameServer(t *testing.T) {
	withBrokerage, withBrokerageRaw := signedRepo()
	without, _ := signedRepo()
	withoutRaw := "raw-other-token"
	without.token.TokenHash = tokenutils.Hash(withoutRaw)
	router := newTestRouter(newTestHandlerWith(
		&serverRepo{tenants: []*tokenRepo{withBrokerage, without}},
		brokerageByOrganization{withBrokerage.entity.OrganizationID: true},
	))

	allowed := preview(router, withBrokerageRaw)
	require.Equal(t, http.StatusOK, allowed.Code, allowed.Body.String())
	assert.Contains(t, allowed.Body.String(), "Eastline Transport")

	refused := preview(router, withoutRaw)
	require.GreaterOrEqual(t, refused.Code, http.StatusBadRequest)
	require.Less(t, refused.Code, http.StatusInternalServerError)
	assert.Contains(
		t,
		refused.Body.String(),
		"no longer valid",
		"the same answer as an unknown link",
	)
	assert.NotContains(t, refused.Body.String(), "Eastline Transport")
}
