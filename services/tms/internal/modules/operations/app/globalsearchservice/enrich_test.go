package globalsearchservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/searchconfig"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type fakeCardReader struct {
	cards       map[pulid.ID]*shipmentports.ShipmentSearchCard
	links       []*shipmentports.ShipmentPartyLink
	linkRequest *shipmentports.ListShipmentPartyLinksRequest
}

func (f *fakeCardReader) ListSearchCards(
	_ context.Context,
	req *shipmentports.ListShipmentSearchCardsRequest,
) ([]*shipmentports.ShipmentSearchCard, error) {
	cards := make([]*shipmentports.ShipmentSearchCard, 0, len(req.ShipmentIDs))
	for _, id := range req.ShipmentIDs {
		if card, ok := f.cards[id]; ok {
			cards = append(cards, card)
		}
	}
	return cards, nil
}

func (f *fakeCardReader) ListPartyLinks(
	_ context.Context,
	req *shipmentports.ListShipmentPartyLinksRequest,
) ([]*shipmentports.ShipmentPartyLink, error) {
	f.linkRequest = req
	return f.links, nil
}

type fakeBalances struct {
	balances []*kernelports.CustomerOpenBalance
	called   bool
}

func (f *fakeBalances) ListCustomerOpenBalances(
	context.Context,
	*kernelports.ListCustomerOpenBalancesRequest,
) ([]*kernelports.CustomerOpenBalance, error) {
	f.called = true
	return f.balances, nil
}

type fakeLabels struct {
	labels map[string][]*captureports.CaptureRecordLabel
	asked  []string
}

func (f *fakeLabels) Exists(
	context.Context,
	pagination.TenantInfo,
	string,
	pulid.ID,
) (bool, error) {
	return true, nil
}

func (f *fakeLabels) Labels(
	_ context.Context,
	req *captureports.ListCaptureRecordLabelsRequest,
) ([]*captureports.CaptureRecordLabel, error) {
	f.asked = append(f.asked, req.ResourceType)
	return f.labels[req.ResourceType], nil
}

func permissionsDenying(
	t *testing.T,
	denied ...permission.Resource,
) *permissionportsmock.MockPermissionEngine {
	t.Helper()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.EXPECT().
		CheckBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *permissionports.BatchPermissionCheckRequest,
		) (*permissionports.BatchPermissionCheckResult, error) {
			results := make([]permissionports.PermissionCheckResult, len(req.Checks))
			for idx, check := range req.Checks {
				allowed := true
				for _, resource := range denied {
					if check.Resource == resource.String() {
						allowed = false
					}
				}
				results[idx] = permissionports.PermissionCheckResult{Allowed: allowed}
			}
			return &permissionports.BatchPermissionCheckResult{Results: results}, nil
		})
	return engine
}

func indexConfig() *searchconfig.Config {
	return &searchconfig.Config{
		Enabled: true,
		Meilisearch: searchconfig.MeilisearchConfig{
			Indexes: searchconfig.MeilisearchIndexConfig{
				Shipments: "shipments",
				Customers: "customers",
				Workers:   "workers",
				Documents: "documents",
			},
		},
	}
}

