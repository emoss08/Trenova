//go:build integration

package executor

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/reporting/compiler"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/reporting/render"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/permtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
)

const (
	e2eProOne = "E2E-PRO-0001"
	e2eProTwo = "E2E-PRO-0002"
)

type pipelineFixture struct {
	db       *bun.DB
	tenant   pagination.TenantInfo
	compiler *compiler.Compiler
	executor services.ReportDatasetExecutor
}

func setupPipeline(t *testing.T, ctx context.Context, db *bun.DB) *pipelineFixture {
	t.Helper()

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenantInfo := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}

	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenantInfo)
	for i, pro := range []string{e2eProOne, e2eProTwo} {
		graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenantInfo,
			testutil.ShipmentGraphParams{
				BOL:          pro + "-BOL",
				ProNumber:    pro,
				ShipmentID:   pulid.MustNew("shp_"),
				MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
			})
		_, err := db.NewUpdate().
			Model((*shipment.Shipment)(nil)).
			Set("total_charge_amount = ?", []string{"125.5000", "74.2500"}[i]).
			Where("id = ?", graph.Shipment.ID).
			Exec(ctx)
		require.NoError(t, err)
	}

	return &pipelineFixture{
		db:     db,
		tenant: tenantInfo,
		compiler: compiler.NewWithCatalog(
			&reportcatalog.Default,
			permtest.AllowAll(),
			permission.NewRegistry(),
			&config.ReportingConfig{},
			zap.NewNop(),
		),
		executor: New(Params{
			DB:     &postgres.ReportingConnection{Connection: postgres.NewTestConnection(db)},
			Config: &config.Config{},
			Logger: zap.NewNop(),
		}),
	}
}

func (f *pipelineFixture) compile(
	t *testing.T,
	def *report.Definition,
) *services.CompiledReportQuery {
	t.Helper()
	compiled, err := f.compiler.Compile(context.Background(), &services.ReportCompileRequest{
		Definition:  def,
		Tenant:      f.tenant,
		OrgTimezone: "America/Chicago",
		NowUnix:     timeutils.NowUnix(),
	})
	require.NoError(t, err)
	return compiled
}

func e2eDefinition() *report.Definition {
	return &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			{
				ID:   "c_pro",
				Ref:  report.FieldRef{Field: "proNumber"},
				Kind: report.ColumnKindDimension,
			},
			{
				ID:   "c_customer",
				Ref:  report.FieldRef{Path: []string{"customer"}, Field: "name"},
				Kind: report.ColumnKindDimension,
			},
			{
				ID:   "c_total",
				Ref:  report.FieldRef{Field: "totalChargeAmount"},
				Kind: report.ColumnKindDimension,
			},
		},
		Sort: []report.SortSpec{
			{ColumnID: "c_pro", Direction: "asc"},
		},
	}
}

func renderMeta() services.ReportRunMeta {
	return services.ReportRunMeta{
		Title:           "E2E Pipeline Report",
		GeneratedAtUnix: timeutils.NowUnix(),
		Timezone:        "America/Chicago",
		RequestedBy:     "integration-test",
	}
}

