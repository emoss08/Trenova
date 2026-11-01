package iamresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/iamports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/iamportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingAccessSettings struct {
	update *iamports.UpdateAccessSettingsRequest
}

func (r *recordingAccessSettings) Get(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) (*iam.AccessSettings, error) {
	return &iam.AccessSettings{OrganizationID: tenantInfo.OrgID}, nil
}

func (r *recordingAccessSettings) Update(
	_ context.Context,
	req *iamports.UpdateAccessSettingsRequest,
) (*iam.AccessSettings, error) {
	r.update = req
	return &iam.AccessSettings{
		OrganizationID:        req.TenantInfo.OrgID,
		RequireRoleActivation: req.RequireRoleActivation,
		Version:               req.Version + 1,
	}, nil
}

func iamAuth() *authctx.AuthContext {
	userID := pulid.MustNew("usr_")
	return &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func granting(
	resource permission.Resource,
	op permission.Operation,
) *resolvertest.GrantingPermissionEngine {
	return &resolvertest.GrantingPermissionEngine{
		Granted: map[string]bool{resource.String() + "|" + string(op): true},
	}
}

func TestIAMResolversCheckTheirOwnPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("idp_").String()
	cases := []struct {
		name     string
		resource permission.Resource
		op       permission.Operation
		call     func(ctx context.Context, d *Deps) error
	}{
		{
			"identityProviders",
			permission.ResourceIdentityProvider,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := (&QueryResolver{d}).IdentityProviders(ctx)
				return err
			},
		},
		{
			"createIdentityProvider",
			permission.ResourceIdentityProvider,
			permission.OpCreate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).CreateIdentityProvider(
					ctx,
					gqlmodel.IdentityProviderInput{},
				)
				return err
			},
		},
		{
			"updateIdentityProvider",
			permission.ResourceIdentityProvider,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).UpdateIdentityProvider(
					ctx,
					id,
					gqlmodel.IdentityProviderInput{},
				)
				return err
			},
		},
		{
			"deleteIdentityProvider",
			permission.ResourceIdentityProvider,
			permission.OpDelete,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).DeleteIdentityProvider(ctx, id)
				return err
			},
		},
		{
			"accessPolicies",
			permission.ResourceAccessPolicy,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := (&QueryResolver{d}).AccessPolicies(ctx)
				return err
			},
		},
		{
			"createAccessPolicy",
			permission.ResourceAccessPolicy,
			permission.OpCreate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).CreateAccessPolicy(
					ctx,
					gqlmodel.AccessPolicyInput{},
				)
				return err
			},
		},
		{
			"updateAccessPolicy",
			permission.ResourceAccessPolicy,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).UpdateAccessPolicy(
					ctx,
					id,
					gqlmodel.AccessPolicyInput{},
				)
				return err
			},
		},
		{
			"deleteAccessPolicy",
			permission.ResourceAccessPolicy,
			permission.OpDelete,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).DeleteAccessPolicy(ctx, id)
				return err
			},
		},
		{
			"accessSettings",
			permission.ResourceAccessPolicy,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := (&QueryResolver{d}).AccessSettings(ctx)
				return err
			},
		},
		{
			"updateAccessSettings",
			permission.ResourceAccessPolicy,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).UpdateAccessSettings(
					ctx,
					gqlmodel.UpdateAccessSettingsInput{},
				)
				return err
			},
		},
		{
			"createSCIMDirectory",
			permission.ResourceSCIMDirectory,
			permission.OpCreate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).CreateSCIMDirectory(
					ctx,
					gqlmodel.SCIMDirectoryInput{},
				)
				return err
			},
		},
		{
			"updateSCIMDirectory",
			permission.ResourceSCIMDirectory,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).UpdateSCIMDirectory(
					ctx,
					id,
					gqlmodel.SCIMDirectoryInput{},
				)
				return err
			},
		},
		{
			"deleteSCIMDirectory",
			permission.ResourceSCIMDirectory,
			permission.OpDelete,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).DeleteSCIMDirectory(ctx, id)
				return err
			},
		},
		{
			"scimTokens",
			permission.ResourceSCIMDirectory,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := (&QueryResolver{d}).ScimTokens(ctx, id)
				return err
			},
		},
		{
			"createSCIMToken",
			permission.ResourceSCIMDirectory,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).CreateSCIMToken(
					ctx,
					id,
					gqlmodel.CreateSCIMTokenInput{Name: "okta"},
				)
				return err
			},
		},
		{
			"revokeSCIMToken",
			permission.ResourceSCIMDirectory,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).RevokeSCIMToken(ctx, id)
				return err
			},
		},
		{
			"createSCIMGroupRoleMapping",
			permission.ResourceSCIMDirectory,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).CreateSCIMGroupRoleMapping(
					ctx,
					id,
					gqlmodel.SCIMGroupRoleMappingInput{},
				)
				return err
			},
		},
		{
			"updateSCIMGroupRoleMapping",
			permission.ResourceSCIMDirectory,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).UpdateSCIMGroupRoleMapping(
					ctx,
					id,
					gqlmodel.SCIMGroupRoleMappingInput{},
				)
				return err
			},
		},
		{
			"deleteSCIMGroupRoleMapping",
			permission.ResourceSCIMDirectory,
			permission.OpUpdate,
			func(ctx context.Context, d *Deps) error {
				_, err := (&MutationResolver{d}).DeleteSCIMGroupRoleMapping(ctx, id)
				return err
			},
		},
		{
			"provisioningAuditRecords",
			permission.ResourceProvisioningAudit,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := (&QueryResolver{d}).ProvisioningAuditRecords(ctx, nil, nil)
				return err
			},
		},
		{
			"securityBriefing",
			permission.ResourceOrganization,
			permission.OpRead,
			func(ctx context.Context, d *Deps) error {
				_, err := (&QueryResolver{d}).SecurityBriefing(ctx)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := &resolvertest.GrantingPermissionEngine{}
			service := iamportsmock.NewMockIAMService(t)
			ctx := gqlctx.WithAuthContext(t.Context(), iamAuth())

			err := tc.call(ctx, &Deps{
				Core:       &resolvercore.Core{L: zap.NewNop(), PermissionEngine: engine},
				IamService: service,
			})

			require.Error(t, err)
			assert.True(t, errortypes.IsAuthorizationError(err))
			require.Len(t, engine.Requests, 1)
			assert.Equal(t, tc.resource.String(), engine.Requests[0].Resource)
			assert.Equal(t, tc.op, engine.Requests[0].Operation)
		})
	}
}

