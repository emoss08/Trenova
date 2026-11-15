package globalsearchservice

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/onplan"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// planEngine answers the way the permission engine does: a resource whose
// feature the plan leaves out is refused ahead of any role logic, and every
// other resource is allowed.
type planEngine struct {
	mocks.AllowAllPermissionEngine

	registry *permissioncatalog.Registry
	gate     *plantest.Gate
}

func (e *planEngine) CheckBatch(
	ctx context.Context,
	req *permissionports.BatchPermissionCheckRequest,
) (*permissionports.BatchPermissionCheckResult, error) {
	results := make([]permissionports.PermissionCheckResult, len(req.Checks))
	for idx, check := range req.Checks {
		results[idx] = permissionports.PermissionCheckResult{
			Allowed:   true,
			DataScope: permission.DataScopeOrganization,
		}
		feature, known := e.registry.FeatureOf(check.Resource)
		if !known {
			continue
		}
		if on, _ := e.gate.Enabled(ctx, feature); !on {
			denial := permissionports.NewPlanDenial(feature)
			results[idx] = permissionports.PermissionCheckResult{PlanDenial: &denial}
		}
	}

	return &permissionports.BatchPermissionCheckResult{Results: results}, nil
}

type searchedIndexes struct {
	mu      sync.Mutex
	indexes []string
}

func (s *searchedIndexes) add(index string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.indexes = append(s.indexes, index)
}

func (s *searchedIndexes) has(index string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, searched := range s.indexes {
		if searched == index {
			return true
		}
	}
	return false
}

func recordingRepo(
	t *testing.T,
	searched *searchedIndexes,
	pages map[string]*customerports.SearchPage,
) *customerportsmock.MockSearchRepository {
	t.Helper()

	repo := customerportsmock.NewMockSearchRepository(t)
	repo.EXPECT().Enabled().Return(true).Maybe()
	repo.EXPECT().
		SearchPage(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req customerports.SearchRequest) (*customerports.SearchPage, error) {
			searched.add(req.Index)
			if found, ok := pages[req.Index]; ok {
				return found, nil
			}
			return page(), nil
		}).
		Maybe()
	return repo
}

func planService(
	t *testing.T,
	registry *permissioncatalog.Registry,
	gate *plantest.Gate,
	pages map[string]*customerports.SearchPage,
	balances *fakeBalances,
) (*Service, *searchedIndexes) {
	t.Helper()

	engine := &planEngine{registry: registry, gate: gate}
	searched := &searchedIndexes{}

	return &Service{
		shipmentRead: &fakeCardReader{},
		states:       fakeStates{},
		recordLabels: &fakeLabels{},
		balances:     balances,
		views:        refview.NewPort(refview.New(registry, onplan.New(registry, gate)), engine),
		logger:       zaptest.NewLogger(t),
		config:       indexConfig(),
		searchRepo:   recordingRepo(t, searched, pages),
		permissions:  engine,
		visibility:   everyRecordVisible{},
	}, searched
}

func planRequest(entityTypes ...search.EntityType) *kernelports.GlobalSearchRequest {
	return &kernelports.GlobalSearchRequest{
		Query:       "acme",
		TenantInfo:  searchTenant(),
		Principal:   kernelports.PrincipalInfo{UserID: pulid.MustNew("usr_")},
		EntityTypes: entityTypes,
	}
}

func TestEveryKindNamesAResourceThePermissionRegistryKnows(t *testing.T) {
	t.Parallel()

	svc := &Service{config: indexConfig()}
	registry := permissioncatalog.NewRegistry()

	for _, definition := range svc.definitions() {
		require.NotEmptyf(t, definition.resource,
			"%s is searched with no permission resource, so a plan could not refuse it",
			definition.entityType)
		_, known := registry.FeatureOf(definition.resource.String())
		require.Truef(t, known,
			"%s names resource %q the permission registry has no feature for",
			definition.entityType, definition.resource)
	}
}

func TestSearchNeverSearchesAKindWhoseResourceIsOffThePlan(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	for _, definition := range (&Service{config: indexConfig()}).definitions() {
		t.Run(string(definition.entityType), func(t *testing.T) {
			t.Parallel()

			feature, known := registry.FeatureOf(definition.resource.String())
			require.True(t, known)
			svc, searched := planService(
				t,
				registry,
				plantest.Without(feature),
				map[string]*customerports.SearchPage{
					definition.index: page(map[string]any{
						"id":        pulid.MustNew("shp_").String(),
						"name":      "Acme",
						"number":    "INV-1",
						fieldStatus: "Active",
					}),
				},
				&fakeBalances{},
			)

			result, err := svc.Search(t.Context(), planRequest())
			require.NoError(t, err)

			assert.False(t, searched.has(definition.index),
				"an off-plan kind's index is not asked")
			assert.Nil(t, groupOf(result, definition.entityType),
				"an off-plan kind returns nothing")
		})
	}
}

