package managedfeaturesservice_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestTheGateWithNoList(t *testing.T) {
	t.Parallel()

	svc, _ := newService(newRepo())
	assert.True(t, enabled(t, svc, platformcatalog.FeatureCoreTMS), "a base feature is on")
	assert.False(t, enabled(t, svc, platformcatalog.FeatureAccounting), "an add-on is off")
}

func TestTheGateWithAList(t *testing.T) {
	t.Parallel()

	svc, _ := newService(newRepo(instanceListed(platformcatalog.FeatureAccounting)))
	assert.True(t, enabled(t, svc, platformcatalog.FeatureAccounting), "a listed add-on is on")
	assert.False(t, enabled(t, svc, platformcatalog.FeatureAIDesk), "an unlisted add-on is off")
	assert.True(t, enabled(t, svc, platformcatalog.FeatureDispatch), "a base feature is on")
	assert.False(t, enabled(t, svc, "tms.teleport"), "an unknown key is off")
}

func TestADedicatedInstanceAnswersForItselfWhateverTenantTheContextCarries(t *testing.T) {
	t.Parallel()

	svc, _ := newService(newRepo(instanceListed(platformcatalog.FeatureAccounting)))
	assert.True(t, enabledFor(t, svc, tenant(), platformcatalog.FeatureAccounting))
	assert.True(t, enabled(t, svc, platformcatalog.FeatureAccounting), "and with no tenant at all")
}

func TestTheGateSurfacesARepositoryError(t *testing.T) {
	t.Parallel()

	down := errors.New("database down")
	repo := newRepo()
	repo.err = down
	svc, _ := newService(repo)
	_, err := svc.Enabled(t.Context(), platformcatalog.FeatureAccounting)
	require.ErrorIs(t, err, down)
}

func TestTheListIsCached(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	svc, _ := newService(repo)
	for range 5 {
		_, err := svc.Enabled(t.Context(), platformcatalog.FeatureAccounting)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, repo.gets[managedscope.InstanceKey])
}

func TestSetFeaturesStoresTheNormalisedList(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	svc, logs := newService(repo)
	setFeatures(t, svc, []platformcatalog.FeatureKey{
		"tms.rating", "tms.core", "ai.desk", "tms.teleport", "ai.desk",
	})

	state := repo.instance()
	require.NotNil(t, state)
	assert.Equal(t, managedscope.InstanceKey, state.ScopeKey)
	assert.Equal(t, managedfeatures.LifecycleActive, state.Lifecycle)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk", "tms.rating"}, state.Features)
	assert.Equal(t, now, state.ReceivedAt)

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, zapcore.WarnLevel, entries[0].Level)
	fields := entries[0].ContextMap()
	assert.Equal(t, int64(2), fields["features"])
	assert.Equal(t, managedscope.InstanceKey, fields["scope"])
	assert.Equal(t, []any{"ai.desk", "tms.rating"}, fields["added"])
	assert.Equal(t, []any{}, fields["removed"])
	assert.Equal(t, []any{"tms.teleport"}, fields["ignored"])
}

func TestSetFeaturesStoresAnEmptyList(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	svc, _ := newService(repo)
	setFeatures(t, svc, nil)
	require.NotNil(t, repo.instance())
	assert.Equal(t, []platformcatalog.FeatureKey{}, repo.instance().Features)
}

func TestSetFeaturesWritesNothingWhenTheListIsUnchanged(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	svc, logs := newService(repo)
	setFeatures(t, svc, []platformcatalog.FeatureKey{"tms.accounting", "ai.desk"})
	setFeatures(t, svc,
		[]platformcatalog.FeatureKey{"ai.desk", "tms.accounting", "tms.core"})

	assert.Equal(t, 1, repo.saves)
	assert.Equal(t, 1, logs.Len())
}

func TestSetFeaturesStoresAChangeAndTheGateSeesItAtOnce(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	svc, logs := newService(repo)
	setFeatures(t, svc, []platformcatalog.FeatureKey{"tms.accounting", "ai.desk"})
	assert.True(t, enabled(t, svc, platformcatalog.FeatureAccounting))

	setFeatures(t, svc, []platformcatalog.FeatureKey{"ai.desk", "tms.rating"})
	assert.Equal(t, 2, repo.saves)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk", "tms.rating"}, repo.instance().Features)
	assert.False(t, enabled(t, svc, platformcatalog.FeatureAccounting), "the cache was dropped")
	assert.True(t, enabled(t, svc, platformcatalog.FeatureRating))

	entries := logs.All()
	require.Len(t, entries, 2)
	assert.Equal(t, zapcore.InfoLevel, entries[1].Level)
	fields := entries[1].ContextMap()
	assert.Equal(t, int64(2), fields["features"])
	assert.Equal(t, []any{"tms.rating"}, fields["added"])
	assert.Equal(t, []any{"tms.accounting"}, fields["removed"])
	assert.NotContains(t, fields, "ignored")
}

