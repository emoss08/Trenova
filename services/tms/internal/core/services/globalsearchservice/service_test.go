package globalsearchservice

import (
	"context"
	"errors"
	"testing"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func allowAllPermissionsMock(t *testing.T) *mocks.MockPermissionEngine {
	t.Helper()

	engine := mocks.NewMockPermissionEngine(t)
	engine.EXPECT().
		CheckBatch(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *serviceports.BatchPermissionCheckRequest,
		) (*serviceports.BatchPermissionCheckResult, error) {
			results := make([]serviceports.PermissionCheckResult, len(req.Checks))
			for idx := range req.Checks {
				results[idx] = serviceports.PermissionCheckResult{Allowed: true}
			}
			return &serviceports.BatchPermissionCheckResult{Results: results}, nil
		}).
		Maybe()

	return engine
}

func TestSearchReturnsEmptyForBlankQuery(t *testing.T) {
	repo := mocks.NewMockSearchRepository(t)
	svc := &Service{
		logger:      zaptest.NewLogger(t),
		config:      &config.SearchConfig{},
		searchRepo:  repo,
		permissions: allowAllPermissionsMock(t),
	}

	result, err := svc.Search(context.Background(), &serviceports.GlobalSearchRequest{})
	require.NoError(t, err)
	require.Equal(t, "", result.Query)
	require.Empty(t, result.Groups)
}

func TestSearchFiltersByRequestedEntityTypesAndTenant(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo := mocks.NewMockSearchRepository(t)
	var requests []repoports.SearchRequest
	repo.EXPECT().Enabled().Return(true).Once()
	repo.EXPECT().
		Search(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repoports.SearchRequest) ([]map[string]any, error) {
			requests = append(requests, req)
			return []map[string]any{
				{
					"id":               "wrk_123",
					"first_name":       "Sam",
					"last_name":        "Carter",
					"status":           "Active",
					"organization_id":  orgID.String(),
					"business_unit_id": buID.String(),
				},
			}, nil
		}).
		Once()

	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: &config.SearchConfig{
			Enabled: true,
			Meilisearch: config.MeilisearchConfig{
				Indexes: config.MeilisearchIndexConfig{
					Shipments: "shipments",
					Customers: "customers",
					Workers:   "workers",
					Documents: "documents",
				},
			},
		},
		searchRepo:  repo,
		permissions: allowAllPermissionsMock(t),
	}

	result, err := svc.Search(context.Background(), &serviceports.GlobalSearchRequest{
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
		`organization_id = "`+orgID.String()+`" AND business_unit_id = "`+buID.String()+`"`,
		requests[0].Filter,
	)
	require.Len(t, result.Groups, 1)
	require.Equal(t, search.EntityTypeWorker, result.Groups[0].EntityType)
	require.Len(t, result.Groups[0].Hits, 1)
	require.Equal(t, "Sam Carter", result.Groups[0].Hits[0].Title)
	require.Equal(
		t,
		"/dispatch/workers?panelEntityId=wrk_123&panelType=edit",
		result.Groups[0].Hits[0].Href,
	)
}

func TestSearchClampsRequestedLimit(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo := mocks.NewMockSearchRepository(t)
	var requests []repoports.SearchRequest
	repo.EXPECT().Enabled().Return(true).Once()
	repo.EXPECT().
		Search(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req repoports.SearchRequest) ([]map[string]any, error) {
			requests = append(requests, req)
			return nil, nil
		}).
		Once()

	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: &config.SearchConfig{
			Enabled: true,
			Meilisearch: config.MeilisearchConfig{
				Indexes: config.MeilisearchIndexConfig{
					Workers: "workers",
				},
			},
		},
		searchRepo:  repo,
		permissions: allowAllPermissionsMock(t),
	}

	_, err := svc.Search(context.Background(), &serviceports.GlobalSearchRequest{
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
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo := mocks.NewMockSearchRepository(t)
	repo.EXPECT().Enabled().Return(true).Once()
	repo.EXPECT().
		Search(mock.Anything, mock.MatchedBy(func(req repoports.SearchRequest) bool {
			return req.Index == "customers"
		})).
		Return(nil, errors.New("boom")).
		Once()
	repo.EXPECT().
		Search(mock.Anything, mock.MatchedBy(func(req repoports.SearchRequest) bool {
			return req.Index == "workers"
		})).
		Return([]map[string]any{
			{
				"id":               "wrk_123",
				"first_name":       "Sam",
				"last_name":        "Carter",
				"status":           "Active",
				"organization_id":  orgID.String(),
				"business_unit_id": buID.String(),
			},
		}, nil).
		Once()

	svc := &Service{
		logger: zaptest.NewLogger(t),
		config: &config.SearchConfig{
			Enabled: true,
			Meilisearch: config.MeilisearchConfig{
				Indexes: config.MeilisearchIndexConfig{
					Customers: "customers",
					Workers:   "workers",
				},
			},
		},
		searchRepo:  repo,
		permissions: allowAllPermissionsMock(t),
	}

	result, err := svc.Search(context.Background(), &serviceports.GlobalSearchRequest{
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
