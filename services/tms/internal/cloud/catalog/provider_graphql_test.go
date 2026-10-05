package catalog

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/stretchr/testify/require"
)

func TestRegistry_PolicyForGraphQLRootField(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	tests := []struct {
		name        string
		operation   string
		field       string
		source      platformcatalog.GraphQLSource
		wantClass   platformcatalog.RouteAccessClass
		wantFeature platformcatalog.FeatureKey
	}{
		{
			name:        "workforce source",
			operation:   platformcatalog.GraphQLOperationQuery,
			field:       "trainingCourses",
			source:      "worker_training.graphqls",
			wantClass:   platformcatalog.RouteAccessClassProduct,
			wantFeature: platformcatalog.FeatureWorkforceTalent,
		},
		{
			name:        "workforce mutation",
			operation:   platformcatalog.GraphQLOperationMutation,
			field:       "createPtoPolicy",
			source:      "pto_policy.graphqls",
			wantClass:   platformcatalog.RouteAccessClassProduct,
			wantFeature: platformcatalog.FeatureWorkforceTimeOff,
		},
		{
			name:        "root field override wins over source",
			operation:   platformcatalog.GraphQLOperationQuery,
			field:       "settlementDisputes",
			source:      "driver_portal.graphqls",
			wantClass:   platformcatalog.RouteAccessClassProduct,
			wantFeature: platformcatalog.FeatureSettlement,
		},
		{
			name:        "self service override wins over source",
			operation:   platformcatalog.GraphQLOperationMutation,
			field:       "inviteWorkerToPortal",
			source:      "driver_portal.graphqls",
			wantClass:   platformcatalog.RouteAccessClassProduct,
			wantFeature: platformcatalog.FeatureWorkforceSelfService,
		},
		{
			name:        "unoverridden driver portal field",
			operation:   platformcatalog.GraphQLOperationQuery,
			field:       "myLoads",
			source:      "driver_portal.graphqls",
			wantClass:   platformcatalog.RouteAccessClassProduct,
			wantFeature: platformcatalog.FeatureDriverPortal,
		},
		{
			name:      "shell source",
			operation: platformcatalog.GraphQLOperationQuery,
			field:     "selectOptions",
			source:    "select_options.graphqls",
			wantClass: platformcatalog.RouteAccessClassAccountShell,
		},
		{
			name:      "unknown source",
			operation: platformcatalog.GraphQLOperationQuery,
			field:     "somethingNew",
			source:    "brand_new.graphqls",
			wantClass: platformcatalog.RouteAccessClassUnclassified,
		},
		{
			name:      "missing source",
			operation: platformcatalog.GraphQLOperationQuery,
			field:     "somethingNew",
			source:    "",
			wantClass: platformcatalog.RouteAccessClassUnclassified,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			policy := registry.PolicyForGraphQLRootField(tt.operation, tt.field, tt.source)
			require.Equal(t, tt.wantClass, policy.AccessClass)
			require.Equal(t, tt.wantFeature, policy.FeatureKey)
		})
	}
}

func TestRegistry_UnclassifiedGraphQLSources(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	unclassified := registry.UnclassifiedGraphQLSources([]platformcatalog.GraphQLSource{
		"worker.graphqls",
		"select_options.graphqls",
		"brand_new.graphqls",
		"brand_new.graphqls",
		"another_new.graphqls",
	})

	require.Equal(
		t,
		[]platformcatalog.GraphQLSource{"another_new.graphqls", "brand_new.graphqls"},
		unclassified,
	)
}

func TestStaticProvider_GraphQLOwnershipIsUnique(t *testing.T) {
	t.Parallel()

	sources := make(map[platformcatalog.GraphQLSource]platformcatalog.FeatureKey)
	fields := make(map[string]platformcatalog.FeatureKey)

	for _, feature := range NewStaticProvider().Features() {
		for _, source := range feature.GraphQLSources {
			owner, exists := sources[source]
			require.Falsef(t, exists, "source %q owned by %q and %q", source, owner, feature.Key)
			sources[source] = feature.Key

			require.Falsef(
				t,
				graphQLSourceIsShell(source),
				"source %q is both feature-owned and shell",
				source,
			)
		}
		for _, field := range feature.GraphQLRootFields {
			owner, exists := fields[field.Key()]
			require.Falsef(
				t,
				exists,
				"field %q owned by %q and %q",
				field.Key(),
				owner,
				feature.Key,
			)
			fields[field.Key()] = feature.Key
		}
	}

	require.NotEmpty(t, sources)
	require.NotEmpty(t, fields)
}

func TestStaticProvider_GraphQLOwnersReferenceRealFeatures(t *testing.T) {
	t.Parallel()

	registry := newTestRegistry(t)

	for featureKey := range graphQLSourceOwners {
		_, ok := registry.GetFeature(featureKey)
		require.Truef(t, ok, "graphQLSourceOwners references unknown feature %q", featureKey)
	}
	for featureKey := range graphQLRootFieldOwners {
		_, ok := registry.GetFeature(featureKey)
		require.Truef(t, ok, "graphQLRootFieldOwners references unknown feature %q", featureKey)
	}
}

func TestRegistry_RejectsFeatureClaimingShellSource(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		graphQLSources:    make(map[platformcatalog.GraphQLSource]platformcatalog.FeatureKey),
		graphQLRootFields: make(map[string]platformcatalog.FeatureKey),
	}

	err := registry.registerGraphQLOwnership(&platformcatalog.Feature{
		Key:            platformcatalog.FeatureCoreTMS,
		GraphQLSources: []platformcatalog.GraphQLSource{"select_options.graphqls"},
	})
	require.ErrorContains(t, err, "claims GraphQL shell source")
}

func TestRegistry_RejectsDuplicateGraphQLSource(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		graphQLSources:    make(map[platformcatalog.GraphQLSource]platformcatalog.FeatureKey),
		graphQLRootFields: make(map[string]platformcatalog.FeatureKey),
	}

	require.NoError(t, registry.registerGraphQLOwnership(&platformcatalog.Feature{
		Key:            platformcatalog.FeatureCoreTMS,
		GraphQLSources: []platformcatalog.GraphQLSource{"shared_thing.graphqls"},
	}))

	err := registry.registerGraphQLOwnership(&platformcatalog.Feature{
		Key:            platformcatalog.FeatureBilling,
		GraphQLSources: []platformcatalog.GraphQLSource{"shared_thing.graphqls"},
	})
	require.ErrorContains(t, err, "assigned to both feature")
}

func TestRegistry_RejectsInvalidRootFieldOperation(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		graphQLSources:    make(map[platformcatalog.GraphQLSource]platformcatalog.FeatureKey),
		graphQLRootFields: make(map[string]platformcatalog.FeatureKey),
	}

	err := registry.registerGraphQLOwnership(&platformcatalog.Feature{
		Key:               platformcatalog.FeatureCoreTMS,
		GraphQLRootFields: []platformcatalog.GraphQLRootField{{Operation: "Subscription", Field: "thing"}},
	})
	require.ErrorContains(t, err, "invalid operation")
}
