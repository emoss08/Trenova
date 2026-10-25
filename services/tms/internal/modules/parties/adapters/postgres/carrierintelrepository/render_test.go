package carrierintelrepository

import (
	"testing"

	"github.com/emoss08/keel"

	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel/carrierinteldb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestCarrierSummaryUpdateOnlyTouchesChangedCarriers(t *testing.T) {
	t.Parallel()

	q, sets := carrierSummaryUpdate(&carrierintelports.UpdateCarrierIntelSummaryRequest{
		CarrierID:      pulid.MustNew("car_"),
		RiskLevel:      string(carrierintel.RiskLevelHigh),
		ReviewRequired: true,
		BlockingCount:  2,
	})
	sql, args := keeltest.Render(t, q)

	assert.Contains(t, sql, `"intel_risk_level" IS DISTINCT FROM`, sql)
	assert.Contains(t, sql, `"intel_review_required" IS DISTINCT FROM`, sql)
	assert.Contains(t, sql, `"intel_blocking_count" IS DISTINCT FROM`, sql)
	assert.Contains(t, args, string(carrierintel.RiskLevelHigh))
	assert.Contains(t, args, true)
	assert.Len(t, sets, 5)
}

func TestPruneHistoryDeletesOnlyRankedNonCurrentSnapshots(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, pruneHistoryQuery(12))

	assert.Contains(t, sql, `WITH "ranked"`, sql)
	assert.Contains(
		t,
		sql,
		`row_number() OVER (PARTITION BY history."subject_type", history."subject_id"`,
		sql,
	)
	assert.Contains(t, sql, `"is_current" IS FALSE`, sql)
	assert.Contains(t, args, 12)
}

func TestEventFiltersBoundTheDetectedWindowOnlyWhenAsked(t *testing.T) {
	t.Parallel()

	e := carrierinteldb.CarrierIntelEvents
	bounded, args := keeltest.Render(
		t,
		e.Where(eventFilters(&carrierintelports.ListCarrierIntelEventsRequest{
			DetectedFrom: 1_790_000_000,
			DetectedTo:   1_790_604_799,
		})...),
	)

	assert.Contains(t, bounded, `"detected_at" >=`, bounded)
	assert.Contains(t, bounded, `"detected_at" <=`, bounded)
	assert.Contains(t, args, int64(1_790_000_000))
	assert.Contains(t, args, int64(1_790_604_799))

	open, _ := keeltest.Render(
		t,
		e.Where(eventFilters(&carrierintelports.ListCarrierIntelEventsRequest{})...),
	)

	assert.NotContains(t, open, `"detected_at" >=`, open)
	assert.NotContains(t, open, `"detected_at" <=`, open)
}

func TestEventFiltersCombineStatusesAndSubject(t *testing.T) {
	t.Parallel()

	e := carrierinteldb.CarrierIntelEvents
	sql, args := keeltest.Render(
		t,
		e.Where(eventFilters(&carrierintelports.ListCarrierIntelEventsRequest{
			OpenOnly:    true,
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   "car_1",
		})...),
	)

	assert.Contains(t, sql, `"status" IN ('Open', 'Acknowledged')`, sql)
	assert.Contains(t, sql, `"subject_type" = 'Carrier'`, sql)
	assert.Contains(t, sql, `"subject_id" =`, sql)
	assert.Contains(t, args, "car_1")
}

func TestDigestOrdersBySeverityRank(t *testing.T) {
	t.Parallel()

	e := carrierinteldb.CarrierIntelEvents
	sql, args := keeltest.Render(t, e.Query().Order(keel.Asc(eventSeverityRank())).Limit(1))

	assert.Contains(t, sql, "CASE", sql)
	for _, severity := range eventSeverityOrder {
		assert.Contains(t, args, string(severity))
	}
}

func TestPendingSyncCoversEnrolAndUnenrol(t *testing.T) {
	t.Parallel()

	en := carrierinteldb.CarrierMonitoringEnrollments
	sql, args := keeltest.Render(t, en.Where(pendingSyncPredicate()))

	assert.Contains(t, sql, " OR ", sql)
	assert.Contains(t, sql, `cme."desired_state" = 'Enrolled'`, sql)
	assert.Contains(t, sql, `cme."desired_state" = 'NotEnrolled'`, sql)
	assert.Equal(t, []any{"bu_1", "org_1"}, args)
}