func repoReturning(
	t *testing.T,
	pages map[string]*customerports.SearchPage,
) *customerportsmock.MockSearchRepository {
	t.Helper()

	repo := customerportsmock.NewMockSearchRepository(t)
	repo.EXPECT().Enabled().Return(true).Once()
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

func groupOf(
	result *kernelports.GlobalSearchResult,
	entityType search.EntityType,
) *kernelports.GlobalSearchGroup {
	for _, group := range result.Groups {
		if group.EntityType == entityType {
			return group
		}
	}
	return nil
}

func TestSearchFindsACustomersShipmentsAndAWorkersCurrentLoad(t *testing.T) {
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	customerID := pulid.MustNew("cus_")
	workerID := pulid.MustNew("wrk_")
	customerLoad := pulid.MustNew("shp_")
	workerLoad := pulid.MustNew("shp_")

	reader := &fakeCardReader{
		cards: map[pulid.ID]*shipmentports.ShipmentSearchCard{
			customerLoad: {
				ShipmentID:       customerLoad,
				ProNumber:        "S-240193",
				BOL:              "88213",
				Status:           shipmentvocab.StatusInTransit,
				CustomerID:       customerID,
				CustomerName:     "Acme Foods",
				CustomerCode:     "ACME",
				OriginCity:       "Dallas",
				OriginState:      "TX",
				DestinationCity:  "Atlanta",
				DestinationState: "GA",
				DriverID:         workerID,
				DriverName:       "Leon Park",
			},
			workerLoad: {
				ShipmentID: workerLoad,
				ProNumber:  "S-240201",
				Status:     shipmentvocab.StatusDelayed,
			},
		},
		links: []*shipmentports.ShipmentPartyLink{
			{ShipmentID: customerLoad, CustomerID: customerID},
			{ShipmentID: workerLoad, WorkerID: workerID},
		},
	}

	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: indexConfig(),
		searchRepo: repoReturning(t, map[string]*customerports.SearchPage{
			"customers": page(
				map[string]any{"id": customerID.String(), "name": "Acme Foods", "code": "ACME"},
			),
			"workers": page(map[string]any{
				"id":          workerID.String(),
				"first_name":  "Leon",
				"last_name":   "Park",
				"external_id": "W-1063",
			}),
		}),
		shipmentRead: reader,
		permissions:  allowAllPermissionsMock(t),
		visibility:   everyRecordVisible{},
	}

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:      "acme",
		TenantInfo: tenant,
		Limit:      5,
	})
	require.NoError(t, err)

	require.Equal(t, []pulid.ID{customerID}, reader.linkRequest.CustomerIDs)
	require.Equal(t, 5, reader.linkRequest.CustomerLimit)
	require.Equal(t, []pulid.ID{workerID}, reader.linkRequest.WorkerIDs)

	shipments := groupOf(result, search.EntityTypeShipment)
	require.NotNil(t, shipments)
	require.Len(t, shipments.Hits, 2)
	require.Equal(t, 2, shipments.Total)
	require.Equal(t, "S-240193", shipments.Hits[0].Title)
	require.Equal(t, map[string]string{
		"status":       "InTransit",
		"proNumber":    "S-240193",
		"bol":          "88213",
		"customerName": "Acme Foods",
		"customerCode": "ACME",
		"customerId":   customerID.String(),
		"origin":       "Dallas, TX",
		"destination":  "Atlanta, GA",
		"driverName":   "Leon Park",
		"driverId":     workerID.String(),
	}, shipments.Hits[0].Metadata)
	require.Equal(t, "S-240201", shipments.Hits[1].Title)

	workers := groupOf(result, search.EntityTypeWorker)
	require.NotNil(t, workers)
	require.Equal(t, "S-240201", workers.Hits[0].Metadata["currentLoad"])
	require.Equal(t, workerLoad.String(), workers.Hits[0].Metadata["currentLoadId"])
	require.Equal(t, "Delayed", workers.Hits[0].Metadata["currentLoadStatus"])
	require.Equal(t, "W-1063", workers.Hits[0].Metadata["externalId"])
}

func TestSearchNamesNoShipmentWithoutShipmentAccess(t *testing.T) {
	workerID := pulid.MustNew("wrk_")
	reader := &fakeCardReader{}

	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: indexConfig(),
		searchRepo: repoReturning(t, map[string]*customerports.SearchPage{
			"workers": page(
				map[string]any{"id": workerID.String(), "first_name": "Leon", "last_name": "Park"},
			),
		}),
		shipmentRead: reader,
		permissions:  permissionsDenying(t, permission.ResourceShipment),
		visibility:   everyRecordVisible{},
	}

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:      "leon",
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})
	require.NoError(t, err)
	require.Nil(t, reader.linkRequest)
	require.Nil(t, groupOf(result, search.EntityTypeShipment))
	workers := groupOf(result, search.EntityTypeWorker)
	require.NotContains(t, workers.Hits[0].Metadata, "currentLoad")
}

func TestSearchShowsOpenBalancesOnlyWithReceivablesAccess(t *testing.T) {
	customerID := pulid.MustNew("cus_")
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	pages := map[string]*customerports.SearchPage{
		"customers": page(
			map[string]any{"id": customerID.String(), "name": "Acme Foods", "city": "Dallas"},
		),
	}

	balances := &fakeBalances{balances: []*kernelports.CustomerOpenBalance{
		{CustomerID: customerID, OpenMinor: 4821050, OverdueMinor: 1701000, CurrencyCode: "USD"},
	}}
	svc := &Service{
		logger:       zaptest.NewLogger(t),
		config:       indexConfig(),
		searchRepo:   repoReturning(t, pages),
		shipmentRead: &fakeCardReader{},
		permissions:  allowAllPermissionsMock(t),
		visibility:   everyRecordVisible{},
		balances:     balances,
	}
	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       "acme",
		TenantInfo:  tenant,
		EntityTypes: []search.EntityType{search.EntityTypeCustomer},
	})
	require.NoError(t, err)
	customers := groupOf(result, search.EntityTypeCustomer)
	require.Equal(t, "48210.50", customers.Hits[0].Metadata["openBalance"])
	require.Equal(t, "17010.00", customers.Hits[0].Metadata["overdueBalance"])
	require.Equal(t, "USD", customers.Hits[0].Metadata["currencyCode"])
	require.Equal(t, "Dallas", customers.Hits[0].Metadata["city"])

	denied := &fakeBalances{}
	svc = &Service{
		logger:       zaptest.NewLogger(t),
		config:       indexConfig(),
		searchRepo:   repoReturning(t, pages),
		shipmentRead: &fakeCardReader{},
		permissions:  permissionsDenying(t, permission.ResourceAccountsReceivable),
		visibility:   everyRecordVisible{},
		balances:     denied,
	}
	result, err = svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       "acme",
		TenantInfo:  tenant,
		EntityTypes: []search.EntityType{search.EntityTypeCustomer},
	})
	require.NoError(t, err)
	require.False(t, denied.called)
	require.NotContains(
		t,
		groupOf(result, search.EntityTypeCustomer).Hits[0].Metadata,
		"openBalance",
	)
}

