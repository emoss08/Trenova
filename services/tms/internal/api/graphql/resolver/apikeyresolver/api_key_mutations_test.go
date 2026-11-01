package apikeyresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/apikeyports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestAPIKeyResolversCheckTheirOwnPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ak_").String()
	input := gqlmodel.APIKeyInput{Name: "Integration"}
	cases := []struct {
		name string
		op   permission.Operation
		call func(ctx context.Context, d *Deps) error
	}{
		{"createApiKey", permission.OpCreate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).CreateAPIKey(ctx, input)
			return err
		}},
		{"updateApiKey", permission.OpUpdate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).UpdateAPIKey(ctx, id, input)
			return err
		}},
		{"rotateApiKey", permission.OpUpdate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).RotateAPIKey(ctx, id)
			return err
		}},
		{"revokeApiKey", permission.OpUpdate, func(ctx context.Context, d *Deps) error {
			_, err := (&MutationResolver{d}).RevokeAPIKey(ctx, id)
			return err
		}},
		{"apiKeyDetail", permission.OpRead, func(ctx context.Context, d *Deps) error {
			_, err := (&QueryResolver{d}).APIKeyDetail(ctx, id)
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := &resolvertest.GrantingPermissionEngine{}
			userID := pulid.MustNew("usr_")
			ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
				PrincipalType:  authctx.PrincipalTypeUser,
				PrincipalID:    userID,
				UserID:         userID,
				OrganizationID: pulid.MustNew("org_"),
				BusinessUnitID: pulid.MustNew("bu_"),
			})

			err := tc.call(ctx, &Deps{
				Core: &resolvercore.Core{L: zap.NewNop(), PermissionEngine: engine},
			})

			require.Error(t, err)
			assert.True(t, errortypes.IsAuthorizationError(err))
			require.Len(t, engine.Requests, 1)
			assert.Equal(t, permission.ResourceAPIKey.String(), engine.Requests[0].Resource)
			assert.Equal(t, tc.op, engine.Requests[0].Operation)
		})
	}
}

func TestCreateAPIKeyRequestFromInputMapsTheGrants(t *testing.T) {
	t.Parallel()

	description := "Carrier portal"
	expires := 1_800_000_000

	req := createAPIKeyRequestFromInput(&gqlmodel.APIKeyInput{
		Name:        "Portal",
		Description: &description,
		ExpiresAt:   &expires,
		Permissions: []*gqlmodel.APIKeyPermissionInput{
			{Resource: "shipment", Operations: []string{"read"}, DataScope: "organization"},
			nil,
		},
	})

	assert.Equal(t, "Portal", req.Name)
	assert.Equal(t, "Carrier portal", req.Description)
	assert.Equal(t, int64(expires), req.ExpiresAt)
	require.Len(t, req.Permissions, 1)
	assert.Equal(t, "shipment", req.Permissions[0].Resource)
	assert.Equal(t, []permission.Operation{permission.OpRead}, req.Permissions[0].Operations)
	assert.Equal(t, permission.DataScopeOrganization, req.Permissions[0].DataScope)

	noExpiry := createAPIKeyRequestFromInput(&gqlmodel.APIKeyInput{Name: "Forever"})
	assert.Zero(t, noExpiry.ExpiresAt)
	assert.Empty(t, noExpiry.Description)
}

func TestIssuedAPIKeyCarriesTheSecretOnlyBesideTheKey(t *testing.T) {
	t.Parallel()

	issued := issuedAPIKeyToModel(&apikeyports.APIKeySecretResponse{
		APIKeyResponse: apikeyports.APIKeyResponse{ID: "ak_1", Name: "Portal", KeyPrefix: "trv_ab"},
		Token:          "trv_ab.secret",
	})

	assert.Equal(t, "trv_ab.secret", issued.Token)
	require.NotNil(t, issued.APIKey)
	assert.Equal(t, "ak_1", issued.APIKey.ID)
	assert.Equal(t, "trv_ab", issued.APIKey.KeyPrefix)
}
