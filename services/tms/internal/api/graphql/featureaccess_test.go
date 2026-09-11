package graphql

import (
	"context"
	"errors"
	"testing"
	"time"

	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/generated"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/core/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/zap"
)

var errAuthorizerUnavailable = errors.New("authorizer unavailable")

type recordingAuthorizer struct {
	allow    map[platformcatalog.FeatureKey]bool
	err      error
	requests []*services.AccessAuthorizeRequest
}

func (a *recordingAuthorizer) AuthorizeAccess(
	_ context.Context,
	req *services.AccessAuthorizeRequest,
) (*services.AccessAuthorizeResult, error) {
	a.requests = append(a.requests, req)
	if a.err != nil {
		return nil, a.err
	}

	return &services.AccessAuthorizeResult{
		FeatureKey: req.FeatureKey,
		Allowed:    a.allow[req.FeatureKey],
		Reason:     "not entitled",
		CheckedAt:  req.CheckedAt,
	}, nil
}

func (a *recordingAuthorizer) featureKeys() []platformcatalog.FeatureKey {
	keys := make([]platformcatalog.FeatureKey, 0, len(a.requests))
	for _, req := range a.requests {
		keys = append(keys, req.FeatureKey)
	}

	return keys
}

func newFeatureAccessTestRegistry(t *testing.T) *platformcatalog.Registry {
	t.Helper()

	registry, err := platformcatalog.NewRegistry(platformcatalog.RegistryParams{
		Providers: []platformcatalog.CatalogProvider{platformcatalog.NewStaticProvider()},
	})
	require.NoError(t, err)

	return registry
}

func newFeatureAccessExtension(
	t *testing.T,
	mode config.GraphQLAccessMode,
	authorizer services.AccessAuthorizer,
) *FeatureAccessExtension {
	t.Helper()

	return &FeatureAccessExtension{
		cfg: &config.Config{
			Platform: config.PlatformConfig{
				ControlPlane: config.PlatformControlPlaneConfig{
					Enabled:           true,
					GraphQLAccessMode: mode,
				},
			},
		},
		registry:   newFeatureAccessTestRegistry(t),
		authorizer: authorizer,
		metrics:    metrics.NewGraphQL(nil, zap.NewNop(), false, metrics.GraphQLOptions{}),
		l:          zap.NewNop(),
		now:        func() time.Time { return time.Unix(123, 0) },
	}
}

func rootField(name string, source string) *ast.Field {
	return &ast.Field{
		Name: name,
		Definition: &ast.FieldDefinition{
			Name:     name,
			Position: &ast.Position{Src: &ast.Source{Name: "../schema/" + source}},
		},
	}
}

func newOperationContext(operation ast.Operation, selections ...ast.Selection) *gqlgen.OperationContext {
	return &gqlgen.OperationContext{
		OperationName: "TestOperation",
		Operation: &ast.OperationDefinition{
			Operation:    operation,
			SelectionSet: selections,
		},
	}
}

func newAuthedContext() context.Context {
	return gqlctx.WithAuthContext(context.Background(), &authctx.AuthContext{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
	})
}

func TestFeatureAccess_SchemaSourcesAreFullyMapped(t *testing.T) {
	t.Parallel()

	registry := newFeatureAccessTestRegistry(t)
	schema := generated.NewExecutableSchema(generated.Config{}).Schema()

	sources := schemaSources(schema)
	require.NotEmpty(t, sources)

	unmapped := registry.UnclassifiedGraphQLSources(sources)
	require.Emptyf(
		t,
		unmapped,
		"every GraphQL schema source must be owned by a feature or listed as shell; unmapped: %v",
		unmapped,
	)
}

func TestFeatureAccess_ValidateAcceptsMappedSchema(t *testing.T) {
	t.Parallel()

	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, nil)
	require.NoError(t, extension.Validate(generated.NewExecutableSchema(generated.Config{})))
}

func TestFeatureAccess_EnforceDeniesUnentitledFeature(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("trainingCourses", "worker_training.graphqls")),
	)

	require.Error(t, err)
	require.Contains(t, err.Message, "trainingCourses")
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceTalent},
		authorizer.featureKeys(),
	)
}

func TestFeatureAccess_EnforceAllowsEntitledFeature(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{
		allow: map[platformcatalog.FeatureKey]bool{
			platformcatalog.FeatureWorkforceTalent: true,
		},
	}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("trainingCourses", "worker_training.graphqls")),
	)

	require.Nil(t, err)
}

func TestFeatureAccess_ObserveNeverDenies(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeObserve, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("trainingCourses", "worker_training.graphqls")),
	)

	require.Nil(t, err)
	require.NotEmpty(t, authorizer.requests)
}

