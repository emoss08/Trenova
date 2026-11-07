package scimhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/scimproto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const validToken = "scim_AbCdEfGhIjKl.secret"

type fakeProvisioner struct {
	scimports.Provisioner

	dir     *scimports.Directory
	scope   dbscope.Tenant
	created *scimproto.User
}

func (f *fakeProvisioner) Authenticate(
	_ context.Context,
	req scimports.AuthenticateRequest,
) (*scimports.Directory, error) {
	if req.Token != validToken || req.TenantSlug != f.dir.TenantSlug {
		return nil, scimproto.Unauthorized("The bearer token is not valid for this directory")
	}
	return f.dir, nil
}

func (f *fakeProvisioner) GetUser(
	ctx context.Context,
	dir *scimports.Directory,
	userID pulid.ID,
) (*scimproto.User, error) {
	f.scope, _ = dbscope.TenantFrom(ctx)
	if dir != f.dir || userID != "usr_known" {
		return nil, errortypes.NewNotFoundError("SCIM user not found")
	}
	return &scimproto.User{
		Schemas:  []string{scimproto.SchemaUser},
		ID:       userID.String(),
		UserName: "ada@example.com",
		Active:   scimproto.BoolOf(true),
	}, nil
}

func (f *fakeProvisioner) CreateUser(
	_ context.Context,
	_ *scimports.Directory,
	resource *scimproto.User,
) (*scimproto.User, error) {
	f.created = resource
	resource.ID = "usr_created"
	return resource, nil
}

func newRouter(t *testing.T) (*gin.Engine, *fakeProvisioner) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	provisioner := &fakeProvisioner{dir: &scimports.Directory{
		ID:             pulid.MustNew("scd_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		TokenID:        pulid.MustNew("sct_"),
		TenantSlug:     "acme",
	}}
	handler := New(Params{
		Provisioner: provisioner,
		RateLimiter: &middleware.RateLimiter{},
		AppConfig:   &appconfig.Config{WebBaseURL: "https://tms.example.com"},
		Logger:      zap.NewNop(),
	})

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))

	return router, provisioner
}

func serve(router *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", scimproto.MediaType)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return w
}

func decodeError(t *testing.T, w *httptest.ResponseRecorder) scimproto.ErrorBody {
	t.Helper()

	var body scimproto.ErrorBody
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, []string{scimproto.SchemaError}, body.Schemas)

	return body
}

func TestSCIMRoutes_RefuseRequestsWithoutTheDirectorysToken(t *testing.T) {
	t.Parallel()

	router, _ := newRouter(t)

	for name, tc := range map[string]struct {
		path  string
		token string
	}{
		"no token":         {path: "/api/v1/scim/acme/v2/Users/usr_known"},
		"wrong token":      {path: "/api/v1/scim/acme/v2/Users/usr_known", token: "scim_other.secret"},
		"another slug":     {path: "/api/v1/scim/globex/v2/Users/usr_known", token: validToken},
		"discovery, too":   {path: "/api/v1/scim/acme/v2/ServiceProviderConfig"},
		"not a bearer one": {path: "/api/v1/scim/acme/v2/Users", token: ""},
	} {
		w := serve(router, http.MethodGet, tc.path, tc.token, "")

		assert.Equal(t, http.StatusUnauthorized, w.Code, name)
		assert.Equal(t, `Bearer realm="SCIM"`, w.Header().Get("WWW-Authenticate"), name)
		assert.Equal(t, scimproto.MediaType, w.Header().Get("Content-Type"), name)
		assert.Equal(t, "401", decodeError(t, w).Status, name)
	}
}

func TestSCIMRoutes_ScopeTheRequestToTheTokensTenant(t *testing.T) {
	t.Parallel()

	router, provisioner := newRouter(t)

	w := serve(router, http.MethodGet, "/api/v1/scim/acme/v2/Users/usr_known", validToken, "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, provisioner.dir.OrganizationID, provisioner.scope.OrganizationID)
	assert.Equal(t, provisioner.dir.BusinessUnitID, provisioner.scope.BusinessUnitID)
	var user scimproto.User
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &user))
	assert.Equal(t,
		"https://tms.example.com/api/v1/scim/acme/v2/Users/usr_known",
		user.Meta.Location,
	)
}

func TestSCIMRoutes_AnswerNotFoundInSCIMForm(t *testing.T) {
	t.Parallel()

	router, _ := newRouter(t)

	w := serve(router, http.MethodGet, "/api/v1/scim/acme/v2/Users/usr_unknown", validToken, "")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "404", decodeError(t, w).Status)
}

func TestSCIMRoutes_CreateAnswersWithTheResourceAndItsLocation(t *testing.T) {
	t.Parallel()

	router, provisioner := newRouter(t)

	w := serve(router, http.MethodPost, "/api/v1/scim/acme/v2/Users", validToken, `{
		"schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
		"userName": "grace@example.com",
		"active": true
	}`)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t,
		"https://tms.example.com/api/v1/scim/acme/v2/Users/usr_created",
		w.Header().Get("Location"),
	)
	require.NotNil(t, provisioner.created)
	assert.Equal(t, "grace@example.com", provisioner.created.UserName)
}

func TestSCIMRoutes_RefuseABodyWithoutTheUserSchema(t *testing.T) {
	t.Parallel()

	router, provisioner := newRouter(t)

	w := serve(
		router,
		http.MethodPost,
		"/api/v1/scim/acme/v2/Users",
		validToken,
		`{"userName": "x"}`,
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, scimproto.ScimTypeInvalidValue, decodeError(t, w).ScimType)
	assert.Nil(t, provisioner.created)
}

func TestSCIMRoutes_RefuseAnOversizedBody(t *testing.T) {
	t.Parallel()

	router, provisioner := newRouter(t)
	body := `{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"userName":"` +
		strings.Repeat("a", maxBodyBytes) + `"}`

	w := serve(router, http.MethodPost, "/api/v1/scim/acme/v2/Users", validToken, body)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.Nil(t, provisioner.created)
}
