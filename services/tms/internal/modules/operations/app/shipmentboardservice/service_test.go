package shipmentboardservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/boardgrouping"
	"github.com/emoss08/trenova/internal/modules/operations/domain/quickfilter"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubBoardRepo struct {
	stageRows   []*shipmentboardports.ShipmentStageSummaryRow
	counts      map[quickfilter.QuickFilter]int
	groupRows   []*shipmentboardports.ShipmentBoardGroupRow
	lastScope   *shipmentboardports.ShipmentBoardScope
	lastFilters []quickfilter.QuickFilterSpec
	lastGroups  *shipmentboardports.SummarizeShipmentBoardGroupsQuery
}

func (s *stubBoardRepo) GroupSummary(
	_ context.Context,
	req *shipmentboardports.SummarizeShipmentBoardGroupsQuery,
) ([]*shipmentboardports.ShipmentBoardGroupRow, error) {
	s.lastScope = req.Scope
	s.lastGroups = req
	return s.groupRows, nil
}

func (s *stubBoardRepo) StageSummary(
	_ context.Context,
	scope *shipmentboardports.ShipmentBoardScope,
) ([]*shipmentboardports.ShipmentStageSummaryRow, error) {
	s.lastScope = scope
	return s.stageRows, nil
}

func (s *stubBoardRepo) QuickFilterTotals(
	_ context.Context,
	req *shipmentboardports.CountShipmentQuickFiltersRequest,
) ([]shipmentboardports.ShipmentQuickFilterTotal, error) {
	s.lastScope = req.Scope
	s.lastFilters = req.Filters
	totals := make([]shipmentboardports.ShipmentQuickFilterTotal, len(req.Filters))
	for i, spec := range req.Filters {
		totals[i].Count = s.counts[spec.Filter]
	}
	return totals, nil
}

type stubBasis struct {
	last *shipmentports.ResolveShipmentQuickFilterBasisRequest
}

func (s *stubBasis) Resolve(
	_ context.Context,
	req *shipmentports.ResolveShipmentQuickFilterBasisRequest,
) (*shipmentports.ShipmentQuickFilterBasis, error) {
	s.last = req
	return &shipmentports.ShipmentQuickFilterBasis{Now: time.Unix(0, 0), Location: time.UTC}, nil
}

func (s *stubBasis) Prepare(
	context.Context,
	pagination.TenantInfo,
	*shipmentports.ShipmentOptions,
) error {
	return nil
}

// stubNova is whether Nova, which writes the board's briefings and insights,
// runs for the tenant.
type stubNova struct {
	llmports.TaskRunner
	ready bool
}

func (s stubNova) Available(context.Context, pagination.TenantInfo) (bool, error) {
	return s.ready, nil
}

type stubOrgs struct {
	tenantports.OrganizationRepository
	caps *tenantports.OrganizationCapabilities
}

func (s *stubOrgs) GetCapabilities(
	context.Context,
	tenantports.GetOrganizationCapabilitiesRequest,
) (*tenantports.OrganizationCapabilities, error) {
	return s.caps, nil
}

type stubIntegrations struct {
	integrationports.IntegrationRepository
	records []*integration.Integration
}

func (s *stubIntegrations) ListByTenant(
	context.Context,
	pagination.TenantInfo,
) ([]*integration.Integration, error) {
	return s.records, nil
}

type stubMaps struct{ ready bool }

func (s stubMaps) GetClientRuntimeConfig(
	context.Context,
	pagination.TenantInfo,
	integration.Type,
) (*integrationservice.RuntimeConfig, error) {
	return &integrationservice.RuntimeConfig{Ready: s.ready}, nil
}

type Dependencies struct {
	Repo          shipmentboardports.ShipmentBoardRepository
	QuickFilters  shipmentports.ShipmentQuickFilterBasisResolver
	Nova          llmports.TaskRunner
	Organizations tenantports.OrganizationRepository
	Integrations  integrationports.IntegrationRepository
	Maps          MapsReadiness
	Cache         shipmentboardports.ShipmentBoardCache
}

func NewWithDependencies(d *Dependencies) *Service {
	return &Service{
		repo:          d.Repo,
		quickFilters:  d.QuickFilters,
		nova:          d.Nova,
		organizations: d.Organizations,
		integrations:  d.Integrations,
		maps:          d.Maps,
		cache:         d.Cache,
		l:             zap.NewNop(),
	}
}

type memoryBoardCache struct {
	epoch   int64
	entries map[string][]byte
}

func newMemoryBoardCache() *memoryBoardCache {
	return &memoryBoardCache{entries: make(map[string][]byte)}
}

func memoryCacheKey(key *shipmentboardports.ShipmentBoardCacheKey) string {
	return fmt.Sprintf("%s|%s|%d|%s", key.Section, key.Timezone, key.Epoch, key.Variant)
}

func (c *memoryBoardCache) Epoch(context.Context, pagination.TenantInfo) (int64, error) {
	return c.epoch, nil
}

