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
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type heldPlan struct {
	managedfeaturesports.StateRepository

	features []platformcatalog.FeatureKey
	endings  []managedfeatures.AddOnEnding
	planURL  string
	planEnd  int64
	backup   int
	changes  []*managedfeatures.Change
	usage    *managedusage.Usage
}

func (h heldPlan) History(
	context.Context,
	managedscope.Scope,
	int,
) (*managedfeatures.State, []*managedfeatures.Change, error) {
	return &managedfeatures.State{
		Features:        h.features,
		AddOnEndings:    h.endings,
		PlanURL:         h.planURL,
		PlanEndsAt:      h.planEnd,
		FinalBackupDays: h.backup,
	}, h.changes, nil
}

func TestManagedPlanSaysHowLongTheFinalBackupIsKept(t *testing.T) {
	t.Parallel()

	told := resolverFor(true, canUpdateOrganization, heldPlan{backup: 7})
	plan, err := told.ManagedPlan(signedIn(t))
	require.NoError(t, err)
	assert.Equal(t, 7, plan.FinalBackupDays)

	untold := resolverFor(true, canUpdateOrganization, heldPlan{})
	plan, err = untold.ManagedPlan(signedIn(t))
	require.NoError(t, err)
	assert.Equal(t, 30, plan.FinalBackupDays, "30 until the control plane has said")

	selfHosted := resolverFor(false, canUpdateOrganization, heldPlan{backup: 7})
	plan, err = selfHosted.ManagedPlan(signedIn(t))
	require.NoError(t, err)
	assert.Equal(t, 30, plan.FinalBackupDays)
}

func TestManagedPlanSaysWhenThePlanEndsOnlyWhileItIsEnding(t *testing.T) {
	t.Parallel()

	ending := resolverFor(true, canUpdateOrganization, heldPlan{
		features: []platformcatalog.FeatureKey{"tms.accounting"},
		planEnd:  1_797_206_400,
	})
	plan, err := ending.ManagedPlan(signedIn(t))
	require.NoError(t, err)
	require.NotNil(t, plan.EndsAt)
	assert.Equal(t, 1_797_206_400, *plan.EndsAt)

	kept := resolverFor(true, canUpdateOrganization, heldPlan{
		features: []platformcatalog.FeatureKey{"tms.accounting"},
	})
	plan, err = kept.ManagedPlan(signedIn(t))
	require.NoError(t, err)
	assert.Nil(t, plan.EndsAt)

	selfHosted := resolverFor(false, canUpdateOrganization, heldPlan{planEnd: 1_797_206_400})
	plan, err = selfHosted.ManagedPlan(signedIn(t))
	require.NoError(t, err)
	assert.Nil(t, plan.EndsAt, "a self-hosted install has no plan to end")
}

func (h heldPlan) Usage(context.Context, managedscope.Scope) (*managedusage.Usage, error) {
	return h.usage, nil
}

