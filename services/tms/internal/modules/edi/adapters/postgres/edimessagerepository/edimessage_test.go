package edimessagerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestLifecycleMessageQuery_MatchesTheLifecycleIdentity(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, lifecycleMessageQuery(
		&ediports.GetServiceFailure214LifecycleMessageRequest{
			ServiceFailureID: pulid.ID("sf_1"),
			Trigger:          "Reviewed",
		},
	).Limit(1))

	assert.Contains(t, sql, `FROM "edi_messages" em LEFT JOIN "edi_partners" p ON p."id" = em."edi_partner_id"`)
	assert.Contains(t, sql, `em."transaction_set" = '214' AND em."direction" = 'Outbound'`)
	assert.Contains(
		t,
		sql,
		`AND (em."payload_snapshot" #>> $3) = $4 AND (em."payload_snapshot" #>> $5) = $6 `+
			`AND (em."payload_snapshot" #>> $7) = $8`,
	)
	assert.Contains(t, sql, `ORDER BY em."generated_at" DESC LIMIT 1`)
	assert.Contains(t, args, []string{"shipmentStatus", "serviceFailureId"})
	assert.Contains(t, args, []string{"shipmentStatus", "references", "serviceFailure214Trigger"})
	assert.Contains(t, args, "sf_1")
	assert.Contains(t, args, "Reviewed")
}

func TestServiceFailure214StatusQuery_ReadsTheLatestLifecycleMessage(t *testing.T) {
	t.Parallel()

	keeltest.Snapshot(t, serviceFailure214StatusQuery("sf_1"))
}

func TestScorecardQuery_GroupsEveryCountByPartner(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, scorecardQuery(&ediports.GetEDIPartnerScorecardsRequest{
		Since:                  100,
		OverdueAckPendingSince: 200,
		PendingOver4hBefore:    300,
		PendingOver24hBefore:   400,
	}))

	assert.Contains(t, sql, `SELECT em."edi_partner_id", max(p."name"), max(p."code"), count(*) FILTER`)
	assert.Contains(t, sql, `CAST(percentile_cont($4) WITHIN GROUP `+
		`(ORDER BY (em."ack_received_at" - em."delivery_sent_at") ASC) FILTER (WHERE `)
	assert.Equal(t, 0.95, args[3])
	assert.Contains(t, sql, `em."edi_partner_id" IS NOT NULL AND em."generated_at" >= $3`)
	assert.Contains(t, sql, `GROUP BY em."edi_partner_id" ORDER BY max(p."name") ASC`)
	assert.Contains(t, sql, `count(*) FILTER (WHERE (em."ack_status" = 'Pending' AND em."generated_at" < $5))`)
	assert.Contains(
		t,
		sql,
		`count(*) FILTER (WHERE ((em."direction" = 'Outbound' `+
			`AND em."delivery_status" IN ('Queued', 'Sending', 'Failed')) AND em."generated_at" < $6))`,
	)
	assert.Equal(t, []any{"bu_1", "org_1", int64(100), 0.95, int64(200), int64(300), int64(400)}, args)
}

func TestVolumeQuery_GroupsByTheBucketItSelects(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, volumeQuery(&ediports.GetEDIVolumeSeriesRequest{Since: 100}))

	assert.Contains(
		t,
		sql,
		`SELECT ((em."generated_at" / $4::bigint) * $4::bigint), `+
			`count(*) FILTER (WHERE em."direction" = 'Outbound')`,
	)
	assert.Contains(t, sql, `WHERE em."business_unit_id" = $1 AND em."organization_id" = $2 AND em."generated_at" >= $3`)
	assert.Contains(
		t,
		sql,
		`GROUP BY ((em."generated_at" / $4::bigint) * $4::bigint) `+
			`ORDER BY ((em."generated_at" / $4::bigint) * $4::bigint) ASC`,
	)
	assert.Contains(t, sql, `em."delivery_status" IN ('Failed', 'DeadLettered')`)
	assert.Contains(t, sql, `count(*) FILTER (WHERE em."direction" = 'Inbound')`)
	assert.Equal(t, []any{"bu_1", "org_1", int64(100), int64(3600)}, args)
}