func TestSetFeaturesSurfacesARepositoryError(t *testing.T) {
	t.Parallel()

	down := errors.New("database down")
	repo := newRepo()
	repo.err = down
	svc, _ := newService(repo)
	_, err := svc.SetFeatures(t.Context(), managedscope.Instance(), &managedfeatures.Received{})
	require.ErrorIs(t, err, down)
	assert.Equal(t, 0, repo.saves)
}

func TestStateIsNilBeforeAList(t *testing.T) {
	t.Parallel()

	svc, _ := newService(newRepo())
	state, err := svc.State(t.Context(), managedscope.Instance())
	require.NoError(t, err)
	assert.Nil(t, state)
}

func TestSetFeaturesRefusesABadPlanURLBeforeWritingAnything(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	svc, _ := newService(repo)
	bad := "javascript:alert(1)"
	_, err := svc.SetFeatures(
		t.Context(),
		managedscope.Instance(),
		&managedfeatures.Received{PlanURL: &bad},
	)

	_, refused := errortypes.RefusalOf(err)
	require.True(t, refused)
	assert.Equal(t, 0, repo.saves)
	assert.Nil(t, repo.instance())
}

func TestSetFeaturesStoresThePlanURLAndEndingsAndKeepsThemWhenLeftOut(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	svc, _ := newService(repo)
	planURL := "https://console.trenova.cloud/instances/1/plan"
	endings := []managedfeatures.AddOnEnding{
		{AddOn: platformcatalog.AddOnAccounting, EndsAt: now + 86_400},
	}
	_, err := svc.SetFeatures(t.Context(), managedscope.Instance(), &managedfeatures.Received{
		Features:     []platformcatalog.FeatureKey{"tms.accounting"},
		PlanURL:      &planURL,
		AddOnEndings: &endings,
	})
	require.NoError(t, err)

	setFeatures(t, svc, []platformcatalog.FeatureKey{"tms.accounting"})

	assert.Equal(t, 1, repo.saves, "a request without them changes nothing")
	state, err := svc.State(t.Context(), managedscope.Instance())
	require.NoError(t, err)
	assert.Equal(t, planURL, state.PlanURL)
	assert.Equal(t, endings, state.AddOnEndings)
	assert.Equal(t, int64(now+86_400), state.EndsAt(platformcatalog.AddOnAccounting))
}

func TestAnUnknownAddOnIsIgnoredAndLoggedAtMostOnceAnHour(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	svc, logs := newService(repo)
	endings := []managedfeatures.AddOnEnding{{AddOn: "ZZZ", EndsAt: now + 86_400}}
	received := &managedfeatures.Received{
		Features:     []platformcatalog.FeatureKey{"tms.accounting"},
		AddOnEndings: &endings,
	}
	for range 3 {
		_, err := svc.SetFeatures(t.Context(), managedscope.Instance(), received)
		require.NoError(t, err)
	}

	assert.Empty(t, repo.instance().AddOnEndings)
	entries := logs.All()
	require.Len(t, entries, 2, "the first write, then one reminder for the repeats")
	for _, entry := range entries {
		assert.Equal(t, zapcore.WarnLevel, entry.Level)
		assert.Equal(t, []any{"ZZZ"}, entry.ContextMap()["ignoredAddOns"])
	}
}

func TestOnASharedServerEachOrganizationHasItsOwnPlan(t *testing.T) {
	t.Parallel()

	withAccounting, withDesk := tenant(), tenant()
	repo := newRepo(
		listed(managedscope.Organization(withAccounting), platformcatalog.FeatureAccounting),
		listed(managedscope.Organization(withDesk), platformcatalog.FeatureAIDesk),
	)
	svc, _ := newServiceIn(repo, true)

	assert.True(t, enabledFor(t, svc, withAccounting, platformcatalog.FeatureAccounting))
	assert.False(t, enabledFor(t, svc, withAccounting, platformcatalog.FeatureAIDesk))
	assert.True(t, enabledFor(t, svc, withDesk, platformcatalog.FeatureAIDesk))
	assert.False(t, enabledFor(t, svc, withDesk, platformcatalog.FeatureAccounting))
	assert.True(t, enabledFor(t, svc, withDesk, platformcatalog.FeatureCoreTMS),
		"an organization with a plan has the base plan")
	stranger := tenant()
	assert.False(t, enabledFor(t, svc, stranger, platformcatalog.FeatureCoreTMS),
		"an organization the control plane has sent no plan for has nothing, the base plan included")
	assert.False(t, enabledFor(t, svc, stranger, platformcatalog.FeatureAccounting))
}