// TestPipelineCompileExecuteRender drives the real seam end to end on live
// Postgres: compile → executor cursor stream → each renderer, asserting the
// seeded rows arrive byte-correct in every format.
func TestPipelineCompileExecuteRender(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	f := setupPipeline(t, ctx, db)
	compiled := f.compile(t, e2eDefinition())

	openDataset := func() services.ReportDatasetReader {
		reader, err := f.executor.Open(ctx, &services.OpenReportDatasetRequest{
			Compiled: compiled,
			MaxRows:  1000,
			Timeout:  time.Minute,
		})
		require.NoError(t, err)
		return reader
	}

	t.Run("csv", func(t *testing.T) {
		reader := openDataset()
		defer reader.Close()

		var buf bytes.Buffer
		csvRenderer := render.NewCSV(render.CSVParams{Config: &config.Config{}})
		stats, err := csvRenderer.Render(ctx, &services.ReportRenderRequest{
			Dataset: reader,
			Sink:    &buf,
			Meta:    renderMeta(),
		})
		require.NoError(t, err)
		assert.EqualValues(t, 2, stats.Rows)
		assert.False(t, stats.Truncated)

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		require.Len(t, lines, 3)
		assert.Contains(t, lines[1], e2eProOne)
		assert.Contains(t, lines[1], "125.5")
		assert.Contains(t, lines[2], e2eProTwo)
	})

	t.Run("json", func(t *testing.T) {
		reader := openDataset()
		defer reader.Close()

		var buf bytes.Buffer
		stats, err := render.NewJSON().Render(ctx, &services.ReportRenderRequest{
			Dataset: reader,
			Sink:    &buf,
			Meta:    renderMeta(),
		})
		require.NoError(t, err)
		assert.EqualValues(t, 2, stats.Rows)

		var envelope struct {
			Rows [][]any `json:"rows"`
		}
		require.NoError(t, sonic.Unmarshal(buf.Bytes(), &envelope))
		require.Len(t, envelope.Rows, 2)
		assert.Equal(t, e2eProOne, envelope.Rows[0][0])
	})

	t.Run("xlsx", func(t *testing.T) {
		reader := openDataset()
		defer reader.Close()

		var buf bytes.Buffer
		stats, err := render.NewXLSX().Render(ctx, &services.ReportRenderRequest{
			Dataset: reader,
			Sink:    &buf,
			Meta:    renderMeta(),
		})
		require.NoError(t, err)
		assert.EqualValues(t, 2, stats.Rows)

		workbook, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
		require.NoError(t, err)
		defer workbook.Close()

		cell, err := workbook.GetCellValue("Data", "A2")
		require.NoError(t, err)
		assert.Equal(t, e2eProOne, cell)
	})
}

// TestPipelineRowCapTruncates asserts the executor's row cap flows through the
// dataset contract: the reader reports truncation and the renderer flags it.
func TestPipelineRowCapTruncates(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	f := setupPipeline(t, ctx, db)
	compiled := f.compile(t, e2eDefinition())

	reader, err := f.executor.Open(ctx, &services.OpenReportDatasetRequest{
		Compiled: compiled,
		MaxRows:  1,
		Timeout:  time.Minute,
	})
	require.NoError(t, err)
	defer reader.Close()

	var buf bytes.Buffer
	stats, err := render.NewCSV(render.CSVParams{Config: &config.Config{}}).
		Render(ctx, &services.ReportRenderRequest{
			Dataset: reader,
			Sink:    &buf,
			Meta:    renderMeta(),
		})
	require.NoError(t, err)
	assert.EqualValues(t, 1, stats.Rows)
	assert.True(t, stats.Truncated)
	assert.True(t, reader.Truncated())
	assert.Contains(t, buf.String(), e2eProOne)
	assert.NotContains(t, buf.String(), e2eProTwo)
}

// TestPipelineAggregationCorrectness executes a grouped money aggregation on
// live Postgres and asserts NUMERIC math survives the typed decode path.
func TestPipelineAggregationCorrectness(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	f := setupPipeline(t, ctx, db)
	compiled := f.compile(t, &report.Definition{
		IRVersion: report.CurrentIRVersion,
		Entity:    "shipment",
		Columns: []report.ColumnSpec{
			{
				ID:   "c_customer",
				Ref:  report.FieldRef{Path: []string{"customer"}, Field: "name"},
				Kind: report.ColumnKindDimension,
			},
			{
				ID:   "c_total",
				Ref:  report.FieldRef{Field: "totalChargeAmount"},
				Kind: report.ColumnKindMeasure,
				Agg:  reportcatalog.AggSum,
			},
			{
				ID:   "c_count",
				Ref:  report.FieldRef{Field: "id"},
				Kind: report.ColumnKindMeasure,
				Agg:  reportcatalog.AggCount,
			},
		},
	})

	reader, err := f.executor.Open(ctx, &services.OpenReportDatasetRequest{
		Compiled: compiled,
		MaxRows:  100,
		Timeout:  time.Minute,
	})
	require.NoError(t, err)
	defer reader.Close()

	var buf bytes.Buffer
	stats, err := render.NewJSON().Render(ctx, &services.ReportRenderRequest{
		Dataset: reader,
		Sink:    &buf,
		Meta:    renderMeta(),
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, stats.Rows)

	var envelope struct {
		Rows [][]any `json:"rows"`
	}
	require.NoError(t, sonic.Unmarshal(buf.Bytes(), &envelope))
	require.Len(t, envelope.Rows, 1)
	assert.Equal(t, "199.75", envelope.Rows[0][1])
	assert.EqualValues(t, 2, envelope.Rows[0][2])
}
