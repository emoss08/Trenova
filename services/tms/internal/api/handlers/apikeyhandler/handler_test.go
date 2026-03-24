package apikeyhandler_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/handlers/apikeyhandler"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/domain/apikey"
	permissiondomain "github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/apikeyservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	sharedtestutil "github.com/emoss08/trenova/shared/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupAPIKeyHandler(t *testing.T) (*apikeyhandler.Handler, *mocks.MockAPIKeyRepository) {
	t.Helper()

	repo := mocks.NewMockAPIKeyRepository(t)
	cfg := &config.Config{
		App: config.AppConfig{Debug: true},
		Security: config.SecurityConfig{
			APIToken: config.APITokenConfig{
				Enabled:          true,
				DefaultExpiry:    24 * time.Hour,
				MaxExpiry:        7 * 24 * time.Hour,
				MaxTokensPerUser: 100,
			},
		},
	}
	errorHandler := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: cfg,
	})
	pm := middleware.NewPermissionMiddleware(middleware.PermissionMiddlewareParams{
		PermissionEngine: &mocks.AllowAllPermissionEngine{},
		ErrorHandler:     errorHandler,
	})
	svc := apikeyservice.New(apikeyservice.Params{
		Logger:   zap.NewNop(),
		Repo:     repo,
		Registry: permissiondomain.NewRegistry(),
		Config:   cfg,
	})

	return apikeyhandler.New(apikeyhandler.Params{
		ApiKeyService:        svc,
		ErrorHandler:         errorHandler,
		PermissionMiddleware: pm,
	}), repo
}

func withAPIKeyAuth(
	g *sharedtestutil.GinTestContext,
	apiKeyID, orgID, buID pulid.ID,
) *sharedtestutil.GinTestContext {
	authctx.SetAPIKeyContext(g.Context, apiKeyID, buID, orgID)
	g.Engine.Use(func(c *gin.Context) {
		authctx.SetAPIKeyContext(c, apiKeyID, buID, orgID)
		c.Next()
	})
	return g
}

func TestAPIKeyHandlerListSuccess(t *testing.T) {
	t.Parallel()

	h, repo := setupAPIKeyHandler(t)
	keyID := pulid.MustNew("ak_")
	repo.EXPECT().
		List(mock.Anything, mock.MatchedBy(func(req *repositories.ListAPIKeysRequest) bool {
			return req.Filter != nil &&
				req.Filter.TenantInfo.OrgID == sharedtestutil.TestOrgID &&
				req.Filter.TenantInfo.BuID == sharedtestutil.TestBuID
		})).
		Return(&pagination.ListResult[*apikey.Key]{
			Items: []*apikey.Key{
				{
					ID:             keyID,
					OrganizationID: sharedtestutil.TestOrgID,
					BusinessUnitID: sharedtestutil.TestBuID,
					Name:           "Customer Sync",
					Status:         apikey.StatusActive,
				},
			},
			Total: 1,
		}, nil)

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/api-keys/").
		WithDefaultAuthContext()

	h.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
	var resp map[string]any
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.EqualValues(t, 1, resp["count"])
}