func TestFeatureAccess_DisabledSkipsAuthorization(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeDisabled, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("trainingCourses", "worker_training.graphqls")),
	)

	require.Nil(t, err)
	require.Empty(t, authorizer.requests)
}

func TestFeatureAccess_ControlPlaneDisabledSkipsAuthorization(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)
	extension.cfg.Platform.ControlPlane.Enabled = false

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("trainingCourses", "worker_training.graphqls")),
	)

	require.Nil(t, err)
	require.Empty(t, authorizer.requests)
}

func TestFeatureAccess_LegacyGrantAllowsFleetTenant(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{
		allow: map[platformcatalog.FeatureKey]bool{
			platformcatalog.FeatureFleetMaintenance: true,
		},
	}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("workers", "worker.graphqls")),
	)

	require.Nil(t, err)
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{
			platformcatalog.FeatureWorkforceCore,
			platformcatalog.FeatureFleetMaintenance,
		},
		authorizer.featureKeys(),
	)
}

func TestFeatureAccess_ShellSourceSkipsAuthorization(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("notifications", "notification.graphqls")),
	)

	require.Nil(t, err)
	require.Empty(t, authorizer.requests)
}

func TestFeatureAccess_UnclassifiedFieldDeniedOnlyWhenEnforcing(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}

	observed := newFeatureAccessExtension(t, config.GraphQLAccessModeObserve, authorizer)
	require.Nil(t, observed.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("mystery", "brand_new.graphqls")),
	))

	enforced := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)
	err := enforced.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("mystery", "brand_new.graphqls")),
	)
	require.Error(t, err)
	require.Contains(t, err.Message, "mystery")
	require.Empty(t, authorizer.requests)
}

func TestFeatureAccess_AuthorizerErrorDeniesOnlyWhenEnforcing(t *testing.T) {
	t.Parallel()

	failing := &recordingAuthorizer{err: errAuthorizerUnavailable}

	observed := newFeatureAccessExtension(t, config.GraphQLAccessModeObserve, failing)
	require.Nil(t, observed.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("trainingCourses", "worker_training.graphqls")),
	))

	enforced := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, failing)
	err := enforced.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("trainingCourses", "worker_training.graphqls")),
	)
	require.Error(t, err)
	require.Contains(t, err.Message, "could not be verified")
}

func TestFeatureAccess_MissingAuthorizerDeniesOnlyWhenEnforcing(t *testing.T) {
	t.Parallel()

	observed := newFeatureAccessExtension(t, config.GraphQLAccessModeObserve, nil)
	require.Nil(t, observed.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("trainingCourses", "worker_training.graphqls")),
	))

	enforced := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, nil)
	err := enforced.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("trainingCourses", "worker_training.graphqls")),
	)
	require.Error(t, err)
	require.Contains(t, err.Message, "not configured")
}

func TestFeatureAccess_UnauthenticatedContextSkipsAuthorization(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		context.Background(),
		newOperationContext(ast.Query, rootField("trainingCourses", "worker_training.graphqls")),
	)

	require.Nil(t, err)
	require.Empty(t, authorizer.requests)
}

func TestFeatureAccess_ChecksEachFeatureOnce(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{
		allow: map[platformcatalog.FeatureKey]bool{
			platformcatalog.FeatureWorkforceTalent: true,
		},
	}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(
			ast.Query,
			rootField("trainingCourses", "worker_training.graphqls"),
			rootField("trainingRecords", "worker_training.graphqls"),
			rootField("performanceReviews", "performance_review.graphqls"),
		),
	)

	require.Nil(t, err)
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceTalent},
		authorizer.featureKeys(),
	)
}

func TestCollectRootFields(t *testing.T) {
	t.Parallel()

	fragment := &ast.FragmentDefinition{
		Name: "WorkerBits",
		SelectionSet: ast.SelectionSet{
			rootField("workers", "worker.graphqls"),
		},
	}

	opCtx := newOperationContext(
		ast.Query,
		rootField("trainingCourses", "worker_training.graphqls"),
		rootField("trainingCourses", "worker_training.graphqls"),
		&ast.Field{Name: "__typename"},
		&ast.InlineFragment{
			SelectionSet: ast.SelectionSet{
				rootField("performanceReviews", "performance_review.graphqls"),
			},
		},
		&ast.FragmentSpread{Name: "WorkerBits", Definition: fragment},
		&ast.FragmentSpread{Name: "WorkerBits", Definition: fragment},
		&ast.FragmentSpread{Name: "Dangling"},
	)

	fields := collectRootFields(opCtx)

	names := make([]string, 0, len(fields))
	for _, field := range fields {
		names = append(names, field.Name)
	}

	require.Equal(
		t,
		[]string{"trainingCourses", "trainingCourses", "performanceReviews", "workers"},
		names,
	)
}

