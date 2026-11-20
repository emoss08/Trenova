package platformcatalogports_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type namedObserver struct {
	name    string
	feature platformcatalog.FeatureKey
}

func (o *namedObserver) ObserverFeature() platformcatalog.FeatureKey {
	return o.feature
}

type countingGate struct {
	inner platformcatalogports.FeatureGate
	asked map[platformcatalog.FeatureKey]int
	fail  map[platformcatalog.FeatureKey]error
}

func counting(inner platformcatalogports.FeatureGate) *countingGate {
	return &countingGate{
		inner: inner,
		asked: map[platformcatalog.FeatureKey]int{},
		fail:  map[platformcatalog.FeatureKey]error{},
	}
}

func (g *countingGate) Enabled(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	g.asked[feature]++
	if err := g.fail[feature]; err != nil {
		return false, err
	}
	return g.inner.Enabled(ctx, feature)
}

func (g *countingGate) EnabledAnywhere(
	ctx context.Context,
	feature platformcatalog.FeatureKey,
) (bool, error) {
	return g.Enabled(ctx, feature)
}

func names(observers []*namedObserver) []string {
	out := make([]string, 0, len(observers))
	for _, observer := range observers {
		out = append(out, observer.name)
	}
	return out
}

func observerList() []*namedObserver {
	return []*namedObserver{
		{name: "order derivation", feature: platformcatalog.FeatureCoreTMS},
		{name: "eta refresh", feature: platformcatalog.FeatureCustomerTracking},
		{name: "edi status", feature: platformcatalog.FeatureEDIIntegration},
		{name: "customer status", feature: platformcatalog.FeatureCustomerTracking},
		{name: "billing queue", feature: platformcatalog.FeatureBilling},
	}
}

func TestObservingSkipsObserversOfAnAddOnThatIsOffAndKeepsBaseOnes(t *testing.T) {
	t.Parallel()

	kept, err := platformcatalogports.Observing(t.Context(), plantest.BaseOnly(), observerList())

	require.NoError(t, err)
	assert.Equal(t, []string{"order derivation", "billing queue"}, names(kept))
}

func TestObservingAsksTheGateOncePerFeature(t *testing.T) {
	t.Parallel()

	gate := counting(plantest.AllFeatures())

	_, err := platformcatalogports.Observing(t.Context(), gate, observerList())

	require.NoError(t, err)
	assert.Equal(t, map[platformcatalog.FeatureKey]int{
		platformcatalog.FeatureCoreTMS:          1,
		platformcatalog.FeatureCustomerTracking: 1,
		platformcatalog.FeatureEDIIntegration:   1,
		platformcatalog.FeatureBilling:          1,
	}, gate.asked)
}

func TestObservingKeepsEveryObserverInOrderWithEveryFeatureOn(t *testing.T) {
	t.Parallel()

	list := observerList()
	kept, err := platformcatalogports.Observing(t.Context(), plantest.AllFeatures(), list)

	require.NoError(t, err)
	assert.Equal(t, names(list), names(kept))
}

func TestObservingSkipsTheObserversOfAFeatureItCouldNotAskAboutAndSaysWhich(t *testing.T) {
	t.Parallel()

	gate := counting(plantest.AllFeatures())
	storeDown := errors.New("plan store down")
	gate.fail[platformcatalog.FeatureCustomerTracking] = storeDown

	kept, err := platformcatalogports.Observing(t.Context(), gate, observerList())

	require.ErrorIs(t, err, storeDown)
	assert.Contains(t, err.Error(), string(platformcatalog.FeatureCustomerTracking))
	assert.Equal(t, []string{"order derivation", "edi status", "billing queue"}, names(kept))
}

func TestObservingRefusesAnObserverThatNamesAnUnknownFeature(t *testing.T) {
	t.Parallel()

	kept, err := platformcatalogports.Observing(
		t.Context(),
		plantest.AllFeatures(),
		[]*namedObserver{
			{name: "unnamed"},
			{name: "base", feature: platformcatalog.FeatureCoreTMS},
		},
	)

	require.ErrorIs(t, err, platformcatalogports.ErrUnknownFeature)
	assert.Equal(t, []string{"base"}, names(kept))
}

func TestObservingSkipsNilObservers(t *testing.T) {
	t.Parallel()

	kept, err := platformcatalogports.Observing(
		t.Context(),
		plantest.AllFeatures(),
		[]platformcatalogports.FeatureObserver{
			nil,
			&namedObserver{feature: platformcatalog.FeatureCoreTMS},
		},
	)

	require.NoError(t, err)
	assert.Len(t, kept, 1)
}