func TestOnASharedServerAQuestionWithNoTenantIsAnError(t *testing.T) {
	t.Parallel()

	svc, _ := newServiceIn(newRepo(), true)

	_, err := svc.Enabled(t.Context(), platformcatalog.FeatureCoreTMS)
	require.ErrorIs(t, err, managedscope.ErrNoTenantScope, "a bare context")

	system := dbscope.WithSystem(t.Context(), "a sweep")
	_, err = svc.Enabled(system, platformcatalog.FeatureAccounting)
	require.ErrorIs(t, err, managedscope.ErrNoTenantScope, "a system context")
}

func TestASuspendedOrganizationHasNoFeaturesAtAll(t *testing.T) {
	t.Parallel()

	suspended, active := tenant(), tenant()
	repo := newRepo(
		listed(managedscope.Organization(suspended), platformcatalog.FeatureAccounting),
		listed(managedscope.Organization(active), platformcatalog.FeatureAccounting),
	)
	svc, _ := newServiceIn(repo, true)
	require.True(t, enabledFor(t, svc, suspended, platformcatalog.FeatureCoreTMS))

	_, err := svc.SetLifecycle(
		t.Context(),
		managedscope.Organization(suspended),
		managedfeatures.LifecycleSuspended,
	)
	require.NoError(t, err)

	assert.False(t, enabledFor(t, svc, suspended, platformcatalog.FeatureCoreTMS), "base included")
	assert.False(t, enabledFor(t, svc, suspended, platformcatalog.FeatureAccounting))
	assert.True(t, enabledFor(t, svc, active, platformcatalog.FeatureAccounting),
		"another organization is untouched")

	_, err = svc.SetLifecycle(
		t.Context(),
		managedscope.Organization(suspended),
		managedfeatures.LifecycleActive,
	)
	require.NoError(t, err)
	assert.True(t, enabledFor(t, svc, suspended, platformcatalog.FeatureAccounting),
		"resuming gives the plan back")
}

func TestAnOrganizationSuspendedBeforeItsPlanArrivesKeepsTheStartingPlan(t *testing.T) {
	t.Parallel()

	org := tenant()
	scope := managedscope.Organization(org)
	repo := newRepo()
	svc, _ := newServiceIn(repo, true)

	_, err := svc.SetLifecycle(t.Context(), scope, managedfeatures.LifecycleSuspended)
	require.NoError(t, err)
	assert.False(t, enabledFor(t, svc, org, platformcatalog.FeatureCoreTMS))

	_, err = svc.SetFeatures(t.Context(), scope, &managedfeatures.Received{
		Features: []platformcatalog.FeatureKey{platformcatalog.FeatureAccounting},
	})
	require.NoError(t, err)
	assert.False(t, enabledFor(t, svc, org, platformcatalog.FeatureAccounting),
		"a new list does not lift a suspension")
	require.Len(t, repo.changes, 1)
	assert.True(t, repo.changes[0].Starting)
	assert.Equal(t, scope.Key(), repo.changes[0].ScopeKey)
}

func TestAWriteDropsOnlyItsOwnScopeFromTheCache(t *testing.T) {
	t.Parallel()

	changed, untouched := tenant(), tenant()
	repo := newRepo(
		listed(managedscope.Organization(changed), platformcatalog.FeatureAccounting),
		listed(managedscope.Organization(untouched), platformcatalog.FeatureAccounting),
	)
	svc, _ := newServiceIn(repo, true)
	require.True(t, enabledFor(t, svc, changed, platformcatalog.FeatureAccounting))
	require.True(t, enabledFor(t, svc, untouched, platformcatalog.FeatureAccounting))

	_, err := svc.SetFeatures(
		t.Context(),
		managedscope.Organization(changed),
		&managedfeatures.Received{},
	)
	require.NoError(t, err)

	assert.False(t, enabledFor(t, svc, changed, platformcatalog.FeatureAccounting))
	assert.True(t, enabledFor(t, svc, untouched, platformcatalog.FeatureAccounting))
	assert.Equal(t, 2, repo.gets[managedscope.Organization(changed).Key()], "read again")
	assert.Equal(t, 1, repo.gets[managedscope.Organization(untouched).Key()], "still cached")
}

