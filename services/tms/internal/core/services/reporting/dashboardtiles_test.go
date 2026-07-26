package reporting

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/reporting/canned"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
	_ context.Context, _ *repositories.GetReportDashboardRequest,
) (*report.Dashboard, error) {
	if s.dashboard == nil {
		return nil, errortypes.NewNotFoundError("Dashboard not found")
	}
	return s.dashboard, nil
}

func (s *stubDashboardRepo) List(
	_ context.Context, _ *repositories.ListReportDashboardsRequest,
) ([]*report.Dashboard, error) {
	return nil, nil
}

func (s *stubDashboardRepo) Delete(
	_ context.Context, _ *repositories.GetReportDashboardRequest,
) error {
	return nil
}

type stubDefinitionRepo struct {
	definitions map[pulid.ID]*report.ReportDefinition
}

func (s *stubDefinitionRepo) GetByID(
	_ context.Context, req *repositories.GetReportDefinitionRequest,
) (*report.ReportDefinition, error) {
	entity, ok := s.definitions[req.DefinitionID]
	if !ok {
		return nil, errortypes.NewNotFoundError("ReportDefinition not found")
	}
	return entity, nil
}

func (s *stubDefinitionRepo) ListConnection(
	_ context.Context, _ *repositories.ListReportDefinitionConnectionRequest,
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

func (s *stubDefinitionRepo) UpdateStatus(
	_ context.Context, _ *repositories.GetReportDefinitionRequest,
	_ report.DefinitionStatus, _ []string,
) error {
	return nil
}

func (s *stubDefinitionRepo) List(
	_ context.Context, _ *repositories.ListReportDefinitionsRequest,
) ([]*report.ReportDefinition, error) {
	return nil, nil
}

func (s *stubDefinitionRepo) Delete(
	_ context.Context, _ *repositories.DeleteReportDefinitionRequest,
) error {
	return nil
}

func (s *stubDefinitionRepo) GetRevision(
	_ context.Context, _ *repositories.GetReportRevisionRequest,
) (*report.ReportDefinitionRevision, error) {
	return nil, nil
}

func (s *stubDefinitionRepo) ListRevisions(
	_ context.Context, _ *repositories.ListReportRevisionsRequest,
) ([]*report.ReportDefinitionRevision, error) {
	return nil, nil
}

var (
	tileViewerID = pulid.MustNew("usr_")
	tileDefID    = pulid.MustNew("rd_")
)

func tileTestService(
	dashboard *report.Dashboard,
	definitions map[pulid.ID]*report.ReportDefinition,
) *Service {
	return &Service{
		l:             zap.NewNop(),
		dashboardRepo: &stubDashboardRepo{dashboard: dashboard},
		defRepo:       &stubDefinitionRepo{definitions: definitions},
		canned:        canned.Default(),
	}
}

func tileRequest() Request {
	return Request{
		TenantInfo: pagination.TenantInfo{UserID: tileViewerID},
	}
}

func sharedDefinition(entity string, filters *report.FilterGroup) *report.ReportDefinition {
	return &report.ReportDefinition{
		ID:         tileDefID,
		Name:       "Shipment Volume",
		Visibility: report.VisibilityShared,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entity,
			Filters:   filters,
			Parameters: []report.ParameterDef{
				{Name: "windowDays"},
			},
		},
	}
}

func dashboardWith(tiles []report.DashboardTile, filters []report.DashboardFilter) *report.Dashboard {
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
		[]report.DashboardFilter{{
			ID:       "f1",
			Entity:   "shipment",
			Ref:      report.FieldRef{Field: "status"},
			Operator: dbtype.OpEqual,
		}},
	)
	service := tileTestService(dashboard, map[pulid.ID]*report.ReportDefinition{
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
	require.NotNil(t, resolved[0].Definition.Filters)
	assert.Equal(t, "Completed", resolved[0].Definition.Filters.Filters[0].Value)
	assert.Equal(t, map[string]any{"windowDays": 90}, resolved[0].Params)
	assert.Equal(t, "Shipment Volume", resolved[0].Title, "an untitled tile falls back to the report")
}

// A text tile is a note on the canvas; querying for it would be meaningless.
func TestResolveDashboardTilesSkipsTilesWithoutReports(t *testing.T) {
	dashboard := dashboardWith([]report.DashboardTile{
		{ID: "note", Kind: report.TileKindText, Text: "Weekly review"},
		{ID: "t1", Kind: report.TileKindTable, DefinitionID: tileDefID, Title: "Volume"},
	}, nil)
	service := tileTestService(dashboard, map[pulid.ID]*report.ReportDefinition{
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
	stored := sharedDefinition("shipment", &report.FilterGroup{
		Op: report.BoolOpAnd,
		Filters: []report.FieldFilter{
			{Ref: report.FieldRef{Field: "status"}, Operator: dbtype.OpEqual, Value: "New"},
		},
	})
	dashboard := dashboardWith(
		[]report.DashboardTile{{
			ID: "t1", Kind: report.TileKindTable, DefinitionID: tileDefID, Limit: 25,
		}},
		[]report.DashboardFilter{{
			ID: "f1", Entity: "shipment",
			Ref: report.FieldRef{Field: "status"}, Operator: dbtype.OpEqual,
		}},
	)
	service := tileTestService(dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: stored,
	})

	resolved, err := service.ResolveDashboardTiles(t.Context(), &ResolveDashboardTilesRequest{
		Request:      tileRequest(),
		FilterValues: map[string]any{"f1": "Completed"},
	})

	require.NoError(t, err)
	assert.Equal(t, 25, resolved[0].Definition.Limit)
	assert.Equal(t, "Completed", resolved[0].Definition.Filters.Filters[0].Value)
	assert.Equal(t, "New", stored.Definition.Filters.Filters[0].Value, "stored report untouched")
	assert.Zero(t, stored.Definition.Limit, "the tile's limit never leaks into the report")
}

// A tile limit must be applied even when no dashboard filter was active — that
// is the path where ApplyFilters hands back the original pointer.
func TestResolveDashboardTilesAppliesLimitWithoutFilters(t *testing.T) {
	stored := sharedDefinition("shipment", nil)
	dashboard := dashboardWith([]report.DashboardTile{{
		ID: "t1", Kind: report.TileKindChart, DefinitionID: tileDefID, Limit: 10,
	}}, nil)
	service := tileTestService(dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: stored,
	})

	resolved, err := service.ResolveDashboardTiles(t.Context(), &ResolveDashboardTilesRequest{
		Request: tileRequest(),
	})

	require.NoError(t, err)
	assert.Equal(t, 10, resolved[0].Definition.Limit)
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
	service := tileTestService(dashboard, map[pulid.ID]*report.ReportDefinition{
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
	service := tileTestService(dashboard, nil)

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
		[]report.DashboardFilter{{
			ID: "f1", Entity: "invoice",
			Ref: report.FieldRef{Field: "status"}, Operator: dbtype.OpEqual,
		}},
	)
	service := tileTestService(dashboard, map[pulid.ID]*report.ReportDefinition{
		tileDefID: sharedDefinition("shipment", nil),
	})

	resolved, err := service.ResolveDashboardTiles(t.Context(), &ResolveDashboardTilesRequest{
		Request:      tileRequest(),
		FilterValues: map[string]any{"f1": "Posted"},
	})

	require.NoError(t, err)
	assert.Nil(t, resolved[0].Definition.Filters)
}