func TestSearchOnABaseOnlyPlanKeepsEveryBaseKind(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	svc, searched := planService(t, registry, plantest.BaseOnly(), nil, &fakeBalances{})

	_, err := svc.Search(t.Context(), planRequest())
	require.NoError(t, err)

	for _, index := range []string{"shipments", "customers", "workers", "documents", "invoices"} {
		assert.Truef(t, searched.has(index), "%s is a base kind and is still searched", index)
	}
}

func customerPage() map[string]*customerports.SearchPage {
	return map[string]*customerports.SearchPage{
		"customers": page(map[string]any{
			"id":     pulid.MustNew("cus_").String(),
			"name":   "Acme Foods",
			"code":   "ACME",
			"status": "Active",
		}),
	}
}

func TestSearchCarriesNoReceivablesFigureOnACustomerHitOnABaseOnlyPlan(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	balances := &fakeBalances{balances: []*kernelports.CustomerOpenBalance{{
		CurrencyCode: "USD", OpenMinor: 125000, OverdueMinor: 5000,
	}}}
	svc, _ := planService(t, registry, plantest.BaseOnly(), customerPage(), balances)

	result, err := svc.Search(t.Context(), planRequest(search.EntityTypeCustomer))
	require.NoError(t, err)

	customers := groupOf(result, search.EntityTypeCustomer)
	require.NotNil(t, customers)
	require.Len(t, customers.Hits, 1)
	metadata := customers.Hits[0].Metadata
	assert.NotContains(t, metadata, "openBalance")
	assert.NotContains(t, metadata, "overdueBalance")
	assert.NotContains(t, metadata, "currencyCode")
	assert.Equal(t, "ACME", metadata["code"], "the customer's own values stay")
	assert.False(t, balances.called, "the receivables are not read for a plan without them")
}

func TestSearchCarriesTheReceivablesFigureOnACustomerHitWhenAccountingIsOn(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewRegistry()
	customerID := pulid.MustNew("cus_")
	balances := &fakeBalances{balances: []*kernelports.CustomerOpenBalance{{
		CustomerID: customerID, CurrencyCode: "USD", OpenMinor: 125000, OverdueMinor: 5000,
	}}}
	svc, _ := planService(t, registry, plantest.AllFeatures(), map[string]*customerports.SearchPage{
		"customers": page(map[string]any{
			"id": customerID.String(), "name": "Acme Foods", "code": "ACME", "status": "Active",
		}),
	}, balances)

	result, err := svc.Search(t.Context(), planRequest(search.EntityTypeCustomer))
	require.NoError(t, err)

	customers := groupOf(result, search.EntityTypeCustomer)
	require.NotNil(t, customers)
	require.Len(t, customers.Hits, 1)
	assert.Equal(t, "1250.00", customers.Hits[0].Metadata["openBalance"])
}

// A value of a hit's own record is dropped when the registry declares a feature of its own for
// that field and the plan leaves it out, the same as a figure the reader has no read for.
func TestSearchDropsAnOwnValueWhoseFieldBelongsToAnAddOnTheOrganizationLacks(t *testing.T) {
	t.Parallel()

	registry := permissioncatalog.NewEmptyRegistry()
	require.NoError(t, registry.Register(&permissionspec.ResourceDefinition{
		Resource:    permission.ResourceInvoice.String(),
		Feature:     platformcatalog.FeatureBilling,
		DisplayName: "Invoice",
		FieldFeatures: map[string]platformcatalog.FeatureKey{
			"disputeStatus": platformcatalog.FeatureAccounting,
		},
	}))
	pages := map[string]*customerports.SearchPage{"invoices": page(invoiceDocument(pulid.MustNew("inv_")))}

	withoutAccounting, _ := planService(
		t, registry, plantest.Without(platformcatalog.FeatureAccounting), pages, &fakeBalances{},
	)
	result, err := withoutAccounting.Search(t.Context(), planRequest(search.EntityTypeInvoice))
	require.NoError(t, err)
	invoices := groupOf(result, search.EntityTypeInvoice)
	require.NotNil(t, invoices)
	require.Len(t, invoices.Hits, 1)
	assert.NotContains(t, invoices.Hits[0].Metadata, "disputeStatus")
	assert.Equal(t, "Posted", invoices.Hits[0].Metadata[fieldStatus], "the rest of the hit stays")

	withAccounting, _ := planService(t, registry, plantest.AllFeatures(), pages, &fakeBalances{})
	result, err = withAccounting.Search(t.Context(), planRequest(search.EntityTypeInvoice))
	require.NoError(t, err)
	invoices = groupOf(result, search.EntityTypeInvoice)
	require.NotNil(t, invoices)
	require.Len(t, invoices.Hits, 1)
	assert.Contains(t, invoices.Hits[0].Metadata, "disputeStatus")
}
