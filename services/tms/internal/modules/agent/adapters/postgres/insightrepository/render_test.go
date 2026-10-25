package insightrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestActiveInsightsReadTheMostUrgentFirst(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, activeQuery(&insightports.ListActiveInsightsRequest{
		AllowedDetectorKeys: insightports.AllowedDetectorKeys{"ontime"},
		Limit:               500,
	}))

	assert.Contains(t, sql, `ORDER BY CASE WHEN i."severity" = $5 THEN $6::bigint`, sql)
	assert.Contains(t, sql, `ELSE $11::bigint END DESC, i."detected_at" DESC LIMIT 50`, sql)
	assert.Equal(t, []any{
		string(insight.SeverityCritical), int64(3),
		string(insight.SeverityWarning), int64(2),
		string(insight.SeverityInfo), int64(1),
		int64(0),
	}, args[4:11])
	assert.Contains(t, sql, `i."detector_key" = ANY($`, sql)
	assert.NotContains(t, sql, `"category"`+" = ANY", sql)
	assert.Contains(t, args, string(insight.StatusActive))
}

func TestListDefaultsToWhatNeedsAttentionNow(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, listFilter(&insightports.ListInsightsQuery{
		AllowedDetectorKeys: insightports.AllowedDetectorKeys{"ontime"},
	}))
	assert.Contains(t, sql, `i."status" = $`, sql)
	assert.Contains(t, args, string(insight.StatusActive))

	sql, _ = keeltest.Render(t, listFilter(&insightports.ListInsightsQuery{
		AllowedDetectorKeys: insightports.AllowedDetectorKeys{"ontime"},
		Statuses:            []insight.Status{insight.StatusDismissed, insight.StatusResolved},
	}))
	assert.Contains(t, sql, `i."status" = ANY($`, sql)
}

func TestCloseActiveSupersedesNothingWhenARunFoundNothing(t *testing.T) {
	t.Parallel()

	assert.Nil(t, closeActiveQuery(closeParams{detectorKey: "ontime", within: true}))

	resolvesAll, _ := keeltest.Render(t, closeActiveQuery(closeParams{detectorKey: "ontime"}))
	assert.NotContains(t, resolvesAll, `"dedupe_key" <> ALL`, resolvesAll)

	resolvesMissing, _ := keeltest.Render(t, closeActiveQuery(closeParams{
		detectorKey: "ontime",
		keys:        []string{"ontime:cus_1"},
	}))
	assert.Contains(t, resolvesMissing, `i."dedupe_key" <> ALL($`, resolvesMissing)

	supersedes, _ := keeltest.Render(t, closeActiveQuery(closeParams{
		detectorKey: "ontime",
		keys:        []string{"ontime:cus_1"},
		within:      true,
	}))
	assert.Contains(t, supersedes, `i."dedupe_key" = ANY($`, supersedes)
}

func TestMetricQueriesRender(t *testing.T) {
	t.Parallel()

	window := insightports.InsightWindowRequest{WindowStart: 1_790_000_000, WindowEnd: 1_792_592_000}

	t.Run("customer on-time", func(t *testing.T) {
		t.Parallel()
		keeltest.Snapshot(t, customerOnTimeQuery(window))
	})
	t.Run("unbilled shipments", func(t *testing.T) {
		t.Parallel()
		keeltest.Snapshot(t, unbilledShipmentsQuery(insightports.UnbilledShipmentsRequest{
			DeliveredBefore: 1_790_000_000,
		}))
	})
	t.Run("unbilled detention", func(t *testing.T) {
		t.Parallel()
		keeltest.Snapshot(t, unbilledDetentionQuery(window))
	})
	t.Run("customer empty miles", func(t *testing.T) {
		t.Parallel()
		keeltest.Snapshot(t, customerEmptyMilesQuery(window))
	})
	t.Run("expiring credentials", func(t *testing.T) {
		t.Parallel()
		keeltest.Snapshot(t, expiringCredentialsQuery(insightports.ExpiringCredentialsRequest{
			From:    1_790_000_000,
			Through: 1_792_592_000,
		}))
	})
}