func (c *memoryBoardCache) Get(
	_ context.Context,
	key *shipmentboardports.ShipmentBoardCacheKey,
	dest any,
) (bool, error) {
	raw, ok := c.entries[memoryCacheKey(key)]
	if !ok {
		return false, nil
	}
	return true, sonic.Unmarshal(raw, dest)
}

func (c *memoryBoardCache) Set(
	_ context.Context,
	key *shipmentboardports.ShipmentBoardCacheKey,
	value any,
) error {
	raw, err := sonic.Marshal(value)
	if err != nil {
		return err
	}
	c.entries[memoryCacheKey(key)] = raw
	return nil
}

func scopeRequest(
	filters ...quickfilter.QuickFilterSpec,
) *shipmentboardports.ShipmentBoardScopeRequest {
	return &shipmentboardports.ShipmentBoardScopeRequest{
		Filter:       &pagination.QueryOptions{},
		QuickFilters: filters,
		Timezone:     "UTC",
		Margins:      true,
	}
}

func scopeWithoutMargins(
	filters ...quickfilter.QuickFilterSpec,
) *shipmentboardports.ShipmentBoardScopeRequest {
	req := scopeRequest(filters...)
	req.Margins = false
	return req
}

func TestCapabilities(t *testing.T) {
	t.Parallel()

	svc := NewWithDependencies(&Dependencies{
		Nova: stubNova{ready: true},
		Organizations: &stubOrgs{caps: &tenantports.OrganizationCapabilities{
			BrokerageEnabled: true,
		}},
		Integrations: &stubIntegrations{records: []*integration.Integration{{
			Type:     integration.TypeSamsara,
			Category: integration.CategoryTelematics,
			Enabled:  true,
		}}},
		Maps: stubMaps{ready: true},
	})

	caps, err := svc.Capabilities(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	assert.True(t, caps.AI)
	assert.Equal(t, tenant.OperationTypeBrokerage, caps.OperationType)
	assert.True(t, caps.HOS)
	assert.True(t, caps.Maps)

	svc = NewWithDependencies(&Dependencies{
		Nova: stubNova{},
		Organizations: &stubOrgs{caps: &tenantports.OrganizationCapabilities{
			AssetOperationsEnabled: true,
		}},
		Integrations: &stubIntegrations{},
		Maps:         stubMaps{},
	})
	caps, err = svc.Capabilities(t.Context(), pagination.TenantInfo{})
	require.NoError(t, err)
	assert.False(t, caps.AI)
	assert.Equal(t, tenant.OperationTypeAsset, caps.OperationType)
	assert.False(t, caps.HOS)
	assert.False(t, caps.Maps)
}

func TestStageSummaryZeroFillsEveryStage(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{stageRows: []*shipmentboardports.ShipmentStageSummaryRow{
		{StageRank: shipmentvocab.StageMoving.Rank(), Count: 4, Revenue: decimal.NewFromInt(900)},
	}}
	basis := &stubBasis{}
	svc := NewWithDependencies(&Dependencies{Repo: repo, QuickFilters: basis})

	rows, err := svc.StageSummary(
		t.Context(),
		scopeRequest(quickfilter.Quick(quickfilter.QuickFilterLowMargin)),
	)
	require.NoError(t, err)
	require.Len(t, rows, len(shipmentvocab.Stages()))
	assert.Equal(t, shipmentvocab.StageLate, rows[0].Stage)
	assert.Equal(t, 0, rows[0].Count)
	assert.True(t, rows[0].Revenue.IsZero())
	assert.Equal(t, 4, rows[2].Count)
	assert.True(t, basis.last.Margin)
	assert.False(t, basis.last.Detention)
	assert.Len(t, repo.lastScope.Options.QuickFilters, 1)
}

func TestQuickFilterCountsCountsEveryCountableFilter(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{counts: map[quickfilter.QuickFilter]int{quickfilter.QuickFilterLate: 3}}
	basis := &stubBasis{}
	svc := NewWithDependencies(&Dependencies{Repo: repo, QuickFilters: basis})

	rows, err := svc.QuickFilterCounts(t.Context(), scopeRequest())
	require.NoError(t, err)
	require.Len(t, rows, len(quickfilter.CountableQuickFilters()))
	assert.Equal(t, 3, rows[0].Count)
	assert.True(t, basis.last.Margin)
	assert.True(t, basis.last.Detention)
	assert.Len(t, repo.lastFilters, len(quickfilter.CountableQuickFilters()))
}

func TestQuickFilterCountsAreServedFromTheCacheForTheSameScope(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{counts: map[quickfilter.QuickFilter]int{quickfilter.QuickFilterLate: 3}}
	svc := NewWithDependencies(&Dependencies{
		Repo:         repo,
		QuickFilters: &stubBasis{},
		Cache:        newMemoryBoardCache(),
	})

	first, err := svc.QuickFilterCounts(t.Context(), scopeRequest())
	require.NoError(t, err)

	repo.lastFilters = nil
	second, err := svc.QuickFilterCounts(t.Context(), scopeRequest())
	require.NoError(t, err)

	assert.Nil(t, repo.lastFilters)
	assert.Equal(t, first, second)
}

func TestQuickFilterCountsAreNotSharedAcrossScopes(t *testing.T) {
	t.Parallel()

	narrowed := scopeRequest()
	narrowed.Filter.Query = "PRO-1"

	cases := []struct {
		name string
		req  *shipmentboardports.ShipmentBoardScopeRequest
	}{
		{name: "a search", req: narrowed},
		{name: "a reader without the margins group", req: scopeWithoutMargins()},
		{name: "a quick filter", req: scopeRequest(quickfilter.Quick(quickfilter.QuickFilterLate))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &stubBoardRepo{}
			svc := NewWithDependencies(&Dependencies{
				Repo:         repo,
				QuickFilters: &stubBasis{},
				Cache:        newMemoryBoardCache(),
			})

			_, err := svc.QuickFilterCounts(t.Context(), scopeRequest())
			require.NoError(t, err)

			repo.lastFilters = nil
			_, err = svc.QuickFilterCounts(t.Context(), tc.req)
			require.NoError(t, err)

			assert.NotNil(t, repo.lastFilters)
		})
	}
}

