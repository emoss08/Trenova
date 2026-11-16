package builtinembeddingservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/agent/app/builtinembeddingservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var service = managedconfig.Embedding{
	URL:   "http://trenova-embedding:11434",
	Model: "nomic-embed-text:v1.5",
}

type failingGate struct{}

func (failingGate) Enabled(context.Context, platformcatalog.FeatureKey) (bool, error) {
	return false, errors.New("the feature state could not be read")
}

func newService(
	config *managedconfig.Config,
	gate platformcatalogports.FeatureGate,
) *builtinembeddingservice.Service {
	return builtinembeddingservice.New(builtinembeddingservice.Params{
		Managed:  config,
		Features: gate,
	})
}

func TestProvider_ManagedWithTrenovaAIAndTheServiceNamed(t *testing.T) {
	t.Parallel()

	tenant := plantest.Tenant()
	svc := newService(
		&managedconfig.Config{Enabled: true, Embedding: service},
		plantest.AllFeatures(),
	)

	provider, ok, err := svc.Provider(t.Context(), tenant)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, aiprovider.BuiltInEmbeddingID, provider.ID)
	assert.Equal(t, tenant.OrgID, provider.OrganizationID)
	assert.Equal(t, tenant.BuID, provider.BusinessUnitID)
	assert.Equal(t, "nomic-embed-text:v1.5", provider.Model)
	assert.Equal(t, "http://trenova-embedding:11434/v1", provider.ResolvedBaseURL())
	assert.True(t, provider.BuiltIn)
}

func TestProvider_NoneWhen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *managedconfig.Config
		gate   platformcatalogports.FeatureGate
	}{
		{
			name:   "the install is self-hosted, even with the service named",
			config: &managedconfig.Config{Embedding: service},
			gate:   plantest.AllFeatures(),
		},
		{
			name:   "the instance is managed but runs no embedding service",
			config: &managedconfig.Config{Enabled: true},
			gate:   plantest.AllFeatures(),
		},
		{
			name:   "the plan does not include Trenova AI",
			config: &managedconfig.Config{Enabled: true, Embedding: service},
			gate:   plantest.Without(platformcatalog.FeatureAIDesk),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider, ok, err := newService(
				tt.config,
				tt.gate,
			).Provider(t.Context(), plantest.Tenant())
			require.NoError(t, err)
			assert.False(t, ok)
			assert.Nil(t, provider)
		})
	}
}

func TestProvider_AFeatureGateFailureIsReturned(t *testing.T) {
	t.Parallel()

	svc := newService(&managedconfig.Config{Enabled: true, Embedding: service}, failingGate{})

	_, ok, err := svc.Provider(t.Context(), plantest.Tenant())
	require.Error(t, err)
	assert.False(t, ok)
}

func TestProvider_SelfHostedNeverReadsThePlan(t *testing.T) {
	t.Parallel()

	svc := newService(&managedconfig.Config{Embedding: service}, failingGate{})

	_, ok, err := svc.Provider(t.Context(), plantest.Tenant())
	require.NoError(t, err)
	assert.False(t, ok)
}
