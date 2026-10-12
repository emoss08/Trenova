package triageinsightrepository

import (
	"database/sql"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

var tenantInfo = pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"}

func renderDB() *bun.DB {
	return bun.NewDB(new(sql.DB), pgdialect.New())
}

func window() triageports.BriefingWindow {
	return triageports.BriefingWindow{
		TenantInfo: tenantInfo,
		DayStart:   1_760_245_200,
		Now:        1_760_290_000,
		Limit:      200,
	}
}

func assertTenant(t *testing.T, sqlText string) {
	t.Helper()

	assert.Contains(t, sqlText, "organization_id = 'org_1'", sqlText)
	assert.Contains(t, sqlText, "business_unit_id = 'bu_1'", sqlText)
}

func TestBriefingCountsIsOneRoundTripOfIndexedCounts(t *testing.T) {
	t.Parallel()

	sqlText := briefingCountsQuery(renderDB(), window()).String()

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, "AS waiting", sqlText)
	assert.Contains(t, sqlText, "imsg.due_at < 1760290000", sqlText)
	assert.Contains(t, sqlText, "imsg.reviewed_by IS NOT NULL", sqlText)
	assert.Contains(t, sqlText, "imsg.handled_by_rule_id IS NOT NULL", sqlText)
	assert.Contains(t, sqlText, "imsg.parked IS NOT NULL", sqlText)
	assert.Contains(t, sqlText, "EXISTS (SELECT 1", sqlText)
}

func TestHandledKeepsThreeSamplesPerGroup(t *testing.T) {
	t.Parallel()

	sqlText := handledQuery(renderDB(), window()).String()

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, "PARTITION BY imsg.handled_action, imsg.classification", sqlText)
	assert.Contains(t, sqlText, "ranked.sample_rank <= 3", sqlText)
	assert.Contains(t, sqlText, "COALESCE(sp.pro_number, '') AS reference", sqlText)
	assert.NotContains(t, sqlText, "ranked.*", sqlText)
}

func TestArrivalsAreBucketedByHourSinceMidnight(t *testing.T) {
	t.Parallel()

	sqlText := arrivalsQuery(renderDB(), window()).String()

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, "FLOOR((imsg.received_at - 1760245200) / 3600)::int AS hour", sqlText)
	assert.Contains(t, sqlText, "imsg.received_at <= 1760290000", sqlText)
	assert.Contains(t, sqlText, "GROUP BY 1", sqlText)
}

func TestRiverBoundsEachLane(t *testing.T) {
	t.Parallel()

	sqlText := riverQuery(renderDB(), window()).String()

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, "ranked.lane_rank <= 200", sqlText)
	assert.Contains(t, sqlText, "AS nova", sqlText)
}

func TestWaitingIsSoonestDueFirstAndBounded(t *testing.T) {
	t.Parallel()

	entities := make([]*inboundmessage.InboundMessage, 0)
	sqlText := waitingQuery(renderDB(), triageports.WaitingRequest{
		TenantInfo: tenantInfo,
		Limit:      1000,
	}, &entities).String()

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, "imsg.status IN ('InReview', 'Quarantined')", sqlText)
	assert.Contains(t, sqlText, "imsg.parked IS NULL", sqlText)
	assert.Contains(t, sqlText, "ORDER BY imsg.due_at ASC NULLS LAST", sqlText)
	assert.Contains(t, sqlText, "LIMIT 1000", sqlText)
	assert.NotContains(t, sqlText, `"imsg"."text_body"`, sqlText)
}

func TestCorrectionTotalsCompareTwoWeeks(t *testing.T) {
	t.Parallel()

	sqlText := correctionTotalsQuery(renderDB(), triageports.CorrectionTotalsRequest{
		TenantInfo:    tenantInfo,
		LastWeekStart: 1_759_100_000,
		WeekStart:     1_759_700_000,
	}).String()

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, "iact.actor_kind = 'Person'", sqlText)
	assert.Contains(t, sqlText, "iact.kind IN ('Corrected', 'FactFlagged')", sqlText)
	assert.Contains(t, sqlText, "imsg.nova_read = TRUE", sqlText)
	assert.Contains(t, sqlText, "imsg.received_at >= 1759100000", sqlText)
}