func TestAPIKeyHandlerCreateSuccess(t *testing.T) {
	t.Parallel()

	h, repo := setupAPIKeyHandler(t)
	userID := sharedtestutil.TestUserID

	repo.EXPECT().
		CountActiveByCreator(
			mock.Anything,
			pagination.TenantInfo{
				OrgID:  sharedtestutil.TestOrgID,
				BuID:   sharedtestutil.TestBuID,
				UserID: userID,
			},
			userID,
		).
		Return(0, nil)
	repo.EXPECT().
		CreateWithPermissions(mock.Anything, mock.AnythingOfType("*apikey.Key"), mock.AnythingOfType("[]*apikey.Permission")).
		Run(func(_ context.Context, key *apikey.Key, perms []*apikey.Permission) {
			key.ID = pulid.MustNew("ak_")
			require.Len(t, perms, 1)
			assert.Equal(t, sharedtestutil.TestOrgID, key.OrganizationID)
			assert.Equal(t, sharedtestutil.TestBuID, key.BusinessUnitID)
			assert.Equal(t, userID, key.CreatedByID)
		}).
		Return(nil)

	body := map[string]any{
		"name": "Customer Sync",
		"permissions": []map[string]any{
			{
				"resource":   permissiondomain.ResourceCustomer.String(),
				"operations": []string{string(permissiondomain.OpRead)},
				"dataScope":  string(permissiondomain.DataScopeOrganization),
			},
		},
	}

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/api-keys/").
		WithJSONBody(body).
		WithDefaultAuthContext()

	h.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusCreated, ginCtx.ResponseCode())
	var resp services.APIKeySecretResponse
	require.NoError(t, ginCtx.ResponseJSON(&resp))
	assert.Equal(t, "Customer Sync", resp.Name)
	assert.NotEmpty(t, resp.Token)
}

func TestAPIKeyHandlerGetUsesTenantInfo(t *testing.T) {
	t.Parallel()

	h, repo := setupAPIKeyHandler(t)
	keyID := pulid.MustNew("ak_")

	repo.EXPECT().
		GetByID(
			mock.Anything,
			pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID},
			keyID,
		).
		Return(&apikey.Key{
			ID:             keyID,
			OrganizationID: sharedtestutil.TestOrgID,
			BusinessUnitID: sharedtestutil.TestBuID,
			Name:           "Customer Sync",
			Status:         apikey.StatusActive,
		}, nil)

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/api-keys/"+keyID.String()+"/").
		WithParam("apiKeyID", keyID.String()).
		WithDefaultAuthContext()

	h.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusOK, ginCtx.ResponseCode())
}

func TestAPIKeyHandlerRotateAndRevokeSuccess(t *testing.T) {
	t.Parallel()

	h, repo := setupAPIKeyHandler(t)
	keyID := pulid.MustNew("ak_")
	key := &apikey.Key{
		ID:             keyID,
		OrganizationID: sharedtestutil.TestOrgID,
		BusinessUnitID: sharedtestutil.TestBuID,
		Name:           "Rotating Key",
		Status:         apikey.StatusActive,
	}

	repo.EXPECT().
		GetByID(mock.Anything, pagination.TenantInfo{OrgID: sharedtestutil.TestOrgID, BuID: sharedtestutil.TestBuID}, keyID).
		Return(key, nil).Twice()
	repo.EXPECT().Update(mock.Anything, mock.AnythingOfType("*apikey.Key")).Return(nil).Twice()

	rotateCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/api-keys/"+keyID.String()+"/rotate/").
		WithParam("apiKeyID", keyID.String()).
		WithDefaultAuthContext()

	h.RegisterRoutes(rotateCtx.Engine.Group("/api/v1"))
	rotateCtx.Engine.ServeHTTP(rotateCtx.Recorder, rotateCtx.Context.Request)
	assert.Equal(t, http.StatusOK, rotateCtx.ResponseCode())

	revokeCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/api-keys/"+keyID.String()+"/revoke/").
		WithParam("apiKeyID", keyID.String()).
		WithDefaultAuthContext()

	h.RegisterRoutes(revokeCtx.Engine.Group("/api/v1"))
	revokeCtx.Engine.ServeHTTP(revokeCtx.Recorder, revokeCtx.Context.Request)
	assert.Equal(t, http.StatusOK, revokeCtx.ResponseCode())
}

func TestAPIKeyHandlerRejectsMachinePrincipal(t *testing.T) {
	t.Parallel()

	h, _ := setupAPIKeyHandler(t)
	apiKeyID := pulid.MustNew("ak_")

	ginCtx := sharedtestutil.NewGinTestContext().
		WithMethod(http.MethodPost).
		WithPath("/api/v1/api-keys/")
	withAPIKeyAuth(ginCtx, apiKeyID, sharedtestutil.TestOrgID, sharedtestutil.TestBuID)

	h.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode())
}
