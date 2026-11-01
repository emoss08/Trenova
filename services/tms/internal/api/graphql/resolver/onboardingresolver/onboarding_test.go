package onboardingresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/onboarding"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/onboardingports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/onboardingportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func completeInput(stateID string) gqlmodel.CompleteOnboardingInput {
	scac := "ACME"
	return gqlmodel.CompleteOnboardingInput{
		Organization: &gqlmodel.OnboardingOrganizationInput{
			Name:         "Acme Freight",
			Timezone:     "America/Chicago",
			AddressLine1: "1 Dock Rd",
			City:         "Joliet",
			StateID:      stateID,
			PostalCode:   "60431",
			SCACCode:     &scac,
		},
		OperationType:  tenant.OperationTypeAsset,
		LoadSampleData: true,
	}
}

func TestCompleteOnboarding_ChecksTheOrganizationPermission(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}

	_, err := (&MutationResolver{deps}).CompleteOnboarding(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		completeInput(pulid.MustNew("st_").String()),
	)

	resolvertest.RequireRefused(
		t,
		err,
		engine,
		permission.ResourceOrganization,
		permission.OpUpdate,
	)
}

func TestCompleteOnboarding_RefusesAnAPIKey(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}
	authCtx := resolvertest.UserAuthContext()
	authCtx.PrincipalType = authctx.PrincipalTypeAPIKey

	_, err := (&MutationResolver{deps}).CompleteOnboarding(
		resolvertest.AuthedContext(t, authCtx),
		completeInput(pulid.MustNew("st_").String()),
	)

	var authzErr *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authzErr)
}

func TestCompleteOnboarding_MapsTheInputForTheCaller(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	stateID := pulid.MustNew("st_")
	state := &onboardingports.OnboardingState{Status: onboarding.StatusCompleted}
	service := onboardingportsmock.NewMockOnboardingService(t)
	service.EXPECT().
		Complete(mock.Anything, mock.MatchedBy(func(req *onboardingports.CompleteOnboardingRequest) bool {
			return req.TenantInfo.OrgID == authCtx.OrganizationID &&
				req.TenantInfo.BuID == authCtx.BusinessUnitID &&
				req.TenantInfo.UserID == authCtx.UserID &&
				req.Actor.UserID == authCtx.UserID &&
				req.Organization.Name == "Acme Freight" &&
				req.Organization.StateID == stateID &&
				req.Organization.ScacCode == "ACME" &&
				req.Organization.DOTNumber == "" &&
				req.OperationType == tenant.OperationTypeAsset &&
				req.LoadSampleData
		})).
		Return(state, nil)
	deps := &Deps{
		Core:              &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		OnboardingService: service,
	}

	got, err := (&MutationResolver{deps}).CompleteOnboarding(
		resolvertest.AuthedContext(t, authCtx),
		completeInput(stateID.String()),
	)
	require.NoError(t, err)
	assert.Same(t, state, got)
}

func TestCompleteOnboarding_NamesAMalformedState(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}}}

	_, err := (&MutationResolver{deps}).CompleteOnboarding(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		completeInput("TX"),
	)

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "organization.stateId", fieldErr.Field)
}

func TestOnboardingState_ReadsTheCallersOrganization(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	state := &onboardingports.OnboardingState{Required: true, Status: onboarding.StatusPending}
	service := onboardingportsmock.NewMockOnboardingService(t)
	service.EXPECT().
		Get(mock.Anything, pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		}).
		Return(state, nil)
	deps := &Deps{Core: &resolvercore.Core{}, OnboardingService: service}

	got, err := (&QueryResolver{deps}).OnboardingState(resolvertest.AuthedContext(t, authCtx))
	require.NoError(t, err)
	assert.Same(t, state, got)
}
