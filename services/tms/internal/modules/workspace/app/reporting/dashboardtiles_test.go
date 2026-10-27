package reporting

import (
	"context"
	"testing"

	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/canned"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDashboardRepo struct {
	dashboard *report.Dashboard
}

func (s *stubDashboardRepo) Create(
	_ context.Context, e *report.Dashboard,
) (*report.Dashboard, error) {
	return e, nil
}

func (s *stubDashboardRepo) Update(
	_ context.Context, e *report.Dashboard,
) (*report.Dashboard, error) {
	return e, nil
}

func (s *stubDashboardRepo) GetByID(
	_ context.Context, _ *reportports.GetReportDashboardRequest,
) (*report.Dashboard, error) {
	if s.dashboard == nil {
		return nil, errortypes.NewNotFoundError("Dashboard not found")
	}
	return s.dashboard, nil
}

func (s *stubDashboardRepo) List(
	_ context.Context, _ *reportports.ListReportDashboardsRequest,
) ([]*report.Dashboard, error) {
	return nil, nil
}

func (s *stubDashboardRepo) Delete(
	_ context.Context, _ *reportports.GetReportDashboardRequest,
) error {
	return nil
}

type stubDefinitionRepo struct {
	definitions map[pulid.ID]*report.ReportDefinition
}

func (s *stubDefinitionRepo) GetByID(
	_ context.Context, req *reportports.GetReportDefinitionRequest,
) (*report.ReportDefinition, error) {
	entity, ok := s.definitions[req.DefinitionID]
	if !ok {
		return nil, errortypes.NewNotFoundError("ReportDefinition not found")
	}
	return entity, nil
}

func (s *stubDefinitionRepo) ListConnection(
	_ context.Context, _ *reportports.ListReportDefinitionConnectionRequest,
) (*pagination.CursorListResult[*report.ReportDefinition], error) {
	return nil, nil
}

func (s *stubDefinitionRepo) Create(
	_ context.Context, e *report.ReportDefinition, _ pulid.ID,
) (*report.ReportDefinition, error) {
	return e, nil
}

func (s *stubDefinitionRepo) Update(
	_ context.Context, e *report.ReportDefinition, _ pulid.ID,
) (*report.ReportDefinition, error) {
	return e, nil
}

func (s *stubDefinitionRepo) ListCatalogStale(
	_ context.Context, _ *reportports.ListCatalogStaleReportDefinitionsRequest,
) ([]*report.ReportDefinition, error) {
	return nil, nil
}

func (s *stubDefinitionRepo) RecordCatalogCheck(
	_ context.Context, _ *reportports.RecordReportCatalogCheckRequest,
) (bool, error) {
	return false, nil
}

func (s *stubDefinitionRepo) List(
	_ context.Context, _ *reportports.ListReportDefinitionsRequest,
) ([]*report.ReportDefinition, error) {
	return nil, nil
}

func (s *stubDefinitionRepo) Delete(
	_ context.Context, _ *reportports.DeleteReportDefinitionRequest,
) error {
	return nil
}

func (s *stubDefinitionRepo) GetRevision(
	_ context.Context, _ *reportports.GetReportRevisionRequest,
) (*report.ReportDefinitionRevision, error) {
	return nil, nil
}

func (s *stubDefinitionRepo) ListRevisions(
	_ context.Context, _ *reportports.ListReportRevisionsRequest,
) ([]*report.ReportDefinitionRevision, error) {
	return nil, nil
}

var (
	tileViewerID = pulid.MustNew("usr_")
	tileDefID    = pulid.MustNew("rd_")
)

func tileTestService(
	t *testing.T,
	dashboard *report.Dashboard,
	definitions map[pulid.ID]*report.ReportDefinition,
) *Service {
	t.Helper()
	library, err := canned.Default()
	require.NoError(t, err)
	svc, _ := engineService(t, engineOptions{})
	svc.dashboardRepo = &stubDashboardRepo{dashboard: dashboard}
	svc.defRepo = &stubDefinitionRepo{definitions: definitions}
	svc.canned = library
	return svc
}

