package shipmentboardservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubBoardRepo struct {
	stageRows   []*repositories.ShipmentStageSummaryRow
	counts      map[shipment.QuickFilter]int
	lastScope   *repositories.ShipmentBoardScope
	lastFilters []shipment.QuickFilterSpec
}

func (s *stubBoardRepo) StageSummary(
	_ context.Context,
	scope *repositories.ShipmentBoardScope,
) ([]*repositories.ShipmentStageSummaryRow, error) {
	s.lastScope = scope
	return s.stageRows, nil
}

func (s *stubBoardRepo) QuickFilterTotals(
	_ context.Context,
	req *repositories.CountShipmentQuickFiltersRequest,
) ([]repositories.ShipmentQuickFilterTotal, error) {
	s.lastScope = req.Scope
	s.lastFilters = req.Filters
	totals := make([]repositories.ShipmentQuickFilterTotal, len(req.Filters))
	for i, spec := range req.Filters {
		totals[i].Count = s.counts[spec.Filter]
	}
	return totals, nil
}

type stubBasis struct {
	last *services.ResolveShipmentQuickFilterBasisRequest
}

func (s *stubBasis) Resolve(
	_ context.Context,
	req *services.ResolveShipmentQuickFilterBasisRequest,
) (*repositories.ShipmentQuickFilterBasis, error) {
	s.last = req
	return &repositories.ShipmentQuickFilterBasis{Now: time.Unix(0, 0), Location: time.UTC}, nil
}

func (s *stubBasis) Prepare(
	context.Context,
	pagination.TenantInfo,
	*repositories.ShipmentOptions,
) error {
	return nil
}

type stubAI struct {
	repositories.AIProviderRepository
	tasks map[aiprovider.Task]bool
}

func (s *stubAI) ListForTask(
	_ context.Context,
	req repositories.ListAIProvidersForTaskRequest,
) ([]*aiprovider.Provider, error) {
	if s.tasks[req.Task] {
		return []*aiprovider.Provider{{}}, nil
	}
	return nil, nil
}

type stubOrgs struct {
	repositories.OrganizationRepository
	caps *repositories.OrganizationCapabilities
}

func (s *stubOrgs) GetCapabilities(
	context.Context,
	repositories.GetOrganizationCapabilitiesRequest,
) (*repositories.OrganizationCapabilities, error) {
	return s.caps, nil
}

type stubIntegrations struct {
	repositories.IntegrationRepository
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

func scopeRequest(filters ...shipment.QuickFilterSpec) *services.ShipmentBoardScopeRequest {
	return &services.ShipmentBoardScopeRequest{
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
		Organizations: &stubOrgs{caps: &repositories.OrganizationCapabilities{
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
		Organizations: &stubOrgs{caps: &repositories.OrganizationCapabilities{
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

	repo := &stubBoardRepo{stageRows: []*repositories.ShipmentStageSummaryRow{
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
