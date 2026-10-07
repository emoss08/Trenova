package aicontrolsummaryservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aicontrolsummary"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dismissalStore struct {
	kept []*aiprovider.FailureDismissal
}

func (d *dismissalStore) List(
	context.Context,
	*repositories.ListFailureDismissalsRequest,
) ([]*aiprovider.FailureDismissal, error) {
	return d.kept, nil
}

func (d *dismissalStore) Upsert(_ context.Context, dismissal *aiprovider.FailureDismissal) error {
	d.kept = append(d.kept, dismissal)
	return nil
}

func (d *dismissalStore) Delete(context.Context, *repositories.DeleteFailureDismissalRequest) error {
	d.kept = nil
	return nil
}

func TestADismissalHoldsOnlyUntilTheProviderFailsAgain(t *testing.T) {
	t.Parallel()

	quiet := pulid.MustNew("aiprv_")
	again := pulid.MustNew("aiprv_")
	store := &dismissalStore{kept: []*aiprovider.FailureDismissal{
		{ProviderID: quiet, FailureAt: 100},
		{ProviderID: again, FailureAt: 100},
	}}
	svc := newService(newMemoryCache(), nil)
	svc.dismissals = store

	visible, err := svc.VisibleFailures(t.Context(), &services.VisibleFailuresRequest{
		Failing: []aicontrolsummary.ProviderFailure{
			{ProviderID: quiet, Name: "Quiet", LastFailureAt: 100},
			{ProviderID: again, Name: "Again", LastFailureAt: 200},
		},
	})

	require.NoError(t, err)
	require.Len(t, visible, 1)
	assert.Equal(t, "Again", visible[0].Name)
}

func TestOnlyAPersonCanDismiss(t *testing.T) {
	t.Parallel()

	svc := newService(newMemoryCache(), nil)
	svc.dismissals = &dismissalStore{}

	require.Error(t, svc.DismissFailure(t.Context(), &services.ProviderFailureDismissal{ProviderID: pulid.MustNew("aiprv_")}))
}
