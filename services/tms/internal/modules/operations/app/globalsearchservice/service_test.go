package globalsearchservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/searchconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
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

func allowAllPermissionsMock(t *testing.T) *permissionportsmock.MockPermissionEngine {
	t.Helper()

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.EXPECT().
		CheckBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *permissionports.BatchPermissionCheckRequest,
		) (*permissionports.BatchPermissionCheckResult, error) {
			results := make([]permissionports.PermissionCheckResult, len(req.Checks))
			for idx := range req.Checks {
				results[idx] = permissionports.PermissionCheckResult{Allowed: true}
			}
			return &permissionports.BatchPermissionCheckResult{Results: results}, nil
		})

	return engine
}

func page(documents ...map[string]any) *customerports.SearchPage {
	return &customerports.SearchPage{Documents: documents, EstimatedTotal: len(documents)}
}

func TestSearchReturnsEmptyForBlankQuery(t *testing.T) {
	repo := customerportsmock.NewMockSearchRepository(t)
	svc := &Service{
		logger:      zaptest.NewLogger(t),
		config:      &searchconfig.Config{},
		searchRepo:  repo,
		permissions: permissionportsmock.NewMockPermissionEngine(t),
		visibility:  everyRecordVisible{},
	}

	result, err := svc.Search(context.Background(), &kernelports.GlobalSearchRequest{})
	require.NoError(t, err)
	require.Equal(t, "", result.Query)
	require.Empty(t, result.Groups)
}

func TestSearchFiltersByRequestedEntityTypesAndTenant(t *testing.T) {
	workerID := pulid.MustNew("wrk_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo := customerportsmock.NewMockSearchRepository(t)
	var requests []customerports.SearchRequest
	repo.EXPECT().Enabled().Return(true).Once()
	repo.EXPECT().
		SearchPage(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req customerports.SearchRequest) (*customerports.SearchPage, error) {
			requests = append(requests, req)
			return page(map[string]any{
				"id":               workerID.String(),
				"first_name":       "Sam",
				"last_name":        "Carter",
				"status":           "Active",
				"organization_id":  orgID.String(),
				"business_unit_id": buID.String(),
			}), nil
		}).
		Once()

	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: &searchconfig.Config{
			Enabled: true,
			Meilisearch: searchconfig.MeilisearchConfig{
				Indexes: searchconfig.MeilisearchIndexConfig{
					Shipments: "shipments",
					Customers: "customers",
					Workers:   "workers",
					Documents: "documents",
				},
			},
		},
		searchRepo:  repo,
		permissions: allowAllPermissionsMock(t),
		visibility:  everyRecordVisible{},
	}

	result, err := svc.Search(context.Background(), &kernelports.GlobalSearchRequest{
		Query: "sam",
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		EntityTypes: []search.EntityType{search.EntityTypeWorker},
	})
	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, "workers", requests[0].Index)
	require.Equal(
		t,
		`(organization_id = "`+orgID.String()+`") AND (business_unit_id = "`+buID.String()+`")`,
		requests[0].Filter,
	)
	require.Len(t, result.Groups, 1)
	require.Equal(t, search.EntityTypeWorker, result.Groups[0].EntityType)
	require.Len(t, result.Groups[0].Hits, 1)
	require.Equal(t, "Sam Carter", result.Groups[0].Hits[0].Title)
	require.Equal(
		t,
		"/hr/workers?panelEntityId="+workerID.String()+"&panelType=edit",
		result.Groups[0].Hits[0].Href,
	)
}

func TestSearchClampsRequestedLimit(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo := customerportsmock.NewMockSearchRepository(t)
	var requests []customerports.SearchRequest
	repo.EXPECT().Enabled().Return(true).Once()
	repo.EXPECT().
		SearchPage(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req customerports.SearchRequest) (*customerports.SearchPage, error) {
			requests = append(requests, req)
			return page(), nil
		}).
		Once()

	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: &searchconfig.Config{
			Enabled: true,
			Meilisearch: searchconfig.MeilisearchConfig{
				Indexes: searchconfig.MeilisearchIndexConfig{
					Workers: "workers",
				},
			},
		},
		searchRepo:  repo,
		permissions: allowAllPermissionsMock(t),
		visibility:  everyRecordVisible{},
	}

	_, err := svc.Search(context.Background(), &kernelports.GlobalSearchRequest{
		Query: "sam",
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		EntityTypes: []search.EntityType{search.EntityTypeWorker},
		Limit:       10_000,
	})
	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, maxSearchLimit, requests[0].Limit)
}

func TestSearchContinuesWhenOneIndexFails(t *testing.T) {
	workerID := pulid.MustNew("wrk_")
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo := customerportsmock.NewMockSearchRepository(t)
	repo.EXPECT().Enabled().Return(true).Once()
	repo.EXPECT().
		SearchPage(mock.Anything, mock.MatchedBy(func(req customerports.SearchRequest) bool {
			return req.Index == "customers"
		})).
		Return(nil, errors.New("boom")).
		Once()
	repo.EXPECT().
		SearchPage(mock.Anything, mock.MatchedBy(func(req customerports.SearchRequest) bool {
			return req.Index == "workers"
		})).
		Return(page(
			map[string]any{
				"id":               workerID.String(),
				"first_name":       "Sam",
				"last_name":        "Carter",
				"status":           "Active",
				"organization_id":  orgID.String(),
				"business_unit_id": buID.String(),
			},
		), nil).
		Once()

	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: &searchconfig.Config{
			Enabled: true,
			Meilisearch: searchconfig.MeilisearchConfig{
				Indexes: searchconfig.MeilisearchIndexConfig{
					Customers: "customers",
					Workers:   "workers",
				},
			},
		},
		searchRepo:  repo,
		permissions: allowAllPermissionsMock(t),
		visibility:  everyRecordVisible{},
	}

	result, err := svc.Search(context.Background(), &kernelports.GlobalSearchRequest{
		Query: "sam",
		TenantInfo: pagination.TenantInfo{
			OrgID: orgID,
			BuID:  buID,
		},
		EntityTypes: []search.EntityType{search.EntityTypeCustomer, search.EntityTypeWorker},
	})
	require.NoError(t, err)
	require.Len(t, result.Groups, 1)
	require.Equal(t, search.EntityTypeWorker, result.Groups[0].EntityType)
}
