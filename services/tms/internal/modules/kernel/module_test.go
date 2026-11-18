package kernel

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/app/entitlementservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type noFeatureList struct{}

func (noFeatureList) Get(context.Context) (*managedfeatures.State, error) {
	return nil, managedfeatures.ErrNoState
}

func (noFeatureList) Replace(
	context.Context,
	*managedfeaturesports.ReplaceRequest,
) (*managedfeatures.Replacement, error) {
	return &managedfeatures.Replacement{}, nil
}

func (noFeatureList) RemindPlanEnd(context.Context, int64) (*managedfeatures.Replacement, error) {
	return &managedfeatures.Replacement{}, nil
}

func (noFeatureList) History(
	context.Context,
	int,
) (*managedfeatures.State, []*managedfeatures.Change, error) {
	return nil, nil, nil
}

func newManagedFeatures() *managedfeaturesservice.Service {
	return managedfeaturesservice.New(managedfeaturesservice.Params{
		Repo:   noFeatureList{},
		Clock:  func() int64 { return 0 },
		Logger: zap.NewNop(),
	})
}

func TestFeatureGateIsTheManagedListInManagedMode(t *testing.T) {
	t.Parallel()

	managed := newManagedFeatures()
	gate := featureGate(&managedconfig.Config{Enabled: true}, managed)
	assert.Same(t, managed, gate)

	on, err := gate.Enabled(t.Context(), platformcatalog.FeatureAccounting)
	require.NoError(t, err)
	assert.False(t, on, "no list yet: add-ons are off")
}

func TestFeatureGateAllowsEverythingWhenSelfHosted(t *testing.T) {
	t.Parallel()

	gate := featureGate(&managedconfig.Config{}, newManagedFeatures())
	assert.IsType(t, &entitlementservice.AllowAll{}, gate)

	on, err := gate.Enabled(t.Context(), platformcatalog.FeatureAccounting)
	require.NoError(t, err)
	assert.True(t, on)
}