func filterValue(filter *ir.Filter, field string) (any, bool) {
	if filter == nil {
		return nil, false
	}
	if filter.Field == field && filter.And == nil && filter.Or == nil {
		return filter.Value, true
	}
	for i := range filter.And {
		if value, ok := filterValue(&filter.And[i], field); ok {
			return value, true
		}
	}
	for i := range filter.Or {
		if value, ok := filterValue(&filter.Or[i], field); ok {
			return value, true
		}
	}
	return nil, false
}

func tileRequest() Request {
	return Request{
		TenantInfo: pagination.TenantInfo{UserID: tileViewerID},
	}
}

func sharedDefinition(entity string, filters *ir.Filter) *report.ReportDefinition {
	return &report.ReportDefinition{
		ID:         tileDefID,
		Name:       "Shipment Volume",
		Visibility: report.VisibilityShared,
		Definition: &ir.Definition{
			Version: ir.Version,
			Entity:  entity,
			Columns: []ir.Column{{ID: "status", Dimension: &ir.Dimension{Field: "status"}}},
			Filters: filters,
			Parameters: []ir.Parameter{
				{Name: "windowDays", Type: "int"},
			},
		},
	}
}

func dashboardWith(
	tiles []report.DashboardTile,
	filters []ir.DashboardFilter,
) *report.Dashboard {
	return &report.Dashboard{
		OwnerID:    tileViewerID,
		Visibility: report.VisibilityShared,
		Layout: &report.DashboardLayout{
			Tiles:   tiles,
			Filters: filters,
		},
	}
}

// The whole point of step 2: the server derives what each tile queries, with
// the viewer's filter and parameter choices folded in.
func TestResolveDashboardTilesFoldsFiltersAndParams(t *testing.T) {
	dashboard := dashboardWith(
		[]report.DashboardTile{{
			ID:           "t1",
			Kind:         report.TileKindTable,
			DefinitionID: tileDefID,
		}},
		[]ir.DashboardFilter{{
			ID:     "f1",
			Entity: "shipment",
			Field:  "status",
			Op:     ir.OpEq,
		}},
	)
	service := tileTestService(t, dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: sharedDefinition("shipment", nil),
	})

	resolved, err := service.ResolveDashboardTiles(
		t.Context(),
		&ResolveDashboardTilesRequest{
			Request:      tileRequest(),
			FilterValues: map[string]any{"f1": "Completed"},
			ParamValues:  map[string]any{"windowDays": 90},
		},
	)

	require.NoError(t, err)
	require.Len(t, resolved, 1)
	value, found := filterValue(resolved[0].Definition.Filters, "status")
	require.True(t, found)
	assert.Equal(t, "Completed", value)
	assert.Equal(t, map[string]any{"windowDays": 90}, resolved[0].Params)
	assert.Equal(
		t,
		"Shipment Volume",
		resolved[0].Title,
		"an untitled tile falls back to the report",
	)
}

// A text tile is a note on the canvas; querying for it would be meaningless.
func TestResolveDashboardTilesSkipsTilesWithoutReports(t *testing.T) {
	dashboard := dashboardWith([]report.DashboardTile{
		{ID: "note", Kind: report.TileKindText, Text: "Weekly review"},
		{ID: "t1", Kind: report.TileKindTable, DefinitionID: tileDefID, Title: "Volume"},
	}, nil)
	service := tileTestService(t, dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: sharedDefinition("shipment", nil),
	})

	resolved, err := service.ResolveDashboardTiles(t.Context(), &ResolveDashboardTilesRequest{
		Request: tileRequest(),
	})

	require.NoError(t, err)
	require.Len(t, resolved, 1)
	assert.Equal(t, "t1", resolved[0].TileID)
	assert.Equal(t, "Volume", resolved[0].Title, "an authored title wins")
}