func TestFeatureAccess_AliasedSelectOptionsCheckEveryResource(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{
		allow: map[platformcatalog.FeatureKey]bool{
			platformcatalog.FeatureWorkforceCore: true,
			platformcatalog.FeatureDispatch:      true,
		},
	}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	workers := selectOptionsField("WORKER")
	workers.Alias = "workerOptions"
	shipments := selectOptionsField("SHIPMENT")
	shipments.Alias = "shipmentOptions"

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, workers, shipments),
	)

	require.Nil(t, err)
	require.ElementsMatch(
		t,
		[]platformcatalog.FeatureKey{
			platformcatalog.FeatureWorkforceCore,
			platformcatalog.FeatureDispatch,
		},
		authorizer.featureKeys(),
	)
}

func TestCollectRootFieldsStopsFragmentCycles(t *testing.T) {
	t.Parallel()

	fragment := &ast.FragmentDefinition{Name: "Loop"}
	fragment.SelectionSet = ast.SelectionSet{
		rootField("workers", "worker.graphqls"),
		&ast.FragmentSpread{Name: "Loop", Definition: fragment},
	}

	fields := collectRootFields(
		newOperationContext(ast.Query, &ast.FragmentSpread{Name: "Loop", Definition: fragment}),
	)

	require.Len(t, fields, 1)
	require.Equal(t, "workers", fields[0].Name)
}

func TestRootFieldSource(t *testing.T) {
	t.Parallel()

	require.Equal(
		t,
		"worker_training.graphqls",
		rootFieldSource(rootField("trainingCourses", "worker_training.graphqls")),
	)
	require.Empty(t, rootFieldSource(&ast.Field{Name: "orphan"}))
	require.Empty(t, rootFieldSource(&ast.Field{
		Name:       "orphan",
		Definition: &ast.FieldDefinition{Name: "orphan"},
	}))
}

func TestGraphQLOperationName(t *testing.T) {
	t.Parallel()

	require.Equal(t, platformcatalog.GraphQLOperationQuery, graphQLOperationName(ast.Query))
	require.Equal(t, platformcatalog.GraphQLOperationMutation, graphQLOperationName(ast.Mutation))
	require.Empty(t, graphQLOperationName(ast.Subscription))
}

func selectOptionsField(resource string) *ast.Field {
	field := rootField(selectOptionsFieldName, "select_options.graphqls")
	field.Arguments = ast.ArgumentList{{
		Name: selectOptionsInputArgument,
		Value: &ast.Value{
			Kind: ast.ObjectValue,
			Children: ast.ChildValueList{{
				Name:  selectOptionsResourceField,
				Value: &ast.Value{Kind: ast.EnumValue, Raw: resource},
			}},
		},
	}}

	return field
}

func selectOptionsFieldWithVariable(variable string) *ast.Field {
	field := rootField(selectOptionsFieldName, "select_options.graphqls")
	field.Arguments = ast.ArgumentList{{
		Name:  selectOptionsInputArgument,
		Value: &ast.Value{Kind: ast.Variable, Raw: variable},
	}}

	return field
}

func TestFeatureAccess_SelectOptionsIsGatedByResource(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, selectOptionsField("SHIPMENT")),
	)

	require.Error(t, err)
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureDispatch},
		authorizer.featureKeys(),
	)
}

func TestFeatureAccess_SelectOptionsWorkerResourceUsesWorkforce(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{
		allow: map[platformcatalog.FeatureKey]bool{
			platformcatalog.FeatureWorkforceCore: true,
		},
	}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, selectOptionsField("WORKER")),
	)

	require.Nil(t, err)
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureWorkforceCore},
		authorizer.featureKeys(),
	)
}

func TestFeatureAccess_SelectOptionsShellResourceSkipsAuthorization(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, selectOptionsField("US_STATE")),
	)

	require.Nil(t, err)
	require.Empty(t, authorizer.requests)
}

func TestFeatureAccess_SelectOptionsResolvesVariableInput(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	opCtx := newOperationContext(ast.Query, selectOptionsFieldWithVariable("input"))
	opCtx.Variables = map[string]any{"input": map[string]any{"resource": "SHIPMENT"}}

	err := extension.MutateOperationContext(newAuthedContext(), opCtx)

	require.Error(t, err)
	require.Equal(
		t,
		[]platformcatalog.FeatureKey{platformcatalog.FeatureDispatch},
		authorizer.featureKeys(),
	)
}