func TestSearchLabelsTheRecordADocumentIsAttachedToWhenReadable(t *testing.T) {
	shipmentID := pulid.MustNew("shp_")
	customerID := pulid.MustNew("cus_")
	labels := &fakeLabels{labels: map[string][]*captureports.CaptureRecordLabel{
		"shipment": {{ID: shipmentID, Title: "S-240193"}},
		"customer": {{ID: customerID, Title: "Acme Foods"}},
	}}

	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: indexConfig(),
		searchRepo: repoReturning(t, map[string]*customerports.SearchPage{
			"documents": page(
				map[string]any{
					"id":            "doc_1",
					"original_name": "BOL-88213.pdf",
					"resource_type": "shipment",
					"resource_id":   shipmentID.String(),
					"detected_kind": "BillOfLading",
					"created_at":    float64(1760171520),
				},
				map[string]any{
					"id":            "doc_2",
					"original_name": "W9.pdf",
					"resource_type": "customer",
					"resource_id":   customerID.String(),
				},
			),
		}),
		shipmentRead: &fakeCardReader{},
		permissions:  permissionsDenying(t, permission.ResourceCustomer),
		visibility:   everyRecordVisible{},
		recordLabels: labels,
	}

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query: "bol",
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		EntityTypes: []search.EntityType{search.EntityTypeDocument},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"shipment"}, labels.asked)

	documents := groupOf(result, search.EntityTypeDocument)
	require.Len(t, documents.Hits, 2)
	require.Equal(t, "S-240193", documents.Hits[0].Metadata["attachedLabel"])
	require.Equal(t, "BillOfLading", documents.Hits[0].Metadata["detectedKind"])
	require.Equal(t, "1760171520", documents.Hits[0].Metadata["receivedAt"])
	require.NotContains(t, documents.Hits[1].Metadata, "attachedLabel")
}

func TestSearchReportsTheIndexTotalBeyondTheReturnedHits(t *testing.T) {
	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: indexConfig(),
		searchRepo: repoReturning(t, map[string]*customerports.SearchPage{
			"workers": {
				Documents: []map[string]any{
					{"id": pulid.MustNew("wrk_").String(), "first_name": "Ann"},
				},
				EstimatedTotal: 41,
			},
		}),
		shipmentRead: &fakeCardReader{},
		permissions:  allowAllPermissionsMock(t),
		visibility:   everyRecordVisible{},
	}

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query: "ann",
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		EntityTypes: []search.EntityType{search.EntityTypeWorker},
	})
	require.NoError(t, err)
	require.Equal(t, 41, groupOf(result, search.EntityTypeWorker).Total)
}

func TestSearchBrowsesOneKindOfRecordOnABlankQuery(t *testing.T) {
	var requests []customerports.SearchRequest
	repo := customerportsmock.NewMockSearchRepository(t)
	repo.EXPECT().Enabled().Return(true).Once()
	repo.EXPECT().
		SearchPage(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req customerports.SearchRequest) (*customerports.SearchPage, error) {
			requests = append(requests, req)
			return &customerports.SearchPage{
				Documents: []map[string]any{
					{"id": pulid.MustNew("wrk_").String(), "first_name": "Ann", "last_name": "Lee"},
				},
				EstimatedTotal: 64,
			}, nil
		}).
		Once()

	svc := &Service{
		logger:       zaptest.NewLogger(t),
		config:       indexConfig(),
		searchRepo:   repo,
		shipmentRead: &fakeCardReader{},
		permissions:  allowAllPermissionsMock(t),
		visibility:   everyRecordVisible{},
	}

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query: "  ",
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		EntityTypes: []search.EntityType{search.EntityTypeWorker},
	})
	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, "", requests[0].Query)
	require.Equal(t, "", result.Query)
	require.Equal(t, 64, groupOf(result, search.EntityTypeWorker).Total)
	require.Equal(t, "Ann Lee", groupOf(result, search.EntityTypeWorker).Hits[0].Title)
}

func TestSearchDoesNotBrowseEverythingOnABlankQuery(t *testing.T) {
	svc := &Service{
		logger:      zaptest.NewLogger(t),
		config:      indexConfig(),
		searchRepo:  customerportsmock.NewMockSearchRepository(t),
		permissions: permissionportsmock.NewMockPermissionEngine(t),
		visibility:  everyRecordVisible{},
	}

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		EntityTypes: []search.EntityType{search.EntityTypeWorker, search.EntityTypeCustomer},
	})
	require.NoError(t, err)
	require.Empty(t, result.Groups)
}
