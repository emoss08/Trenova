package managedplanresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedplanservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type heldPlan struct {
	managedfeaturesports.StateRepository

	features []platformcatalog.FeatureKey
	changes  []*managedfeatures.Change
}

func (h heldPlan) History(
	context.Context,
	int,
) (*managedfeatures.State, []*managedfeatures.Change, error) {
	return &managedfeatures.State{Features: h.features}, h.changes, nil
}

func resolverFor(managed bool, granted map[string]bool, repo heldPlan) *QueryResolver {
	return &QueryResolver{&Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.GrantingPermissionEngine{Granted: granted},
		},
		ManagedPlanService: managedplanservice.New(managedplanservice.Params{
			Config: &managedconfig.Config{Enabled: managed},
			Repo:   repo,
		}),
	}}
}

func signedIn(t *testing.T) context.Context {
	t.Helper()
	userID := pulid.MustNew("usr_")
	return gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

var canUpdateOrganization = map[string]bool{
	permission.ResourceOrganization.String() + "|" + string(permission.OpUpdate): true,
}

func names(addOns []*gqlmodel.ManagedPlanAddOn) []string {
	out := make([]string, 0, len(addOns))
	for _, addOn := range addOns {
		out = append(out, addOn.Name)
	}
	return out
}

func TestManagedPlanListsAddOnsOnAndOffThePlanInCatalogueOrderByName(t *testing.T) {
	t.Parallel()

	resolver := resolverFor(true, canUpdateOrganization, heldPlan{
		features: []platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
		changes: []*managedfeatures.Change{{
			Added:     []platformcatalog.FeatureKey{"ai.desk"},
			Removed:   []platformcatalog.FeatureKey{"tms.dispatch_intelligence"},
			ChangedAt: 300,
		}},
	})

	plan, err := resolver.ManagedPlan(signedIn(t))
	require.NoError(t, err)

	assert.True(t, plan.Managed)
	assert.Equal(t, []string{"Accounting & Cash", "Trenova AI"}, names(plan.Included))
	assert.Equal(t, "Rating & Revenue Recovery", plan.Available[0].Name)
	assert.Len(t, plan.Available, len(platformcatalog.AddOns())-2)
	assert.Equal(t, "Accounting", plan.Included[0].Features[0].Name)
	assert.Equal(t, "Freight", plan.Base[0].Name)
	require.Len(t, plan.Changes, 1)
	assert.Equal(t, 300, plan.Changes[0].ChangedAt)
	assert.Equal(t, []string{"Dispatch Intelligence"}, names(plan.Changes[0].Left))
	assert.Equal(t, []string{"Trenova AI"}, names(plan.Changes[0].Joined))
	assert.Equal(t, "Dispatch intelligence", plan.Changes[0].Left[0].Features[0].Name)
}

func TestManagedPlanIsRefusedWithoutTheOrganizationUpdatePermission(t *testing.T) {
	t.Parallel()

	resolver := resolverFor(true, map[string]bool{
		permission.ResourceOrganization.String() + "|" + string(permission.OpRead): true,
	}, heldPlan{})

	plan, err := resolver.ManagedPlan(signedIn(t))
	require.Error(t, err)
	assert.Nil(t, plan)
}

func TestManagedPlanOnASelfHostedInstallAnswersNotManaged(t *testing.T) {
	t.Parallel()

	resolver := resolverFor(false, canUpdateOrganization, heldPlan{
		features: []platformcatalog.FeatureKey{"ai.desk"},
	})

	plan, err := resolver.ManagedPlan(signedIn(t))
	require.NoError(t, err)
	assert.False(t, plan.Managed)
	assert.Empty(t, plan.Base)
	assert.Empty(t, plan.Included)
	assert.Empty(t, plan.Available)
	assert.Empty(t, plan.Changes)
}
