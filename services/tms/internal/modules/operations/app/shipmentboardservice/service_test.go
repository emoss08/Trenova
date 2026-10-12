package shipmentboardservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/integrationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubBoardRepo struct {
	stageRows   []*shipmentports.ShipmentStageSummaryRow
	counts      map[shipment.QuickFilter]int
	groupRows   []*shipmentports.ShipmentBoardGroupRow
	lastScope   *shipmentports.ShipmentBoardScope
	lastFilters []shipment.QuickFilterSpec
	lastGroups  *shipmentports.SummarizeShipmentBoardGroupsQuery
}

func (s *stubBoardRepo) GroupSummary(
	_ context.Context,
	req *shipmentports.SummarizeShipmentBoardGroupsQuery,
) ([]*shipmentports.ShipmentBoardGroupRow, error) {
	s.lastScope = req.Scope
	s.lastGroups = req
	return s.groupRows, nil
}

func (s *stubBoardRepo) StageSummary(
	_ context.Context,
	scope *shipmentports.ShipmentBoardScope,
) ([]*shipmentports.ShipmentStageSummaryRow, error) {
	s.lastScope = scope
	return s.stageRows, nil
}

func (s *stubBoardRepo) QuickFilterTotals(
	_ context.Context,
	req *shipmentports.CountShipmentQuickFiltersRequest,
) ([]shipmentports.ShipmentQuickFilterTotal, error) {
	s.lastScope = req.Scope
	s.lastFilters = req.Filters
	totals := make([]shipmentports.ShipmentQuickFilterTotal, len(req.Filters))
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

type stubAI struct {
	agentports.AIProviderRepository
	tasks map[aiprovider.Task]bool
}

func (s *stubAI) ListForTask(
	_ context.Context,
	req agentports.ListAIProvidersForTaskRequest,
) ([]*aiprovider.Provider, error) {
	if s.tasks[req.Task] {
		return []*aiprovider.Provider{{}}, nil
	}
	return nil, nil
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

func scopeRequest(filters ...shipment.QuickFilterSpec) *shipmentports.ShipmentBoardScopeRequest {
	return &shipmentports.ShipmentBoardScopeRequest{
		Filter:       &pagination.QueryOptions{},
		QuickFilters: filters,
		Timezone:     "UTC",
	}
}

func TestCapabilities(t *testing.T) {
	t.Parallel()

	svc := NewWithDependencies(&Dependencies{
		AIProviders: &stubAI{
			tasks: map[aiprovider.Task]bool{aiprovider.TaskOperationalInsights: true},
		},
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
		AIProviders: &stubAI{},
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

	repo := &stubBoardRepo{stageRows: []*shipmentports.ShipmentStageSummaryRow{
		{StageRank: shipment.StageMoving.Rank(), Count: 4, Revenue: decimal.NewFromInt(900)},
	}}
	basis := &stubBasis{}
	svc := NewWithDependencies(&Dependencies{Repo: repo, QuickFilters: basis})

	rows, err := svc.StageSummary(
		t.Context(),
		scopeRequest(shipment.Quick(shipment.QuickFilterLowMargin)),
	)
	require.NoError(t, err)
	require.Len(t, rows, len(shipment.Stages()))
	assert.Equal(t, shipment.StageLate, rows[0].Stage)
	assert.Equal(t, 0, rows[0].Count)
	assert.True(t, rows[0].Revenue.IsZero())
	assert.Equal(t, 4, rows[2].Count)
	assert.True(t, basis.last.Margin)
	assert.False(t, basis.last.Detention)
	assert.Len(t, repo.lastScope.Options.QuickFilters, 1)
}

func TestQuickFilterCountsCountsEveryCountableFilter(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{counts: map[shipment.QuickFilter]int{shipment.QuickFilterLate: 3}}
	basis := &stubBasis{}
	svc := NewWithDependencies(&Dependencies{Repo: repo, QuickFilters: basis})

	rows, err := svc.QuickFilterCounts(t.Context(), scopeRequest())
	require.NoError(t, err)
	require.Len(t, rows, len(shipment.CountableQuickFilters()))
	assert.Equal(t, 3, rows[0].Count)
	assert.True(t, basis.last.Margin)
	assert.True(t, basis.last.Detention)
	assert.Len(t, repo.lastFilters, len(shipment.CountableQuickFilters()))
}

func TestGroupSummaryCountsGroupsInTheBoardsScopeAndZone(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{groupRows: []*shipmentports.ShipmentBoardGroupRow{
		{Key: "2026-10-07", Count: 5, Revenue: decimal.NewFromInt(4200)},
		{Key: "", Count: 1, Revenue: decimal.Zero},
	}}
	basis := &stubBasis{}
	svc := NewWithDependencies(&Dependencies{Repo: repo, QuickFilters: basis})

	req := scopeRequest(shipment.Quick(shipment.QuickFilterLate))
	groups, err := svc.GroupSummary(t.Context(), &shipmentports.SummarizeShipmentBoardGroupsRequest{
		Scope:   req,
		GroupBy: shipment.BoardGroupingDeliveryDate,
	})
	require.NoError(t, err)
	require.Len(t, groups, 2)
	assert.Equal(t, "2026-10-07", groups[0].Key)
	assert.Equal(t, 5, groups[0].Count)
	assert.True(t, groups[0].Revenue.Equal(decimal.NewFromInt(4200)))
	assert.Empty(t, groups[1].Key)
	assert.Equal(t, shipment.BoardGroupingDeliveryDate, repo.lastGroups.GroupBy)
	assert.Equal(t, req.Timezone, repo.lastGroups.Timezone)
	assert.Len(t, repo.lastScope.Options.QuickFilters, 1)
}

func TestGroupSummaryRefusesAGroupingTheBoardDoesNotOffer(t *testing.T) {
	t.Parallel()

	repo := &stubBoardRepo{}
	svc := NewWithDependencies(&Dependencies{Repo: repo, QuickFilters: &stubBasis{}})

	_, err := svc.GroupSummary(t.Context(), &shipmentports.SummarizeShipmentBoardGroupsRequest{
		Scope:   scopeRequest(),
		GroupBy: shipment.BoardGrouping("Stage"),
	})
	require.Error(t, err)
	assert.Nil(t, repo.lastGroups)
}
