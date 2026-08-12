package rateconfirmationpublichandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/rateconfirmationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type testDB struct{}

func (testDB) DB() *bun.DB                          { return nil }
func (testDB) DBForContext(context.Context) bun.IDB { return nil }
func (testDB) HealthCheck(context.Context) error    { return nil }
func (testDB) IsHealthy(context.Context) bool       { return true }
func (testDB) Close() error                         { return nil }
func (testDB) WithTx(
	ctx context.Context,
	_ ports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type tokenRepo struct {
	repositories.RateConfirmationRepository

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
	req *repositories.GetRateConfirmationByIDRequest,
) (*rateconfirmation.RateConfirmation, error) {
	if f.entity != nil && f.entity.ID == req.RateConfirmationID {
		return f.entity, nil
	}
	return nil, errortypes.NewNotFoundError("Rate confirmation not found")
}

func (f *tokenRepo) MarkTokenUsed(
	_ context.Context,
	_ *repositories.MarkRateConfirmationTokenUsedRequest,
) (bool, error) {
	return true, nil
}

func newTestHandler(repo *tokenRepo) *Handler {
	service := rateconfirmationservice.New(rateconfirmationservice.Params{
		Logger: zap.NewNop(),
		DB:     testDB{},
		Repo:   repo,
	})
	return New(Params{
		Service: service,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: &config.Config{},
		}),
	})
}

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
	assert.Equal(t, "60", lastRec.Header().Get("Retry-After"))

	otherReq := httptest.NewRequest(
		http.MethodGet, "/api/v1/rate-confirmation-links/other-token/", nil,
	)
	otherRec := httptest.NewRecorder()
	router.ServeHTTP(otherRec, otherReq)
	assert.NotEqual(t, http.StatusTooManyRequests, otherRec.Code,
		"the throttle is per token, not global")
}
