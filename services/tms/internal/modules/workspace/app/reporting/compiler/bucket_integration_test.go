//go:build integration

package compiler

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportfmt"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateBucketsStartAtTheOrganizationsLocalPeriod(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	defer cleanup()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{
		OrgID:  data.Organization.ID,
		BuID:   data.BusinessUnit.ID,
		UserID: data.User.ID,
	}
	fixture := testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant)

	cases := []struct {
		name    string
		pro     string
		local   time.Time
		bucket  report.DateBucket
		start   time.Time
		display string
	}{
		{
			name:    "month",
			pro:     "BUCKET-MONTH",
			local:   time.Date(2026, 3, 1, 0, 30, 0, 0, chicago),
			bucket:  report.DateBucketMonth,
			start:   time.Date(2026, 3, 1, 0, 0, 0, 0, chicago),
			display: "Mar 2026",
		},
		{
			name:    "day across the spring-forward change",
			pro:     "BUCKET-DAY",
			local:   time.Date(2026, 3, 8, 23, 45, 0, 0, chicago),
			bucket:  report.DateBucketDay,
			start:   time.Date(2026, 3, 8, 0, 0, 0, 0, chicago),
			display: "2026-03-08",
		},
		{
			name:    "year",
			pro:     "BUCKET-YEAR",
			local:   time.Date(2026, 1, 1, 0, 5, 0, 0, chicago),
			bucket:  report.DateBucketYear,
			start:   time.Date(2026, 1, 1, 0, 0, 0, 0, chicago),
			display: "2026",
		},
	}

	c := newTestCompiler(allowAllEngine())
	for _, tc := range cases {
		graph := testutil.CreateShipmentGraph(t, ctx, db, fixture, tenant,
			testutil.ShipmentGraphParams{
				BOL:          tc.pro + "-BOL",
				ProNumber:    tc.pro,
				ShipmentID:   pulid.MustNew("shp_"),
				MoveStatuses: []shipment.MoveStatus{shipment.MoveStatusNew},
			})
		_, err = db.NewUpdate().
			Model((*shipment.Shipment)(nil)).
			Set("created_at = ?", tc.local.Unix()).
			Where("id = ?", graph.Shipment.ID).
			Exec(ctx)
		require.NoError(t, err)

		compiled, compileErr := c.Compile(ctx, &reportports.ReportCompileRequest{
			Definition: &report.Definition{
				IRVersion: report.CurrentIRVersion,
				Entity:    "shipment",
				Columns: []report.ColumnSpec{
					{
						ID:     "c_period",
						Ref:    report.FieldRef{Field: "createdAt"},
						Kind:   report.ColumnKindDimension,
						Bucket: tc.bucket,
					},
					measure("c_count", reportcatalog.AggCount, "id"),
				},
				Filters: &report.FilterGroup{
					Op: report.BoolOpAnd,
					Filters: []report.FieldFilter{{
						Ref:      report.FieldRef{Field: "proNumber"},
						Operator: dbtype.OpEqual,
						Value:    tc.pro,
					}},
				},
			},
			Tenant:      tenant,
			OrgTimezone: chicago.String(),
			NowUnix:     tc.local.Unix(),
		})
		require.NoError(t, compileErr, tc.name)

		var period, count int64
		require.NoError(t,
			db.QueryRowContext(ctx, compiled.SQL, compiled.Args...).Scan(&period, &count),
			tc.name)
		assert.Equal(t, int64(1), count, tc.name)
		assert.Equal(t, tc.start.Unix(), period,
			"%s: the bucket is the instant its period starts in the organization's zone", tc.name)

		cells, formatErr := reportfmt.FormatRow(ctx, []reportfmt.Column{{
			Type:    compiled.Columns[0].Type,
			Display: &compiled.Columns[0].Display,
		}}, []any{period}, chicago)
		require.NoError(t, formatErr, tc.name)
		assert.Equal(t, tc.display, cells[0].Text, tc.name)
	}
}