func TestCreateSCIMTokenReturnsTheSecretOnceForTheCallersOrganization(t *testing.T) {
	t.Parallel()

	auth := iamAuth()
	directoryID := pulid.MustNew("scd_")
	expires := 1_800_000_000
	service := iamportsmock.NewMockIAMService(t)
	service.EXPECT().
		CreateSCIMToken(
			mock.Anything,
			auth.OrganizationID,
			directoryID,
			&iamports.SCIMTokenCreateRequest{Name: "Okta", ExpiresAt: int64(expires)},
		).
		Return(&iamports.SCIMTokenCreateResponse{
			SCIMToken: &iam.SCIMToken{Name: "Okta", Prefix: "scim_ab"},
			Token:     "scim_ab.secret",
		}, nil).
		Once()

	issued, err := (&MutationResolver{&Deps{
		Core:       &resolvercore.Core{L: zap.NewNop(), PermissionEngine: granting(permission.ResourceSCIMDirectory, permission.OpUpdate)},
		IamService: service,
	}}).CreateSCIMToken(
		gqlctx.WithAuthContext(t.Context(), auth),
		directoryID.String(),
		gqlmodel.CreateSCIMTokenInput{Name: "Okta", ExpiresAt: &expires},
	)

	require.NoError(t, err)
	assert.Equal(t, "scim_ab.secret", issued.Token)
	assert.Equal(t, "scim_ab", issued.ScimToken.Prefix)
}

