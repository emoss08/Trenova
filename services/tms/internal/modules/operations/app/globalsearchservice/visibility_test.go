package globalsearchservice

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

var databaseScopedKinds = []string{"invoice", "order", "shipment", "worker"}

type everyRecordVisible struct{}

func (everyRecordVisible) Scoped(kind string) bool {
	return slices.Contains(databaseScopedKinds, kind)
}

func (everyRecordVisible) VisibleIDs(
	_ context.Context,
	req *kernelports.ListVisibleRecordsRequest,
) ([]pulid.ID, error) {
	return req.IDs, nil
}

type fakeVisibility struct {
	visible  map[string][]pulid.ID
	err      error
	requests []*kernelports.ListVisibleRecordsRequest
}

func (f *fakeVisibility) Scoped(kind string) bool {
	return slices.Contains(databaseScopedKinds, kind)
}

func (f *fakeVisibility) VisibleIDs(
	_ context.Context,
	req *kernelports.ListVisibleRecordsRequest,
) ([]pulid.ID, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	visible := make([]pulid.ID, 0, len(req.IDs))
	for _, id := range req.IDs {
		if slices.Contains(f.visible[req.Kind], id) {
			visible = append(visible, id)
		}
	}
	return visible, nil
}

type grant struct {
	denied bool
	scope  permission.DataScope
}

func permissionsGranting(
	t *testing.T,
	grants map[permission.Resource]grant,
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
				granted := grants[permission.Resource(check.Resource)]
				scope := granted.scope
				if scope == "" {
					scope = permission.DataScopeOrganization
				}
				results[idx] = permissionports.PermissionCheckResult{
					Allowed:   !granted.denied,
					DataScope: scope,
				}
			}
			return &permissionports.BatchPermissionCheckResult{Results: results}, nil
		})
	return engine
}

func scopedSearchService(
	t *testing.T,
	pages map[string]*customerports.SearchPage,
	engine permissionports.PermissionEngine,
	visibility kernelports.RecordVisibility,
) (*Service, *[]customerports.SearchRequest) {
	t.Helper()

	repo := customerportsmock.NewMockSearchRepository(t)
	requests := make([]customerports.SearchRequest, 0, len(pages))
	repo.EXPECT().Enabled().Return(true)
	repo.EXPECT().
		SearchPage(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req customerports.SearchRequest) (*customerports.SearchPage, error) {
			requests = append(requests, req)
			if found, ok := pages[req.Index]; ok {
				return found, nil
			}
			return page(), nil
		}).
		Maybe()

	return &Service{
		logger:      zaptest.NewLogger(t),
		config:      indexConfig(),
		searchRepo:  repo,
		permissions: engine,
		visibility:  visibility,
	}, &requests
}

func searchTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func groupFor(result *kernelports.GlobalSearchResult, entityType search.EntityType) *kernelports.GlobalSearchGroup {
	for _, group := range result.Groups {
		if group.EntityType == entityType {
			return group
		}
	}
	return nil
}

func hitIDsOf(group *kernelports.GlobalSearchGroup) []string {
	if group == nil {
		return nil
	}
	ids := make([]string, 0, len(group.Hits))
	for _, hit := range group.Hits {
		ids = append(ids, hit.ID)
	}
	return ids
}

func TestSearch_HidesWorkersOutsideTheTeamScope(t *testing.T) {
	t.Parallel()

	mine := pulid.MustNew("wrk_")
	theirs := pulid.MustNew("wrk_")
	workers := page(
		map[string]any{"id": mine.String(), "first_name": "Ana", "last_name": "Diaz"},
		map[string]any{"id": theirs.String(), "first_name": "Ana", "last_name": "Moss"},
	)
	workers.EstimatedTotal = 40
	visibility := &fakeVisibility{visible: map[string][]pulid.ID{"worker": {mine}}}
	svc, _ := scopedSearchService(
		t,
		map[string]*customerports.SearchPage{"workers": workers},
		permissionsGranting(t, map[permission.Resource]grant{
			permission.ResourceWorker: {scope: permission.DataScopeTeam},
		}),
		visibility,
	)

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       "ana",
		TenantInfo:  searchTenant(),
		EntityTypes: []search.EntityType{search.EntityTypeWorker},
	})
	require.NoError(t, err)
	group := groupFor(result, search.EntityTypeWorker)
	require.NotNil(t, group)
	assert.Equal(t, []string{mine.String()}, hitIDsOf(group))
	assert.Equal(t, 1, group.Total)
	require.Len(t, visibility.requests, 1)
	assert.Equal(t, "worker", visibility.requests[0].Kind)
}

