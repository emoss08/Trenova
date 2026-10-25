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

	assert.Contains(t, sql, `ORDER BY CASE i."severity" WHEN 'Critical' THEN 3`, sql)
	assert.Contains(t, sql, `ELSE 0 END DESC, i."detected_at" DESC LIMIT 50`, sql)
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