func TestFeatureAccess_SelectOptionsUnresolvableResourceIsDenied(t *testing.T) {
	t.Parallel()

	authorizer := &recordingAuthorizer{allow: map[platformcatalog.FeatureKey]bool{}}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField(selectOptionsFieldName, "select_options.graphqls")),
	)

	require.Error(t, err)
	require.Empty(t, authorizer.requests)
}

func TestFeatureAccess_SelectOptionResourceMappingIsComplete(t *testing.T) {
	t.Parallel()

	schema := generated.NewExecutableSchema(generated.Config{}).Schema()
	definition, ok := schema.Types["SelectOptionResource"]
	require.True(t, ok)
	require.NotEmpty(t, definition.EnumValues)

	for _, value := range definition.EnumValues {
		t.Run(value.Name, func(t *testing.T) {
			policy := platformcatalog.PolicyForSelectOptionResource(value.Name)
			require.NotEqualf(
				t,
				platformcatalog.RouteAccessClassUnclassified,
				policy.AccessClass,
				"SelectOptionResource %q has no feature mapping", value.Name,
			)
		})
	}
}

type scriptedAuthorizer struct {
	results  map[platformcatalog.FeatureKey]*services.AccessAuthorizeResult
	errs     map[platformcatalog.FeatureKey]error
	requests []*services.AccessAuthorizeRequest
}

func (a *scriptedAuthorizer) AuthorizeAccess(
	_ context.Context,
	req *services.AccessAuthorizeRequest,
) (*services.AccessAuthorizeResult, error) {
	a.requests = append(a.requests, req)
	if err, ok := a.errs[req.FeatureKey]; ok {
		return nil, err
	}
	if result, ok := a.results[req.FeatureKey]; ok {
		return result, nil
	}

	return &services.AccessAuthorizeResult{FeatureKey: req.FeatureKey}, nil
}

func TestFeatureAccess_DenialWinsOverLaterCheckError(t *testing.T) {
	t.Parallel()

	authorizer := &scriptedAuthorizer{
		results: map[platformcatalog.FeatureKey]*services.AccessAuthorizeResult{
			platformcatalog.FeatureWorkforceCore: {
				FeatureKey: platformcatalog.FeatureWorkforceCore,
				Allowed:    false,
				Reason:     "workforce not licensed",
			},
		},
		errs: map[platformcatalog.FeatureKey]error{
			platformcatalog.FeatureFleetMaintenance: errAuthorizerUnavailable,
		},
	}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("workers", "worker.graphqls")),
	)

	require.Error(t, err)
	require.Contains(t, err.Message, "workforce not licensed")
	require.NotContains(t, err.Message, "could not be verified")
}

func TestFeatureAccess_LegacyGrantAllowsAfterPrimaryCheckError(t *testing.T) {
	t.Parallel()

	authorizer := &scriptedAuthorizer{
		results: map[platformcatalog.FeatureKey]*services.AccessAuthorizeResult{
			platformcatalog.FeatureFleetMaintenance: {
				FeatureKey: platformcatalog.FeatureFleetMaintenance,
				Allowed:    true,
			},
		},
		errs: map[platformcatalog.FeatureKey]error{
			platformcatalog.FeatureWorkforceCore: errAuthorizerUnavailable,
		},
	}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("workers", "worker.graphqls")),
	)

	require.Nil(t, err)
	require.Len(t, authorizer.requests, 2)
}

func TestFeatureAccess_AllChecksErroringReportsVerificationFailure(t *testing.T) {
	t.Parallel()

	authorizer := &scriptedAuthorizer{
		errs: map[platformcatalog.FeatureKey]error{
			platformcatalog.FeatureWorkforceCore:    errAuthorizerUnavailable,
			platformcatalog.FeatureFleetMaintenance: errAuthorizerUnavailable,
		},
	}
	extension := newFeatureAccessExtension(t, config.GraphQLAccessModeEnforce, authorizer)

	err := extension.MutateOperationContext(
		newAuthedContext(),
		newOperationContext(ast.Query, rootField("workers", "worker.graphqls")),
	)

	require.Error(t, err)
	require.Contains(t, err.Message, "could not be verified")
}

func TestFeatureAccess_AdministrationSourcesAreSeparateFromCoreTMS(t *testing.T) {
	t.Parallel()

	registry := newFeatureAccessTestRegistry(t)

	for _, source := range []platformcatalog.GraphQLSource{
		"user.graphqls",
		"role.graphqls",
		"tenant.graphqls",
		"audit_log.graphqls",
		"table_configuration.graphqls",
	} {
		policy := registry.PolicyForGraphQLRootField(
			platformcatalog.GraphQLOperationQuery,
			"anything",
			source,
		)
		require.Equal(t, platformcatalog.FeatureAdministration, policy.FeatureKey)
	}
}
