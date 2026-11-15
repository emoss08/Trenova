package userhandler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func completeOrganization() *tenant.Organization {
	return &tenant.Organization{
		ID:             testutil.TestOrgID,
		BusinessUnitID: testutil.TestBuID,
		StateID:        pulid.MustNew("us_"),
		Name:           "Acme Freight",
		ScacCode:       "ACME",
		DOTNumber:      "1234567",
		AddressLine1:   "123 Main St",
		City:           "Chicago",
		PostalCode:     "60601",
		Timezone:       "America/Chicago",
	}
}

func bootstrappedOrganization() *tenant.Organization {
	org := completeOrganization()
	org.AddressLine1 = ""
	org.City = tenant.PlaceholderCity
	org.PostalCode = tenant.PlaceholderPostalCode
	org.ScacCode = tenant.PlaceholderSCAC
	org.DOTNumber = tenant.PlaceholderDOTNumber
	return org
}

func organizationsReturning(
	t *testing.T,
	org *tenant.Organization,
) *tenantportsmock.MockOrganizationService {
	t.Helper()

	orgs := tenantportsmock.NewMockOrganizationService(t)
	orgs.EXPECT().
		GetByID(mock.Anything, tenantports.GetOrganizationByIDRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  testutil.TestOrgID,
				BuID:   testutil.TestBuID,
				UserID: testutil.TestUserID,
			},
		}).
		Return(org, nil).
		Maybe()
	return orgs
}

type organizationUpdateDenied struct {
	*mocks.AllowAllPermissionEngine
}

func (organizationUpdateDenied) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	denied := req.Resource == permission.ResourceOrganization.String() &&
		req.Operation == permission.OpUpdate
	return &permissionports.PermissionCheckResult{Allowed: !denied}, nil
}

type setupAnswer struct {
	OrganizationSetup *struct {
		Complete    bool     `json:"complete"`
		Missing     []string `json:"missing"`
		CanComplete bool     `json:"canComplete"`
	} `json:"organizationSetup"`
}

func currentUser() *tenant.User {
	return &tenant.User{
		ID:                    testutil.TestUserID,
		CurrentOrganizationID: testutil.TestOrgID,
		BusinessUnitID:        testutil.TestBuID,
		Name:                  "Dana Whitfield",
		Username:              "dana",
		EmailAddress:          "dana@acme.example",
		Status:                domaintypes.StatusActive,
	}
}

func getMe(t *testing.T, opts setupOptions) (int, setupAnswer) {
	t.Helper()

	opts.userRepo = makeUserRepoWithGetByID(t, currentUser(), nil)
	handler := setupUserHandler(t, opts)

	ginCtx := testutil.NewGinTestContext().
		WithMethod(http.MethodGet).
		WithPath("/api/v1/users/me/").
		WithDefaultAuthContext()

	handler.RegisterRoutes(ginCtx.Engine.Group("/api/v1"))
	ginCtx.Engine.ServeHTTP(ginCtx.Recorder, ginCtx.Context.Request)

	var answer setupAnswer
	if ginCtx.ResponseCode() == http.StatusOK {
		require.NoError(t, ginCtx.ResponseJSON(&answer))
	}
	return ginCtx.ResponseCode(), answer
}

func TestMeReportsACompleteOrganization(t *testing.T) {
	t.Parallel()

	code, answer := getMe(t, setupOptions{})

	require.Equal(t, http.StatusOK, code)
	require.NotNil(t, answer.OrganizationSetup)
	assert.True(t, answer.OrganizationSetup.Complete)
	assert.Empty(t, answer.OrganizationSetup.Missing)
}

func TestMeTellsAnAdministratorWhatTheOrganizationStillNeeds(t *testing.T) {
	t.Parallel()

	code, answer := getMe(t, setupOptions{
		orgs: organizationsReturning(t, bootstrappedOrganization()),
	})

	require.Equal(t, http.StatusOK, code)
	require.NotNil(t, answer.OrganizationSetup)
	assert.False(t, answer.OrganizationSetup.Complete)
	assert.True(t, answer.OrganizationSetup.CanComplete)
	assert.Equal(t, []string{
		tenant.SetupFieldAddressLine1,
		tenant.SetupFieldCity,
		tenant.SetupFieldPostalCode,
		tenant.SetupFieldSCACCode,
		tenant.SetupFieldDOTNumber,
	}, answer.OrganizationSetup.Missing)
}

func TestMeTellsSomeoneWhoMayNotUpdateTheOrganizationThatTheyCannotFinishIt(t *testing.T) {
	t.Parallel()

	code, answer := getMe(t, setupOptions{
		orgs:       organizationsReturning(t, bootstrappedOrganization()),
		permEngine: organizationUpdateDenied{&mocks.AllowAllPermissionEngine{}},
	})

	require.Equal(t, http.StatusOK, code)
	require.NotNil(t, answer.OrganizationSetup)
	assert.False(t, answer.OrganizationSetup.Complete)
	assert.False(t, answer.OrganizationSetup.CanComplete)
}

func TestMeFailsWhenTheOrganizationCannotBeRead(t *testing.T) {
	t.Parallel()

	orgs := tenantportsmock.NewMockOrganizationService(t)
	orgs.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(nil, errors.New("connection reset"))

	code, _ := getMe(t, setupOptions{orgs: orgs})

	assert.Equal(t, http.StatusInternalServerError, code)
}