func TestSearch_ANarrowedGroupReportsOnlyWhatItShows(t *testing.T) {
	t.Parallel()

	owned := pulid.MustNew("shp_")
	shipments := page(map[string]any{"id": owned.String(), "pro_number": "PRO-1"})
	shipments.EstimatedTotal = 120
	svc, _ := scopedSearchService(
		t,
		map[string]*customerports.SearchPage{"shipments": shipments},
		permissionsGranting(t, map[permission.Resource]grant{
			permission.ResourceShipment: {scope: permission.DataScopeOwn},
		}),
		&fakeVisibility{visible: map[string][]pulid.ID{"shipment": {owned}}},
	)

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       "pro",
		TenantInfo:  searchTenant(),
		EntityTypes: []search.EntityType{search.EntityTypeShipment},
	})
	require.NoError(t, err)
	group := groupFor(result, search.EntityTypeShipment)
	require.NotNil(t, group)
	assert.Equal(t, []string{owned.String()}, hitIDsOf(group))
	assert.Equal(t, 1, group.Total)
}

func TestSearch_AnOrganizationWideGroupKeepsItsEstimatedTotal(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("shp_")
	shipments := page(map[string]any{"id": id.String(), "pro_number": "PRO-1"})
	shipments.EstimatedTotal = 120
	svc, _ := scopedSearchService(
		t,
		map[string]*customerports.SearchPage{"shipments": shipments},
		permissionsGranting(t, nil),
		everyRecordVisible{},
	)

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       "pro",
		TenantInfo:  searchTenant(),
		EntityTypes: []search.EntityType{search.EntityTypeShipment},
	})
	require.NoError(t, err)
	assert.Equal(t, 120, groupFor(result, search.EntityTypeShipment).Total)
}

func TestSearch_HidesDocumentsAttachedToRecordsThePersonCannotRead(t *testing.T) {
	t.Parallel()

	visibleShipment := pulid.MustNew("shp_")
	hiddenShipment := pulid.MustNew("shp_")
	invoice := pulid.MustNew("inv_")
	customer := pulid.MustNew("cus_")
	documents := page(
		map[string]any{
			"id": "doc_visible", "original_name": "bol-visible.pdf",
			"resource_type": "shipment", "resource_id": visibleShipment.String(),
		},
		map[string]any{
			"id": "doc_hidden", "original_name": "bol-hidden.pdf",
			"resource_type": "shipment", "resource_id": hiddenShipment.String(),
		},
		map[string]any{
			"id": "doc_invoice", "original_name": "invoice.pdf",
			"resource_type": "invoice", "resource_id": invoice.String(),
		},
		map[string]any{
			"id": "doc_customer", "original_name": "contract.pdf",
			"resource_type": "customer", "resource_id": customer.String(),
		},
	)
	svc, _ := scopedSearchService(
		t,
		map[string]*customerports.SearchPage{"documents": documents},
		permissionsGranting(t, map[permission.Resource]grant{
			permission.ResourceShipment: {scope: permission.DataScopeOwn},
			permission.ResourceInvoice:  {denied: true},
		}),
		&fakeVisibility{visible: map[string][]pulid.ID{
			"shipment": {visibleShipment},
			"invoice":  {invoice},
		}},
	)

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       "pdf",
		TenantInfo:  searchTenant(),
		EntityTypes: []search.EntityType{search.EntityTypeDocument},
	})
	require.NoError(t, err)
	group := groupFor(result, search.EntityTypeDocument)
	require.NotNil(t, group)
	assert.ElementsMatch(t, []string{"doc_visible", "doc_customer"}, hitIDsOf(group))
	assert.Equal(t, 2, group.Total)
}

func TestSearch_HidesScopedHitsWhenTheirVisibilityCannotBeChecked(t *testing.T) {
	t.Parallel()

	shipments := page(map[string]any{"id": pulid.MustNew("shp_").String(), "pro_number": "PRO-1"})
	customers := page(map[string]any{"id": pulid.MustNew("cus_").String(), "name": "Acme"})
	svc, _ := scopedSearchService(
		t,
		map[string]*customerports.SearchPage{"shipments": shipments, "customers": customers},
		permissionsGranting(t, nil),
		&fakeVisibility{err: errors.New("database unavailable")},
	)

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:      "a",
		TenantInfo: searchTenant(),
		EntityTypes: []search.EntityType{
			search.EntityTypeShipment,
			search.EntityTypeCustomer,
		},
	})
	require.NoError(t, err)
	assert.Nil(t, groupFor(result, search.EntityTypeShipment))
	assert.NotNil(t, groupFor(result, search.EntityTypeCustomer))
}

func TestSearch_AsksEachIndexForOnlyTheFieldsAHitShows(t *testing.T) {
	t.Parallel()

	svc, requests := scopedSearchService(t, nil, permissionsGranting(t, nil), everyRecordVisible{})

	_, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:      "a",
		TenantInfo: searchTenant(),
	})
	require.NoError(t, err)
	require.Len(t, *requests, 4)
	for _, req := range *requests {
		require.NotEmpty(t, req.Attributes, req.Index)
		assert.Contains(t, req.Attributes, "id", req.Index)
		assert.NotContains(t, req.Attributes, "*", req.Index)
	}
}
