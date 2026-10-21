package globalsearchservice

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type viewEngine struct {
	mocks.AllowAllPermissionEngine

	denied  []permission.Resource
	ceiling permission.FieldSensitivity
}

func (e *viewEngine) allowed(resource string) bool {
	return !slices.Contains(e.denied, permission.Resource(resource))
}

func (e *viewEngine) Check(
	_ context.Context,
	req *permissionports.PermissionCheckRequest,
) (*permissionports.PermissionCheckResult, error) {
	return &permissionports.PermissionCheckResult{Allowed: e.allowed(req.Resource)}, nil
}

func (e *viewEngine) CheckBatch(
	_ context.Context,
	req *permissionports.BatchPermissionCheckRequest,
) (*permissionports.BatchPermissionCheckResult, error) {
	results := make([]permissionports.PermissionCheckResult, len(req.Checks))
	for idx, check := range req.Checks {
		results[idx] = permissionports.PermissionCheckResult{
			Allowed:   e.allowed(check.Resource),
			DataScope: permission.DataScopeOrganization,
		}
	}
	return &permissionports.BatchPermissionCheckResult{Results: results}, nil
}

func (e *viewEngine) GetResourcePermissions(
	_ context.Context,
	_, _ pulid.ID,
	resource string,
) (*permissionports.ResourcePermissionDetail, error) {
	if e.ceiling == "" {
		return nil, nil
	}
	return &permissionports.ResourcePermissionDetail{
		Resource:       resource,
		MaxSensitivity: e.ceiling,
	}, nil
}

func fill(t *testing.T, value any) {
	t.Helper()

	target := reflect.ValueOf(value).Elem()
	for idx := range target.NumField() {
		field := target.Field(idx)
		if !field.CanSet() || !field.IsZero() {
			continue
		}
		switch field.Kind() { //nolint:exhaustive // only the kinds a card or balance holds
		case reflect.String:
			field.SetString("v" + target.Type().Field(idx).Name)
		case reflect.Int, reflect.Int64:
			field.SetInt(1)
		}
	}
}

func filledCard(t *testing.T, id pulid.ID) *shipmentports.ShipmentSearchCard {
	t.Helper()

	card := &shipmentports.ShipmentSearchCard{ShipmentID: id}
	fill(t, card)
	return card
}

func documentFrom(attributes []string, id string, overrides map[string]any) map[string]any {
	document := make(map[string]any, len(attributes))
	for _, attribute := range attributes {
		document[attribute] = "v" + attribute
	}
	document["id"] = id
	for key, value := range overrides {
		document[key] = value
	}
	return document
}

func pagesRepo(
	t *testing.T,
	pages map[string]*customerports.SearchPage,
) *customerportsmock.MockSearchRepository {
	t.Helper()

	repo := customerportsmock.NewMockSearchRepository(t)
	repo.EXPECT().Enabled().Return(true).Maybe()
	repo.EXPECT().
		SearchPage(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req customerports.SearchRequest) (*customerports.SearchPage, error) {
			if found, ok := pages[req.Index]; ok {
				return found, nil
			}
			return page(), nil
		})
	return repo
}

func TestEveryHitMetadataKeyIsClassified(t *testing.T) {
	shipmentID := pulid.MustNew("shp_")
	workerLoad := pulid.MustNew("shp_")
	customerID := pulid.MustNew("cus_")
	workerID := pulid.MustNew("wrk_")

	balance := &kernelports.CustomerOpenBalance{CustomerID: customerID}
	fill(t, balance)

	svc := &Service{logger: zaptest.NewLogger(t), config: indexConfig()}
	attributes := make(map[search.EntityType][]string, 4)
	for _, definition := range svc.definitions() {
		attributes[definition.entityType] = definition.attributes
	}
	svc.searchRepo = pagesRepo(t, map[string]*customerports.SearchPage{
		"shipments": page(documentFrom(attributes[search.EntityTypeShipment], shipmentID.String(), nil)),
		"customers": page(documentFrom(attributes[search.EntityTypeCustomer], customerID.String(), map[string]any{
			"state_id": pulid.MustNew("ust_").String(),
		})),
		"workers": page(documentFrom(attributes[search.EntityTypeWorker], workerID.String(), map[string]any{
			"state_id": pulid.MustNew("ust_").String(),
		})),
		"documents": page(documentFrom(attributes[search.EntityTypeDocument], "doc_1", map[string]any{
			"resource_type": "shipment",
			"resource_id":   shipmentID.String(),
		})),
	})
	svc.shipmentRead = &fakeCardReader{
		cards: map[pulid.ID]*shipmentports.ShipmentSearchCard{
			shipmentID: filledCard(t, shipmentID),
			workerLoad: filledCard(t, workerLoad),
		},
		links: []*shipmentports.ShipmentPartyLink{
			{ShipmentID: shipmentID, CustomerID: customerID},
			{ShipmentID: workerLoad, WorkerID: workerID},
		},
	}
	svc.permissions = &viewEngine{}
	svc.visibility = everyRecordVisible{}
	svc.balances = &fakeBalances{balances: []*kernelports.CustomerOpenBalance{balance}}
	svc.recordLabels = &fakeLabels{labels: map[string][]*captureports.CaptureRecordLabel{
		"shipment": {{ID: shipmentID, Title: "S-1"}},
	}}

	req := &kernelports.GlobalSearchRequest{Query: "v", TenantInfo: searchTenant()}
	access, err := svc.resolveAccess(t.Context(), req)
	require.NoError(t, err)
	results := svc.searchIndexes(t.Context(), req, access.definitions, req.Query, maxSearchLimit)
	enricher := &hitEnricher{
		service: svc,
		req:     req,
		access:  access,
		limit:   maxSearchLimit,
		results: results,
	}
	enricher.enrich(t.Context())

	seen := make(map[search.EntityType]int, len(results))
	for _, result := range results {
		sources := hitMetadata[result.definition.entityType]
		for _, hit := range result.hits {
			for key := range hit.Metadata {
				seen[result.definition.entityType]++
				_, ok := sources[key]
				require.Truef(t, ok,
					"%s hits carry metadata %q with no classification: add it to hitMetadata "+
						"with the kind and fields it comes from",
					result.definition.entityType, key)
			}
		}
	}
	for _, entityType := range []search.EntityType{
		search.EntityTypeShipment,
		search.EntityTypeCustomer,
		search.EntityTypeWorker,
		search.EntityTypeDocument,
	} {
		require.Positivef(t, seen[entityType], "no %s metadata was produced", entityType)
	}
}