func TestRuleCandidatesNarrowBySentence(t *testing.T) {
	t.Parallel()

	base := &triageports.RuleCandidatesRequest{
		TenantInfo:     tenantInfo,
		Classification: inboundmessage.ClassificationProofOfDelivery,
		Since:          1_757_700_000,
		Limit:          5000,
	}

	domain := base
	domain.SenderScope = triage.SenderDomain
	domain.Sender = "drivers.example"
	domain.MailboxID = "imbx_1"
	sqlText := ruleCandidatesQuery(renderDB(), domain).String()
	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, "imsg.classification = 'ProofOfDelivery'", sqlText)
	assert.Contains(t, sqlText, "imsg.mailbox_id = 'imbx_1'", sqlText)
	assert.Contains(t, sqlText, "imsg.sender_domain = 'drivers.example'", sqlText)
	assert.Contains(t, sqlText, `imsg.sender_domain LIKE '%.drivers.example'`, sqlText)
	assert.Contains(t, sqlText, "AS person_filed", sqlText)
	assert.Contains(t, sqlText, "LIMIT 5000", sqlText)

	address := base
	address.SenderScope = triage.SenderAddress
	address.Sender = "Ops@Acme.example"
	sqlText = ruleCandidatesCountQuery(renderDB(), address).String()
	assert.Contains(t, sqlText, "imsg.sender_domain = 'acme.example'", sqlText)
	assert.Contains(t, sqlText, "LOWER(BTRIM(imsg.from_address)) = 'ops@acme.example'", sqlText)

	customer := base
	customer.SenderScope = triage.SenderAnyCustomer
	sqlText = ruleCandidatesCountQuery(renderDB(), customer).String()
	assert.Contains(t, sqlText, "imsg.matched_customer_id IS NOT NULL", sqlText)
}

func TestManualOutcomesCountPeopleOnly(t *testing.T) {
	t.Parallel()

	sqlText := manualOutcomesQuery(renderDB(), &triageports.ManualOutcomesRequest{
		TenantInfo: tenantInfo,
		Since:      1_757_700_000,
	}).String()

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, `WITH "manual_outcomes" AS`, sqlText)
	assert.Contains(t, sqlText, "iact.actor_kind = 'Person'", sqlText)
	assert.Contains(t, sqlText, "iact.kind IN ('Filed', 'Handled', 'Ignored')", sqlText)
	assert.Contains(t, sqlText, "GROUP BY imsg.sender_domain, imsg.classification, mo.outcome", sqlText)
}

func TestSenderDomainsEscapeTheSearch(t *testing.T) {
	t.Parallel()

	sqlText := senderDomainsQuery(renderDB(), &triageports.SenderDomainsRequest{
		TenantInfo: tenantInfo,
		Query:      "Dri_%",
		Since:      1,
		Limit:      20,
	}).String()

	assertTenant(t, sqlText)
	assert.Contains(t, sqlText, `imsg.sender_domain LIKE '%dri\_\%%'`, sqlText)
	assert.Contains(t, sqlText, "LIMIT 20", sqlText)
}

func TestSenderRecentLeavesTheOpenMessageOut(t *testing.T) {
	t.Parallel()

	sqlText := senderRecentQuery(renderDB(), &triageports.SenderStatsRequest{
		TenantInfo:   tenantInfo,
		SenderDomain: "acme.example",
		Since:        1,
		ExcludeID:    "imsg_open",
		RecentLimit:  3,
	}).String()

	assert.Contains(t, sqlText, "imsg.id <> 'imsg_open'", sqlText)
	assert.Contains(t, sqlText, "imsg.sender_domain = 'acme.example'", sqlText)
	assert.Contains(t, sqlText, "LIMIT 3", sqlText)
}
