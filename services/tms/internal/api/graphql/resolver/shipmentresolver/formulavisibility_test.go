package shipmentresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func formulaVisibilityDeps(granted map[string]bool) *Deps {
	return &Deps{Core: &base.Core{
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

func TestShipmentFormulaFields_NeedFormulaTemplateRead(t *testing.T) {
	t.Parallel()

	formulaRead := permission.ResourceFormulaTemplate.String() + "|" + string(permission.OpRead)
	shipmentRead := permission.ResourceShipment.String() + "|" + string(permission.OpRead)

	cases := map[string]struct {
		granted        map[string]bool
		wantTemplate   bool
		wantExpression string
	}{
		"a shipment reader sees neither": {
			granted:        map[string]bool{shipmentRead: true},
			wantExpression: "",
		},
		"a formula reader sees both": {
			granted:        map[string]bool{shipmentRead: true, formulaRead: true},
			wantTemplate:   true,
			wantExpression: "baseRate * totalDistance",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			deps := formulaVisibilityDeps(tc.granted)
			ctx := gqlctx.WithAuthContext(t.Context(), formulaVisibilityContext(t))

			template, err := (&ShipmentResolver{deps}).FormulaTemplate(ctx, &gqlmodel.Shipment{
				FormulaTemplate: &gqlmodel.ShipmentFormulaTemplate{
					Name:       "Per Mile",
					Expression: "baseRate * totalDistance",
				},
			})
			require.NoError(t, err)
			assert.Equal(t, tc.wantTemplate, template != nil)

			expression, err := (&ShipmentRatingDetailResolver{deps}).Expression(
				ctx,
				&gqlmodel.ShipmentRatingDetail{Expression: "baseRate * totalDistance"},
			)
			require.NoError(t, err)
			assert.Equal(t, tc.wantExpression, expression)
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
}
