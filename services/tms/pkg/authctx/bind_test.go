package authctx_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type bindEntity struct {
	ID             pulid.ID `json:"id"`
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	UserID         pulid.ID `json:"userId"`
	Name           string   `json:"name"`
}

type bindAltEntity struct {
	OrgID pulid.ID `json:"orgId"`
	BuID  pulid.ID `json:"buId"`
	Name  string   `json:"name"`
}

func bindContext(t *testing.T, body string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c
}

func bindAuthContext() *authctx.AuthContext {
	return &authctx.AuthContext{
		UserID:         pulid.MustNew("usr_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
	}
}

func TestBindJSON_StampsTenantWhenBodyOmitsIt(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	entity := new(bindEntity)

	require.NoError(t, authctx.BindJSON(bindContext(t, `{"name":"Acme"}`), ac, entity))

	assert.Equal(t, "Acme", entity.Name)
	assert.Equal(t, ac.OrganizationID, entity.OrganizationID)
	assert.Equal(t, ac.BusinessUnitID, entity.BusinessUnitID)
	assert.Equal(t, ac.UserID, entity.UserID)
}

func TestBindJSON_AcceptsBodyNamingTheSessionTenant(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	entity := new(bindEntity)
	body := `{"organizationId":"` + ac.OrganizationID.String() +
		`","businessUnitId":"` + ac.BusinessUnitID.String() + `"}`

	require.NoError(t, authctx.BindJSON(bindContext(t, body), ac, entity))
	assert.Equal(t, ac.OrganizationID, entity.OrganizationID)
}

func TestBindJSON_RejectsForeignOrganization(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	entity := new(bindEntity)
	body := `{"organizationId":"` + pulid.MustNew("org_").String() + `"}`

	err := authctx.BindJSON(bindContext(t, body), ac, entity)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestBindJSON_RejectsForeignBusinessUnit(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	entity := new(bindEntity)
	body := `{"businessUnitId":"` + pulid.MustNew("bu_").String() + `"}`

	err := authctx.BindJSON(bindContext(t, body), ac, entity)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestBindJSON_RejectsForeignTenantOnAlternateFieldNames(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	entity := new(bindAltEntity)
	body := `{"orgId":"` + pulid.MustNew("org_").String() + `"}`

	err := authctx.BindJSON(bindContext(t, body), ac, entity)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestBindJSON_OverwritesSpoofedUser(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	entity := new(bindEntity)
	body := `{"userId":"` + pulid.MustNew("usr_").String() + `"}`

	require.NoError(t, authctx.BindJSON(bindContext(t, body), ac, entity))
	assert.Equal(t, ac.UserID, entity.UserID)
}

func TestBindJSON_KeepsPresetIDOverBodyID(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	pathID := pulid.MustNew("car_")
	entity := &bindEntity{ID: pathID}
	body := `{"id":"` + pulid.MustNew("car_").String() + `"}`

	require.NoError(t, authctx.BindJSON(bindContext(t, body), ac, entity))
	assert.Equal(t, pathID, entity.ID)
}

func TestBindJSON_ReturnsDecodeErrors(t *testing.T) {
	t.Parallel()

	err := authctx.BindJSON(bindContext(t, `{`), bindAuthContext(), new(bindEntity))
	require.Error(t, err)
}

type bindUserEntity struct {
	ID                    pulid.ID `json:"id"`
	BusinessUnitID        pulid.ID `json:"businessUnitId"`
	CurrentOrganizationID pulid.ID `json:"currentOrganizationId"`
}

func TestBindJSON_RejectsForeignCurrentOrganization(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	entity := new(bindUserEntity)
	body := `{"currentOrganizationId":"` + pulid.MustNew("org_").String() + `"}`

	err := authctx.BindJSON(bindContext(t, body), ac, entity)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestBindJSON_StampsCurrentOrganization(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	entity := new(bindUserEntity)

	require.NoError(t, authctx.BindJSON(bindContext(t, `{}`), ac, entity))
	assert.Equal(t, ac.OrganizationID, entity.CurrentOrganizationID)
	assert.Equal(t, ac.BusinessUnitID, entity.BusinessUnitID)
}

type bindTenantInfo struct {
	OrgID  pulid.ID `json:"orgId"`
	BuID   pulid.ID `json:"buId"`
	UserID pulid.ID `json:"userId"`
}

type bindNestedPolicy struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

type bindLine struct {
	OrganizationID pulid.ID `json:"organizationId"`
}

type bindRequestWithNesting struct {
	TenantInfo bindTenantInfo    `json:"tenantInfo"`
	Policy     *bindNestedPolicy `json:"policy"`
	Lines      []bindLine        `json:"lines"`
}

func TestBindJSON_RejectsAForeignTenantInsideTenantInfo(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	body := `{"tenantInfo":{"orgId":"` + pulid.MustNew("org_").String() + `"}}`

	err := authctx.BindJSON(bindContext(t, body), ac, new(bindRequestWithNesting))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestBindJSON_RejectsAForeignTenantInANestedEntity(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	body := `{"policy":{"businessUnitId":"` + pulid.MustNew("bu_").String() + `"}}`

	err := authctx.BindJSON(bindContext(t, body), ac, new(bindRequestWithNesting))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestBindJSON_StampsTenantInfoAndNestedEntities(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	req := new(bindRequestWithNesting)

	require.NoError(t, authctx.BindJSON(bindContext(t, `{"policy":{},"lines":[{}]}`), ac, req))

	assert.Equal(t, ac.OrganizationID, req.TenantInfo.OrgID)
	assert.Equal(t, ac.BusinessUnitID, req.TenantInfo.BuID)
	assert.Equal(t, ac.UserID, req.TenantInfo.UserID)
	require.NotNil(t, req.Policy)
	assert.Equal(t, ac.OrganizationID, req.Policy.OrganizationID)
	assert.Equal(t, ac.BusinessUnitID, req.Policy.BusinessUnitID)
}

func TestBindJSON_LeavesSliceElementsToTheirOwners(t *testing.T) {
	t.Parallel()

	ac := bindAuthContext()
	sibling := pulid.MustNew("org_")
	req := new(bindRequestWithNesting)
	body := `{"lines":[{"organizationId":"` + sibling.String() + `"}]}`

	require.NoError(t, authctx.BindJSON(bindContext(t, body), ac, req))
	assert.Equal(t, sibling, req.Lines[0].OrganizationID)
}
