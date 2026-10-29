package shipmentresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formulaVisibilityDeps(granted map[string]bool) *Deps {
	return &Deps{Core: &resolvercore.Core{
		PermissionEngine: &resolvertest.GrantingPermissionEngine{Granted: granted},
	}}
}

func formulaVisibilityContext(t *testing.T) *authctx.AuthContext {
	t.Helper()

	userID := pulid.MustNew("usr_")
	return &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func TestShipmentFormulaContent_NeedsFormulaTemplateRead(t *testing.T) {
	t.Parallel()

	formulaRead := permission.ResourceFormulaTemplate.String() + "|" + string(permission.OpRead)
	shipmentRead := permission.ResourceShipment.String() + "|" + string(permission.OpRead)

	cases := map[string]struct {
		granted      map[string]bool
		wantTemplate bool
	}{
		"a shipment reader does not see the template": {
			granted: map[string]bool{shipmentRead: true},
		},
		"a formula reader sees the template": {
			granted:      map[string]bool{shipmentRead: true, formulaRead: true},
			wantTemplate: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deps := formulaVisibilityDeps(tc.granted)
			ctx := gqlctx.WithAuthContext(t.Context(), formulaVisibilityContext(t))

			template, err := (&ShipmentResolver{deps}).FormulaTemplate(ctx, &gqlmodel.Shipment{
				FormulaTemplate: &gqlmodel.ShipmentFormulaTemplate{
					Name:   "Per Mile",
					Source: "tariff \"Per Mile\"\n",
				},
			})
			require.NoError(t, err)
			assert.Equal(t, tc.wantTemplate, template != nil)

			trace, err := (&ShipmentRatingDetailResolver{deps}).Trace(
				ctx,
				&gqlmodel.ShipmentRatingDetail{
					Trace: map[string]any{"statements": []any{}},
				},
			)
			require.NoError(t, err)
			assert.Equal(t, tc.wantTemplate, trace != nil)
		})
	}
}

func TestShipmentFormulaFields_RequireAnAuthenticatedReader(t *testing.T) {
	t.Parallel()

	deps := formulaVisibilityDeps(nil)

	_, err := (&ShipmentResolver{deps}).FormulaTemplate(t.Context(), &gqlmodel.Shipment{
		FormulaTemplate: &gqlmodel.ShipmentFormulaTemplate{Name: "Per Mile"},
	})
	require.Error(t, err)

	_, err = (&ShipmentRatingDetailResolver{deps}).Trace(
		t.Context(),
		&gqlmodel.ShipmentRatingDetail{
			Trace: map[string]any{"statements": []any{}},
		},
	)
	require.Error(t, err)
}