func TestUpdateAccessSettingsPassesTheActorAndVersion(t *testing.T) {
	t.Parallel()

	auth := iamAuth()
	settings := &recordingAccessSettings{}

	updated, err := (&MutationResolver{&Deps{
		Core:                  &resolvercore.Core{L: zap.NewNop(), PermissionEngine: granting(permission.ResourceAccessPolicy, permission.OpUpdate)},
		AccessSettingsService: settings,
	}}).UpdateAccessSettings(
		gqlctx.WithAuthContext(t.Context(), auth),
		gqlmodel.UpdateAccessSettingsInput{RequireRoleActivation: true, Version: 3},
	)

	require.NoError(t, err)
	assert.True(t, updated.RequireRoleActivation)
	require.NotNil(t, settings.update)
	assert.Equal(t, int64(3), settings.update.Version)
	assert.Equal(t, auth.OrganizationID, settings.update.TenantInfo.OrgID)
	require.NotNil(t, settings.update.Actor)
	assert.Equal(t, auth.UserID, settings.update.Actor.UserID)
}

func TestSecurityBriefingAsksForTheCallersOwnSections(t *testing.T) {
	t.Parallel()

	auth := iamAuth()
	briefings := iamportsmock.NewMockSecurityBriefingService(t)
	briefings.EXPECT().
		Get(mock.Anything, mock.MatchedBy(func(actor *permissionports.RequestActor) bool {
			return actor.UserID == auth.UserID && actor.OrganizationID == auth.OrganizationID
		})).
		Return(&iam.SecurityBriefing{GeneratedAt: 42}, nil).
		Once()

	briefing, err := (&QueryResolver{&Deps{
		Core:                    &resolvercore.Core{L: zap.NewNop(), PermissionEngine: granting(permission.ResourceOrganization, permission.OpRead)},
		SecurityBriefingService: briefings,
	}}).SecurityBriefing(gqlctx.WithAuthContext(t.Context(), auth))

	require.NoError(t, err)
	assert.Equal(t, int64(42), briefing.GeneratedAt)
}

func TestIdentityProviderRequestFromInputRefusesANonTextClaim(t *testing.T) {
	t.Parallel()

	secret := "s3cret"
	slug := "okta"
	version := 4
	req, err := identityProviderRequestFromInput(&gqlmodel.IdentityProviderInput{
		Name:             "Okta",
		Slug:             &slug,
		Protocol:         "OIDC",
		AttributeMap:     map[string]any{"email": "mail"},
		OidcClientSecret: &secret,
		Version:          &version,
	})
	require.NoError(t, err)
	assert.Equal(t, iam.IdentityProviderProtocolOIDC, req.Protocol)
	assert.Equal(t, map[string]string{"email": "mail"}, req.AttributeMap)
	assert.Equal(t, "s3cret", req.OIDCClientSecret)
	assert.Equal(t, "okta", req.Slug)
	assert.Equal(t, int64(4), req.Version)

	_, err = identityProviderRequestFromInput(&gqlmodel.IdentityProviderInput{
		Name:         "Okta",
		AttributeMap: map[string]any{"email": 4},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestIdentityProviderNeverShowsItsClientSecret(t *testing.T) {
	t.Parallel()

	attributes, err := (&IdentityProviderResolver{&Deps{}}).AttributeMap(
		t.Context(),
		&iam.IdentityProvider{
			OIDCClientSecret: "s3cret",
			AttributeMap:     map[string]string{"email": "mail"},
		},
	)

	require.NoError(t, err)
	assert.Equal(t, map[string]any{"email": "mail"}, attributes)
}
