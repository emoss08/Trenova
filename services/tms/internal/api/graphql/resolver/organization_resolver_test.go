package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestQueryResolver_Organization_DelegatesToService(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	expected := &tenant.Organization{
		ID:             orgID,
		BusinessUnitID: buID,
		Name:           "Acme Logistics",
	}
	organizationService := mocks.NewMockOrganizationService(t)
	organizationService.EXPECT().
		GetByID(mock.Anything, mock.MatchedBy(func(req repositories.GetOrganizationByIDRequest) bool {
			return req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				req.TenantInfo.UserID == userID &&
				req.IncludeState &&
				!req.IncludeBU
		})).
		Return(expected, nil).
		Once()
	permissionEngine := &recordingPermissionEngine{}
	resolver := &queryResolver{&Resolver{
		organizationService: organizationService,
		permissionEngine:    permissionEngine,
	}}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	result, err := resolver.Organization(ctx, orgID.String(), nil, nil)
	require.NoError(t, err)

	assert.Same(t, expected, result)
	require.NotNil(t, permissionEngine.request)
	assert.Equal(t, permission.ResourceOrganization.String(), permissionEngine.request.Resource)
	assert.Equal(t, permission.OpRead, permissionEngine.request.Operation)
}

func TestMutationResolver_UpdateOrganization_MapsInputToService(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	stateID := pulid.MustNew("us_")
	loginSlug := "acme-logistics"
	logoURL := "organization/logo/acme.png"
	bucketName := "acme-bucket"
	addressLine2 := "Suite 200"
	taxID := "12-3456789"
	expected := &tenant.Organization{
		ID:             orgID,
		BusinessUnitID: buID,
		StateID:        stateID,
		Name:           "Acme Logistics",
		Version:        3,
	}
	organizationService := mocks.NewMockOrganizationService(t)
	organizationService.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(entity *tenant.Organization) bool {
			return entity.ID == orgID &&
				entity.BusinessUnitID == buID &&
				entity.StateID == stateID &&
				entity.Version == 3 &&
				entity.Name == "Acme Logistics" &&
				entity.LoginSlug == loginSlug &&
				entity.ScacCode == "ACME" &&
				entity.DOTNumber == "1234567" &&
				entity.LogoURL == logoURL &&
				entity.BucketName == bucketName &&
				entity.AddressLine1 == "123 Main St" &&
				entity.AddressLine2 == addressLine2 &&
				entity.City == "Chicago" &&
				entity.PostalCode == "60601" &&
				entity.Timezone == "America/Chicago" &&
				entity.TaxID == taxID
		})).
		Return(expected, nil).
		Once()
	permissionEngine := &recordingPermissionEngine{}
	resolver := &mutationResolver{&Resolver{
		organizationService: organizationService,
		permissionEngine:    permissionEngine,
	}}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	result, err := resolver.UpdateOrganization(ctx, orgID.String(), gqlmodel.OrganizationInput{
		Version:      3,
		Name:         "Acme Logistics",
		LoginSlug:    &loginSlug,
		ScacCode:     "ACME",
		DotNumber:    "1234567",
		LogoURL:      &logoURL,
		BucketName:   &bucketName,
		AddressLine1: "123 Main St",
		AddressLine2: &addressLine2,
		City:         "Chicago",
		StateID:      stateID.String(),
		PostalCode:   "60601",
		Timezone:     "America/Chicago",
		TaxID:        &taxID,
	})
	require.NoError(t, err)

	assert.Same(t, expected, result)
	require.NotNil(t, permissionEngine.request)
	assert.Equal(t, permission.ResourceOrganization.String(), permissionEngine.request.Resource)
	assert.Equal(t, permission.OpUpdate, permissionEngine.request.Operation)
}