func resolverFor(managed bool, granted map[string]bool, repo heldPlan) *QueryResolver {
	return &QueryResolver{&Deps{
		Core: &resolvercore.Core{
			PermissionEngine: &resolvertest.GrantingPermissionEngine{Granted: granted},
		},
		ManagedPlanService: managedplanservice.New(managedplanservice.Params{
			Config: &managedconfig.Config{Enabled: managed},
			Repo:   repo,
			Usage:  repo,
			Scopes: managedscope.NewResolver(false),
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

func TestManagedPlanAnswersTheManageLinkUsageAndScheduledEndings(t *testing.T) {
	t.Parallel()

	shipments, _, _ := platformcatalog.LookupMeter(platformcatalog.MeterShipmentsTotal)
	storage, _, _ := platformcatalog.LookupMeter(platformcatalog.MeterDocumentStorageBytes)
	resolver := resolverFor(true, canUpdateOrganization, heldPlan{
		features: []platformcatalog.FeatureKey{"ai.desk", "tms.accounting"},
		endings: []managedfeatures.AddOnEnding{
			{AddOn: platformcatalog.AddOnAccounting, EndsAt: 1_797_206_400},
		},
		planURL: "https://console.trenova.cloud/instances/0193/plan",
		changes: []*managedfeatures.Change{{
			Ending: []managedfeatures.AddOnEnding{
				{AddOn: platformcatalog.AddOnAccounting, EndsAt: 1_797_206_400},
			},
			Kept:      []platformcatalog.AddOnKey{platformcatalog.AddOnAI},
			ChangedAt: 500,
		}},
		usage: &managedusage.Usage{
			PeriodStart: 1_794_614_400,
			PeriodEnd:   1_797_206_400,
			Meters: []managedusage.MeterUsage{
				{Meter: shipments, Unit: managedusage.UnitCount, Used: 312, Included: 1_500},
				{
					Meter:    storage,
					Unit:     managedusage.UnitBytes,
					Used:     18_400_000_000,
					Included: 100_000_000_000,
				},
			},
		},
	})

	plan, err := resolver.ManagedPlan(signedIn(t))
	require.NoError(t, err)

	require.NotNil(t, plan.ManageURL)
	assert.Equal(t, "https://console.trenova.cloud/instances/0193/plan", *plan.ManageURL)
	require.NotNil(t, plan.Included[0].EndsAt)
	assert.Equal(t, 1_797_206_400, *plan.Included[0].EndsAt)
	assert.Nil(t, plan.Included[1].EndsAt, "an add-on with no scheduled end has no end")

	require.NotNil(t, plan.Usage)
	assert.Equal(t, 1_794_614_400, plan.Usage.PeriodStart)
	assert.Equal(t, 1_797_206_400, plan.Usage.PeriodEnd)
	require.Len(t, plan.Usage.Meters, 2)
	assert.Equal(t, "shipments.total", plan.Usage.Meters[0].Key)
	assert.Equal(t, "Shipments", plan.Usage.Meters[0].Name)
	assert.Equal(t, gqlmodel.ManagedPlanMeterUnitCount, plan.Usage.Meters[0].Unit)
	assert.Equal(t, 312, plan.Usage.Meters[0].Used)
	assert.Equal(t, "Document storage", plan.Usage.Meters[1].Name)
	assert.Equal(t, gqlmodel.ManagedPlanMeterUnitBytes, plan.Usage.Meters[1].Unit)
	assert.Equal(
		t,
		18_400_000_000,
		plan.Usage.Meters[1].Used,
		"bytes past 32 bits are carried whole",
	)

	require.Len(t, plan.Changes, 1)
	require.Len(t, plan.Changes[0].Ending, 1)
	assert.Equal(t, "Accounting & Cash", plan.Changes[0].Ending[0].AddOn.Name)
	assert.Equal(t, 1_797_206_400, plan.Changes[0].Ending[0].EndsAt)
	assert.Equal(t, []string{"Trenova AI"}, names(plan.Changes[0].Kept))
}

func TestManagedPlanHasNoManageLinkOrUsageUntilTheControlPlaneSendsThem(t *testing.T) {
	t.Parallel()

	resolver := resolverFor(true, canUpdateOrganization, heldPlan{
		features: []platformcatalog.FeatureKey{"ai.desk"},
	})

	plan, err := resolver.ManagedPlan(signedIn(t))
	require.NoError(t, err)
	assert.Nil(t, plan.ManageURL)
	assert.Nil(t, plan.Usage)
}

func TestManagedPlanNamesFeaturesAddOnsAndMetersInTheReadersLanguage(t *testing.T) {
	t.Parallel()

	shipments, _, _ := platformcatalog.LookupMeter(platformcatalog.MeterShipmentsTotal)
	resolver := resolverFor(true, canUpdateOrganization, heldPlan{
		features: []platformcatalog.FeatureKey{"tms.accounting"},
		endings: []managedfeatures.AddOnEnding{
			{AddOn: platformcatalog.AddOnAccounting, EndsAt: 1_797_206_400},
		},
		changes: []*managedfeatures.Change{{
			Ending: []managedfeatures.AddOnEnding{
				{AddOn: platformcatalog.AddOnAccounting, EndsAt: 1_797_206_400},
			},
			ChangedAt: 500,
		}},
		usage: &managedusage.Usage{
			PeriodStart: 1_794_614_400,
			PeriodEnd:   1_797_206_400,
			Meters: []managedusage.MeterUsage{
				{Meter: shipments, Unit: managedusage.UnitCount, Used: 3, Included: 10},
			},
		},
	})

	plan, err := resolver.ManagedPlan(i18n.WithLocale(signedIn(t), i18n.ES))
	require.NoError(t, err)

	assert.Equal(t, "Flete", plan.Base[0].Name)
	assert.Equal(t, "Contabilidad y efectivo", plan.Included[0].Name)
	assert.Equal(t, "Contabilidad", plan.Included[0].Features[0].Name)
	assert.Equal(t, "Contabilidad y efectivo", plan.Changes[0].Ending[0].AddOn.Name)
	assert.Equal(t, "Envíos", plan.Usage.Meters[0].Name)
	assert.Equal(t, "ACC", plan.Included[0].Key, "the key is never translated")
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
	assert.Nil(t, plan.ManageURL)
	assert.Nil(t, plan.Usage)
}