func TestEveryClassifiedFieldIsInItsKindsRegistry(t *testing.T) {
	registry := permission.NewRegistry()
	for entityType, sources := range hitMetadata {
		for key, source := range sources {
			if source.kind == "" {
				continue
			}
			_, ok := registry.Get(source.kind.String())
			require.Truef(t, ok, "%s %q names kind %q, which is not registered", entityType, key, source.kind)
			require.Truef(t, source.figure || len(source.fields) > 0,
				"%s %q names kind %q but no field of it", entityType, key, source.kind)
		}
	}
}

func shipmentSearchWithViews(
	t *testing.T,
	engine *viewEngine,
) map[string]string {
	t.Helper()

	shipmentID := pulid.MustNew("shp_")
	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: indexConfig(),
		searchRepo: pagesRepo(t, map[string]*customerports.SearchPage{
			"shipments": page(map[string]any{"id": shipmentID.String(), "pro_number": "S-240193"}),
		}),
		shipmentRead: &fakeCardReader{cards: map[pulid.ID]*shipmentports.ShipmentSearchCard{
			shipmentID: {
				ShipmentID:       shipmentID,
				ProNumber:        "S-240193",
				CustomerID:       pulid.MustNew("cus_"),
				CustomerName:     "Acme Foods",
				CustomerCode:     "ACME",
				ServiceTypeCode:  "EXP",
				OriginCity:       "Dallas",
				OriginState:      "TX",
				DestinationCity:  "Atlanta",
				DestinationState: "GA",
				DriverID:         pulid.MustNew("wrk_"),
				DriverName:       "Leon Park",
			},
		}},
		permissions: engine,
		visibility:  everyRecordVisible{},
		views:       refview.NewPort(refview.New(permission.NewRegistry()), engine),
	}

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       "S-240193",
		TenantInfo:  searchTenant(),
		Principal:   kernelports.PrincipalInfo{UserID: pulid.MustNew("usr_")},
		EntityTypes: []search.EntityType{search.EntityTypeShipment},
	})
	require.NoError(t, err)
	shipments := groupOf(result, search.EntityTypeShipment)
	require.NotNil(t, shipments)
	require.Len(t, shipments.Hits, 1)

	return shipments.Hits[0].Metadata
}

func TestSearchShowsAShipmentsLinkedRecordsThroughTheirReferenceViews(t *testing.T) {
	metadata := shipmentSearchWithViews(t, &viewEngine{denied: []permission.Resource{
		permission.ResourceCustomer,
		permission.ResourceWorker,
		permission.ResourceLocation,
		permission.ResourceServiceType,
	}})

	require.Equal(t, "Acme Foods", metadata["customerName"])
	require.Equal(t, "ACME", metadata["customerCode"])
	require.NotEmpty(t, metadata["customerId"])
	require.Equal(t, "Dallas, TX", metadata["origin"])
	require.Equal(t, "Atlanta, GA", metadata["destination"])
	require.Equal(t, "Leon Park", metadata["driverName"])
	require.NotEmpty(t, metadata["driverId"])
	require.NotContains(t, metadata, "serviceTypeCode",
		"a service type has no reference view, so its code needs service type read")
}

func TestSearchShowsAServiceTypeCodeToItsReaders(t *testing.T) {
	metadata := shipmentSearchWithViews(t, &viewEngine{denied: []permission.Resource{
		permission.ResourceCustomer,
	}})

	require.Equal(t, "EXP", metadata["serviceTypeCode"])
	require.Equal(t, "Acme Foods", metadata["customerName"])
}

func TestSearchHoldsAViewFieldToThePersonsCeiling(t *testing.T) {
	metadata := shipmentSearchWithViews(t, &viewEngine{
		denied:  []permission.Resource{permission.ResourceCustomer},
		ceiling: permission.SensitivityPublic,
	})

	require.NotContains(t, metadata, "customerName")
	require.NotContains(t, metadata, "customerCode")
	require.NotEmpty(t, metadata["customerId"], "an id carries nothing about the record")
	require.Equal(t, "Leon Park", metadata["driverName"], "the ceiling applies only without the kind's read")
}

func TestSearchDropsMetadataWithNoClassification(t *testing.T) {
	hit := &kernelports.GlobalSearchHit{
		ID:         pulid.MustNew("shp_").String(),
		EntityType: search.EntityTypeShipment,
		Metadata:   map[string]string{"proNumber": "S-1", "customerCreditLimit": "50000"},
	}
	enricher := &hitEnricher{
		access: &searchAccess{linked: readsOnly{check: func(context.Context, permission.Resource) (bool, error) {
			return true, nil
		}}},
		results: []*indexResult{{
			definition: entitySearchDefinition{entityType: search.EntityTypeShipment},
			hits:       []*kernelports.GlobalSearchHit{hit},
		}},
	}

	enricher.withholdUnshown(t.Context())

	require.Equal(t, map[string]string{"proNumber": "S-1"}, hit.Metadata)
}