func TestEnabledAnywhereAsksWhetherAnyActiveOrganizationHasTheFeature(t *testing.T) {
	t.Parallel()

	org := tenant()
	repo := newRepo(listed(managedscope.Organization(org), platformcatalog.FeatureAccounting))
	svc, _ := newServiceIn(repo, true)

	anywhere := func(feature platformcatalog.FeatureKey) bool {
		t.Helper()
		on, err := svc.EnabledAnywhere(t.Context(), feature)
		require.NoError(t, err)
		return on
	}
	assert.True(t, anywhere(platformcatalog.FeatureAccounting))
	assert.False(t, anywhere(platformcatalog.FeatureAIDesk))
	assert.True(t, anywhere(platformcatalog.FeatureCoreTMS), "an active organization has base")
	assert.False(t, anywhere("tms.teleport"))

	_, err := svc.SetLifecycle(
		t.Context(),
		managedscope.Organization(org),
		managedfeatures.LifecycleSuspended,
	)
	require.NoError(t, err)
	assert.False(t, anywhere(platformcatalog.FeatureAccounting), "a write drops the answers")
	assert.False(t, anywhere(platformcatalog.FeatureCoreTMS), "no active organization is left")
}

func TestEnabledAnywhereIsCached(t *testing.T) {
	t.Parallel()

	repo := newRepo(listed(managedscope.Organization(tenant()), platformcatalog.FeatureAccounting))
	svc, _ := newServiceIn(repo, true)
	for range 3 {
		_, err := svc.EnabledAnywhere(t.Context(), platformcatalog.FeatureAccounting)
		require.NoError(t, err)
	}
	assert.Equal(t, 1, repo.anywheres)
}

func TestEnabledAnywhereOnADedicatedInstanceIsTheInstancesAnswer(t *testing.T) {
	t.Parallel()

	repo := newRepo(instanceListed(platformcatalog.FeatureAccounting))
	svc, _ := newService(repo)

	on, err := svc.EnabledAnywhere(t.Context(), platformcatalog.FeatureAccounting)
	require.NoError(t, err)
	assert.True(t, on)
	on, err = svc.EnabledAnywhere(t.Context(), platformcatalog.FeatureAIDesk)
	require.NoError(t, err)
	assert.False(t, on)
	on, err = svc.EnabledAnywhere(t.Context(), platformcatalog.FeatureCoreTMS)
	require.NoError(t, err)
	assert.True(t, on)
	assert.Zero(t, repo.anywheres, "never scans for other scopes")
}

func TestAScopeFromTheOtherTenancyIsRefused(t *testing.T) {
	t.Parallel()

	dedicated, _ := newService(newRepo())
	_, err := dedicated.SetFeatures(
		t.Context(),
		managedscope.Organization(tenant()),
		&managedfeatures.Received{},
	)
	require.ErrorIs(t, err, managedscope.ErrWrongTenancy)

	shared, _ := newServiceIn(newRepo(), true)
	_, err = shared.SetFeatures(t.Context(), managedscope.Instance(), &managedfeatures.Received{})
	require.ErrorIs(t, err, managedscope.ErrWrongTenancy)
	_, err = shared.State(t.Context(), managedscope.Instance())
	require.ErrorIs(t, err, managedscope.ErrWrongTenancy)
}

func TestADedicatedInstanceIsNeverSuspended(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	svc, _ := newService(repo)
	_, err := svc.SetLifecycle(
		t.Context(),
		managedscope.Instance(),
		managedfeatures.LifecycleSuspended,
	)
	require.ErrorIs(t, err, managedfeaturesservice.ErrInstanceNotSuspended)
	assert.Zero(t, repo.saves)
}

func TestAnUnknownLifecycleIsRefused(t *testing.T) {
	t.Parallel()

	repo := newRepo()
	svc, _ := newServiceIn(repo, true)
	_, err := svc.SetLifecycle(
		t.Context(),
		managedscope.Organization(tenant()),
		managedfeatures.Lifecycle("archived"),
	)
	require.ErrorIs(t, err, managedfeatures.ErrUnknownLifecycle)
	assert.Zero(t, repo.saves)
}
