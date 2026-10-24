package triageinsightrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func window() triageports.BriefingWindow {
	return triageports.BriefingWindow{
		DayStart: 1_760_245_200,
		Now:      1_760_290_000,
		Limit:    200,
	}
}

func assertTenant(t *testing.T, sqlText string) {
	t.Helper()

	assert.Contains(t, sqlText, `im."business_unit_id" = $`, sqlText)
	assert.Contains(t, sqlText, `im."organization_id" = $`, sqlText)
}

func TestBriefingCountsIsOneRoundTripOfIndexedCounts(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, briefingCountsQuery(window()))

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, `im."due_at" < $9`, sqlText)
	assert.Contains(t, sqlText, `im."reviewed_by" IS NOT NULL`, sqlText)
	assert.Contains(t, sqlText, `im."handled_by_rule_id" IS NOT NULL`, sqlText)
	assert.Contains(t, sqlText, `im."parked" IS NOT NULL`, sqlText)
	assert.Contains(t, sqlText, `EXISTS (SELECT 1 FROM "inbound_messages" any_message`, sqlText)
	assert.Equal(t, int64(1_760_290_000), args[8])
}

func TestHandledKeepsThreeSamplesPerGroup(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, handledQuery(window()))

	assertTenant(t, sqlText)
	assert.Contains(
		t,
		sqlText,
		`PARTITION BY im."handled_action", im."classification" ORDER BY im."received_at" DESC, im."id" DESC`,
		sqlText,
	)
	assert.Contains(t, sqlText, `WHERE r."sample_rank" <= $1`, sqlText)
	assert.Contains(t, sqlText, `COALESCE(s."pro_number", '')`, sqlText)
	assert.NotContains(t, sqlText, `r.*`, sqlText)
	assert.Equal(t, []any{3, "bu_1", "org_1", int64(1_760_245_200)}, args)
}

func TestArrivalsAreBucketedByHourSinceMidnight(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, arrivalsQuery(window()))

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, `FLOOR((im."received_at" - $5) / 3600)::int8`, sqlText)
	assert.Contains(t, sqlText, `im."received_at" <= $4`, sqlText)
	assert.Contains(t, sqlText, `GROUP BY FLOOR((im."received_at" - $5) / 3600)::int8`, sqlText)
	assert.Equal(t, int64(1_760_290_000), args[3])
	assert.Equal(t, int64(1_760_245_200), args[4])
}

func TestRiverBoundsEachLane(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, riverQuery(window()))

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, `WHERE r."lane_rank" <= $1`, sqlText)
	assert.Contains(t, sqlText, `r."nova" FROM "ranked" r`, sqlText)
	assert.Equal(t, 200, args[0])
}

func TestWaitingIsSoonestDueFirstAndBounded(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, waitingQuery(triageports.WaitingRequest{Limit: 1000}))

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, `im."status" = ANY($3) AND im."parked" IS NULL`, sqlText)
	assert.Contains(
		t,
		sqlText,
		`ORDER BY im."due_at" ASC NULLS LAST, im."received_at" ASC, im."id" ASC LIMIT 1000`,
		sqlText,
	)
	assert.NotContains(t, sqlText, `im."text_body"`, sqlText)
	assert.NotContains(t, sqlText, `im."html_key"`, sqlText)
	assert.Equal(t, []any{"bu_1", "org_1", []string{"InReview", "Quarantined"}}, args)
}

func TestCorrectionTotalsCompareTwoWeeks(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, correctionTotalsQuery(triageports.CorrectionTotalsRequest{
		LastWeekStart: 1_759_100_000,
		WeekStart:     1_759_700_000,
	}))

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, `ima."kind" = ANY($4) AND ima."actor_kind" = $5`, sqlText)
	assert.Contains(t, sqlText, `im."nova_read" IS TRUE AND im."received_at" >= $3`, sqlText)
	assert.Equal(t, int64(1_759_100_000), args[2])
	assert.Equal(t, []string{"Corrected", "FactFlagged"}, args[3])
	assert.Equal(t, "Person", args[4])
}