func TestANewEpochRecountsTheQuickFilters(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{}
	cache := newMemoryBoardCache()
	svc := NewWithDependencies(&Dependencies{
		Repo:         repo,
		QuickFilters: &stubBasis{},
		Cache:        cache,
	})

	_, err := svc.QuickFilterCounts(t.Context(), scopeRequest())
	require.NoError(t, err)

	repo.lastFilters = nil
	cache.epoch++
	_, err = svc.QuickFilterCounts(t.Context(), scopeRequest())
	require.NoError(t, err)

	assert.NotNil(t, repo.lastFilters)
}

func TestALowMarginFilterIsRefusedWithoutTheMarginsGroup(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{}
	basis := &stubBasis{}
	svc := NewWithDependencies(&Dependencies{Repo: repo, QuickFilters: basis})

	_, err := svc.StageSummary(
		t.Context(),
		scopeWithoutMargins(quickfilter.Quick(quickfilter.QuickFilterLowMargin)),
	)
	require.Error(t, err, "the loads it returns say which ones run thin")
	assert.Nil(t, repo.lastScope, "nothing is read for a refused filter")
}

func TestQuickFilterCountsLeaveOutTheLowMarginCountWithoutTheMarginsGroup(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{counts: map[quickfilter.QuickFilter]int{quickfilter.QuickFilterLate: 3}}
	basis := &stubBasis{}
	svc := NewWithDependencies(&Dependencies{Repo: repo, QuickFilters: basis})

	rows, err := svc.QuickFilterCounts(t.Context(), scopeWithoutMargins())
	require.NoError(t, err)
	require.Len(t, rows, len(quickfilter.CountableQuickFilters())-1)
	for _, row := range rows {
		assert.NotEqual(t, quickfilter.QuickFilterLowMargin, row.Filter)
	}
	assert.False(t, basis.last.Margin, "no margin basis is resolved for a count nobody sees")
}

func TestGroupSummaryCountsGroupsInTheBoardsScopeAndZone(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{groupRows: []*shipmentboardports.ShipmentBoardGroupRow{
		{Key: "2026-10-07", Count: 5, Revenue: decimal.NewFromInt(4200)},
		{Key: "", Count: 1, Revenue: decimal.Zero},
	}}
	basis := &stubBasis{}
	svc := NewWithDependencies(&Dependencies{Repo: repo, QuickFilters: basis})

	req := scopeRequest(quickfilter.Quick(quickfilter.QuickFilterLate))
	groups, err := svc.GroupSummary(
		t.Context(),
		&shipmentboardports.SummarizeShipmentBoardGroupsRequest{
			Scope:   req,
			GroupBy: boardgrouping.BoardGroupingDeliveryDate,
		},
	)
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, "2026-10-07", groups[0].Key)
	assert.Equal(t, 5, groups[0].Count)
	assert.True(t, groups[0].Revenue.Equal(decimal.NewFromInt(4200)))
	assert.Empty(t, groups[1].Key)
	assert.Equal(t, boardgrouping.BoardGroupingDeliveryDate, repo.lastGroups.GroupBy)
	assert.Equal(t, req.Timezone, repo.lastGroups.Timezone)
	assert.Len(t, repo.lastScope.Options.QuickFilters, 1)
}

func TestGroupSummaryRefusesAGroupingTheBoardDoesNotOffer(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{}
	svc := NewWithDependencies(&Dependencies{Repo: repo, QuickFilters: &stubBasis{}})

	_, err := svc.GroupSummary(t.Context(), &shipmentboardports.SummarizeShipmentBoardGroupsRequest{
		Scope:   scopeRequest(),
		GroupBy: boardgrouping.BoardGrouping("Stage"),
	})
	require.Error(t, err)
	assert.Nil(t, repo.lastGroups)
}
