package globalsearchservice

import (
	"context"
	"errors"
	"sync"
	"testing"

	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestSearchReturnsEmptyForBlankQuery(t *testing.T) {
	svc := &Service{
		logger:      zaptest.NewLogger(t),
		config:      &config.SearchConfig{},
		searchRepo:  &fakeSearchRepository{enabled: true},
		permissions: fakePermissionEngine{},
	}

	result, err := svc.Search(context.Background(), &serviceports.GlobalSearchRequest{})
	require.NoError(t, err)
	require.Equal(t, "", result.Query)
	require.Empty(t, result.Groups)
}

func TestSearchFiltersByRequestedEntityTypesAndTenant(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo := &fakeSearchRepository{
		enabled: true,
		resultsByIndex: map[string][]map[string]any{
			"workers": {
				{
					"id":               "wrk_123",
					"first_name":       "Sam",
					"last_name":        "Carter",
					"status":           "Active",
					"organization_id":  orgID.String(),
					"business_unit_id": buID.String(),
				},
			},
		},
	}

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
		permissions: fakePermissionEngine{},
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
	require.Len(t, repo.requests, 1)
	require.Equal(t, "workers", repo.requests[0].Index)
	require.Equal(
		t,
		`organization_id = "`+orgID.String()+`" AND business_unit_id = "`+buID.String()+`"`,
		repo.requests[0].Filter,
	)
	require.Len(t, result.Groups, 1)
	require.Equal(t, search.EntityTypeWorker, result.Groups[0].EntityType)
	require.Len(t, result.Groups[0].Hits, 1)
	require.Equal(t, "Sam Carter", result.Groups[0].Hits[0].Title)
	require.Equal(t, "/dispatch/workers?panelEntityId=wrk_123&panelType=edit", result.Groups[0].Hits[0].Href)
}

func TestSearchClampsRequestedLimit(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo := &fakeSearchRepository{
		enabled:        true,
		resultsByIndex: map[string][]map[string]any{},
	}

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
		permissions: fakePermissionEngine{},
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
	require.Len(t, repo.requests, 1)
	require.Equal(t, maxSearchLimit, repo.requests[0].Limit)
}

func TestSearchContinuesWhenOneIndexFails(t *testing.T) {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	repo := &fakeSearchRepository{
		enabled: true,
		resultsByIndex: map[string][]map[string]any{
			"workers": {
				{
					"id":               "wrk_123",
					"first_name":       "Sam",
					"last_name":        "Carter",
					"status":           "Active",
					"organization_id":  orgID.String(),
					"business_unit_id": buID.String(),
				},
			},
		},
		errorsByIndex: map[string]error{
			"customers": errors.New("boom"),
		},
	}

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
		permissions: fakePermissionEngine{},
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

type fakeSearchRepository struct {
	enabled        bool
	resultsByIndex map[string][]map[string]any
	errorsByIndex  map[string]error
	mu             sync.Mutex
	requests       []repoports.SearchRequest
}

func (f *fakeSearchRepository) Enabled() bool {
	return f.enabled
}

func (f *fakeSearchRepository) Search(
	_ context.Context,
	req repoports.SearchRequest,
) ([]map[string]any, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.mu.Unlock()
	if err := f.errorsByIndex[req.Index]; err != nil {
		return nil, err
	}
	return f.resultsByIndex[req.Index], nil
}

type fakePermissionEngine struct{}

func (fakePermissionEngine) Check(
	context.Context,
	*serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	return &serviceports.PermissionCheckResult{Allowed: true}, nil
}

func (fakePermissionEngine) CheckBatch(
	_ context.Context,
	req *serviceports.BatchPermissionCheckRequest,
) (*serviceports.BatchPermissionCheckResult, error) {
	results := make([]serviceports.PermissionCheckResult, len(req.Checks))
	for idx := range req.Checks {
		results[idx] = serviceports.PermissionCheckResult{Allowed: true}
	}
	return &serviceports.BatchPermissionCheckResult{Results: results}, nil
}

func (fakePermissionEngine) GetLightManifest(
	context.Context,
	pulid.ID,
	pulid.ID,
) (*serviceports.LightPermissionManifest, error) {
	return nil, nil
}

func (fakePermissionEngine) GetResourcePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
	string,
) (*serviceports.ResourcePermissionDetail, error) {
	return nil, nil
}

func (fakePermissionEngine) InvalidateUser(context.Context, pulid.ID, pulid.ID) error {
	return nil
}

func (fakePermissionEngine) GetEffectivePermissions(
	context.Context,
	pulid.ID,
	pulid.ID,
) (*serviceports.EffectivePermissions, error) {
	return nil, nil
}

func (fakePermissionEngine) SimulatePermissions(
	context.Context,
	*serviceports.SimulatePermissionsRequest,
) (*serviceports.EffectivePermissions, error) {
	return nil, nil
}

var _ serviceports.PermissionEngine = fakePermissionEngine{}
var _ repoports.SearchRepository = (*fakeSearchRepository)(nil)