func TestRuleCandidatesNarrowBySentence(t *testing.T) {
	t.Parallel()

	base := triageports.RuleCandidatesRequest{
		Classification: inboundmessage.ClassificationProofOfDelivery,
		Since:          1_757_700_000,
		Limit:          5000,
	}

	domain := base
	domain.SenderScope = triage.SenderDomain
	domain.Sender = "drivers.example"
	domain.MailboxID = "imbx_1"
	sqlText, args := keeltest.Render(t, ruleCandidatesQuery(&domain))
	assertTenant(t, sqlText)
	assert.Contains(
		t,
		sqlText,
		`im."classification" = $3 AND im."received_at" >= $4 AND im."nova_read" IS TRUE `+
			`AND im."mailbox_id" = $5 AND (im."sender_domain" = $6 OR im."sender_domain" LIKE $7)`,
		sqlText,
	)
	assert.Contains(t, sqlText, `EXISTS (SELECT 1 FROM "inbound_message_attachments" ima`, sqlText)
	assert.Contains(t, sqlText, "LIMIT 5000", sqlText)
	assert.Equal(
		t,
		[]any{"bu_1", "org_1", "ProofOfDelivery", int64(1_757_700_000), "imbx_1", "drivers.example", "%.drivers.example"},
		args,
	)

	address := base
	address.SenderScope = triage.SenderAddress
	address.Sender = "Ops@Acme.example"
	sqlText, args = keeltest.Render(t, messages().Where(ruleCandidates(&address)...))
	assert.Contains(
		t,
		sqlText,
		`im."sender_domain" = $5 AND LOWER(BTRIM(im."from_address")) = $6`,
		sqlText,
	)
	assert.Equal(t, []any{"acme.example", "ops@acme.example"}, args[4:])

	customer := base
	customer.SenderScope = triage.SenderAnyCustomer
	sqlText, _ = keeltest.Render(t, messages().Where(ruleCandidates(&customer)...))
	assert.Contains(t, sqlText, `im."matched_customer_id" IS NOT NULL`, sqlText)
}

func TestManualOutcomesCountPeopleOnly(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, manualOutcomesQuery(&triageports.ManualOutcomesRequest{
		Since: 1_757_700_000,
	}))

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, `WITH "manual_outcomes" ("message_id", "outcome") AS`, sqlText)
	assert.Contains(t, sqlText, `ima."actor_kind" = $4 AND ima."kind" = ANY($5)`, sqlText)
	assert.Contains(
		t,
		sqlText,
		`GROUP BY im."sender_domain", im."classification", mo."outcome"`,
		sqlText,
	)
	assert.Equal(t, "Person", args[3])
	assert.Equal(t, []string{"Filed", "Handled", "Ignored"}, args[4])
}

func TestSenderDomainsEscapeTheSearch(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, senderDomainsQuery(&triageports.SenderDomainsRequest{
		Query: "Dri_%",
		Since: 1,
		Limit: 20,
	}))

	assertTenant(t, sqlText)
	assert.Contains(
		t,
		sqlText,
		`im."sender_domain" LIKE $5 GROUP BY im."sender_domain" `+
			`ORDER BY count(*) DESC, im."sender_domain" ASC LIMIT 20`,
		sqlText,
	)
	assert.Equal(t, `%dri\_\%%`, args[4])
}

func TestSenderRecentLeavesTheOpenMessageOut(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(t, senderRecentQuery(&triageports.SenderStatsRequest{
		SenderDomain: "acme.example",
		Since:        1,
		ExcludeID:    "imsg_open",
		RecentLimit:  3,
	}))

	assert.Contains(
		t,
		sqlText,
		`im."sender_domain" = $3 AND im."received_at" >= $4 AND im."id" <> $5 `+
			`ORDER BY im."received_at" DESC, im."id" DESC LIMIT 3`,
		sqlText,
	)
	assert.Equal(t, []any{"bu_1", "org_1", "acme.example", int64(1), "imsg_open"}, args)
}
