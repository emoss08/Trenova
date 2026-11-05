package globalsearchservice

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A person who sees only their own shipments has the index asked for more
// than the limit, so the shipments they may see are not crowded out by the
// ones they may not; the group is cut back to the limit once those are gone.
func TestSearch_ANarrowedGroupFillsItsLimitFromPastTheHiddenHits(t *testing.T) {
	t.Parallel()

	const limit = 5
	documents := make([]map[string]any, 0, limit*narrowedOverfetch)
	visible := make([]pulid.ID, 0, limit+2)
	for idx := range limit * narrowedOverfetch {
		id := pulid.MustNew("shp_")
		documents = append(documents, map[string]any{
			"id": id.String(), "pro_number": fmt.Sprintf("PRO-%d", idx),
		})
		if idx%2 == 1 && len(visible) < limit+2 {
			visible = append(visible, id)
		}
	}
	svc, requests := scopedSearchService(
		t,
		map[string]*customerports.SearchPage{"shipments": page(documents...)},
		permissionsGranting(t, map[permission.Resource]grant{
			permission.ResourceShipment: {scope: permission.DataScopeOwn},
		}),
		&fakeVisibility{visible: map[string][]pulid.ID{"shipment": visible}},
	)

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       "pro",
		TenantInfo:  searchTenant(),
		Limit:       limit,
		EntityTypes: []search.EntityType{search.EntityTypeShipment},
	})
	require.NoError(t, err)

	require.Len(t, *requests, 1)
	assert.Equal(t, limit*narrowedOverfetch, (*requests)[0].Limit)
	group := groupFor(result, search.EntityTypeShipment)
	require.NotNil(t, group)
	want := make([]string, 0, limit)
	for _, id := range visible[:limit] {
		want = append(want, id.String())
	}
	assert.Equal(t, want, hitIDsOf(group))
	assert.Equal(t, limit+2, group.Total)
}

// Someone who sees every record of the organization is asked for no more than
// the limit: nothing will be hidden from them.
func TestSearch_AnOrganizationWideGroupAsksForOnlyTheLimit(t *testing.T) {
	t.Parallel()

	svc, requests := scopedSearchService(t, nil, permissionsGranting(t, nil), everyRecordVisible{})

	_, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       "pro",
		TenantInfo:  searchTenant(),
		Limit:       5,
		EntityTypes: []search.EntityType{search.EntityTypeShipment},
	})
	require.NoError(t, err)
	require.Len(t, *requests, 1)
	assert.Equal(t, 5, (*requests)[0].Limit)
}

// A pasted page is searched for its first maxQueryRunes characters, cut on a
// character and not inside one.
func TestSearch_SearchesOnlyTheStartOfAVeryLongQuery(t *testing.T) {
	t.Parallel()

	svc, requests := scopedSearchService(t, nil, permissionsGranting(t, nil), everyRecordVisible{})
	long := strings.Repeat("é", maxQueryRunes+50)

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       long,
		TenantInfo:  searchTenant(),
		EntityTypes: []search.EntityType{search.EntityTypeShipment},
	})
	require.NoError(t, err)
	require.Len(t, *requests, 1)
	assert.Equal(t, strings.Repeat("é", maxQueryRunes), (*requests)[0].Query)
	assert.True(t, utf8.ValidString((*requests)[0].Query))
	assert.Equal(t, (*requests)[0].Query, result.Query)
}

// A kind the search does not find is refused as invalid, not answered empty.
func TestSearch_RefusesAKindOfRecordItDoesNotFind(t *testing.T) {
	t.Parallel()

	svc := &Service{balances: &fakeBalances{}, config: indexConfig(), views: searchViews()}

	_, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:       "pro",
		TenantInfo:  searchTenant(),
		EntityTypes: []search.EntityType{search.EntityTypeShipment, "trailer"},
	})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "entityTypes", validation.Field)
}