// The stored report is shared by every tile and by the report library, so
// resolving one tile must never write through to it.
func TestResolveDashboardTilesNeverMutatesTheStoredReport(t *testing.T) {
	stored := sharedDefinition("shipment", ir.And(
		ir.Filter{Field: "status", Op: ir.OpEq, Value: "New"},
	))
	dashboard := dashboardWith(
		[]report.DashboardTile{{
			ID: "t1", Kind: report.TileKindTable, DefinitionID: tileDefID, Limit: 25,
		}},
		[]ir.DashboardFilter{{
			ID: "f1", Entity: "shipment", Field: "status", Op: ir.OpEq,
		}},
	)
	service := tileTestService(t, dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: stored,
	})

	resolved, err := service.ResolveDashboardTiles(t.Context(), &ResolveDashboardTilesRequest{
		Request:      tileRequest(),
		FilterValues: map[string]any{"f1": "Completed"},
	})

	require.NoError(t, err)
	assert.Equal(t, ir.N(25), resolved[0].Definition.Limit)
	assert.NotSame(t, stored.Definition, resolved[0].Definition)
	stillNew, found := filterValue(stored.Definition.Filters, "status")
	require.True(t, found)
	assert.Equal(t, "New", stillNew, "stored report untouched")
	assert.Equal(
		t,
		1,
		countLeaves(stored.Definition.Filters),
		"no filter is added to the stored report",
	)
	assert.Zero(t, stored.Definition.Limit, "the tile's limit never leaks into the report")
}

// A tile limit must be applied even when no dashboard filter was active — that
// is the path where ApplyFilters hands back the original pointer.
func TestResolveDashboardTilesAppliesLimitWithoutFilters(t *testing.T) {
	stored := sharedDefinition("shipment", nil)
	dashboard := dashboardWith([]report.DashboardTile{{
		ID: "t1", Kind: report.TileKindChart, DefinitionID: tileDefID, Limit: 10,
	}}, nil)
	service := tileTestService(t, dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: stored,
	})

	resolved, err := service.ResolveDashboardTiles(t.Context(), &ResolveDashboardTilesRequest{
		Request: tileRequest(),
	})

	require.NoError(t, err)
	assert.Equal(t, ir.N(10), resolved[0].Definition.Limit)
	assert.Zero(t, stored.Definition.Limit, "stored report untouched")
}

// A dashboard is not a way around report visibility: the tile's report is
// fetched through GetDefinition, which enforces the viewer's own access.
func TestResolveDashboardTilesEnforcesReportVisibility(t *testing.T) {
	private := sharedDefinition("shipment", nil)
	private.Visibility = report.VisibilityPrivate
	private.OwnerID = pulid.MustNew("usr_")

	dashboard := dashboardWith([]report.DashboardTile{{
		ID: "t1", Kind: report.TileKindTable, DefinitionID: tileDefID,
	}}, nil)
	service := tileTestService(t, dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: private,
	})

	_, err := service.ResolveDashboardTiles(t.Context(), &ResolveDashboardTilesRequest{
		Request: tileRequest(),
	})

	require.Error(t, err)
}

func TestResolveDashboardTilesRejectsUnknownCannedKey(t *testing.T) {
	dashboard := dashboardWith([]report.DashboardTile{{
		ID: "t1", Kind: report.TileKindTable, CannedKey: "not_a_real_report",
	}}, nil)
	service := tileTestService(t, dashboard, nil)

	_, err := service.ResolveDashboardTiles(t.Context(), &ResolveDashboardTilesRequest{
		Request: tileRequest(),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_a_real_report")
}

// A dashboard filter on another entity has no column to bind to on this
// tile's report, so it must pass through untouched.
func TestResolveDashboardTilesLeavesUnrelatedEntitiesAlone(t *testing.T) {
	dashboard := dashboardWith(
		[]report.DashboardTile{{ID: "t1", Kind: report.TileKindTable, DefinitionID: tileDefID}},
		[]ir.DashboardFilter{{
			ID: "f1", Entity: "invoice", Field: "status", Op: ir.OpEq,
		}},
	)
	service := tileTestService(t, dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: sharedDefinition("shipment", nil),
	})

	resolved, err := service.ResolveDashboardTiles(t.Context(), &ResolveDashboardTilesRequest{
		Request:      tileRequest(),
		FilterValues: map[string]any{"f1": "Posted"},
	})

	require.NoError(t, err)
	assert.Nil(t, resolved[0].Definition.Filters)
}

func countLeaves(filter *ir.Filter) int {
	if filter == nil {
		return 0
	}
	if filter.And == nil && filter.Or == nil {
		return 1
	}
	count := 0
	for i := range filter.And {
		count += countLeaves(&filter.And[i])
	}
	for i := range filter.Or {
		count += countLeaves(&filter.Or[i])
	}
	return count
}
